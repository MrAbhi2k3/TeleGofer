package connection

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mrabhi2k3/telegofer/mtproto/crypto"
	"github.com/mrabhi2k3/telegofer/mtproto/protocol"
	"github.com/mrabhi2k3/telegofer/mtproto/rpc"
	"github.com/mrabhi2k3/telegofer/mtproto/transport"
	"github.com/mrabhi2k3/telegofer/tl"
	"github.com/mrabhi2k3/telegofer/tl/encoder"
)

// Connection represents an authenticated, encrypted MTProto 2.0 connection
// to a Telegram Data Center.
type Connection struct {
	transport *transport.Conn
	authKey   *crypto.AuthKey

	sessionID  int64
	serverSalt int64
	saltMu     sync.RWMutex

	msgIDs *protocol.MessageIDGenerator
	seqNos *protocol.SequenceGenerator
	rpc    *rpc.Engine

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewConnection initializes an active encrypted connection wrapping a transport.Conn.
func NewConnection(tr *transport.Conn, authKey *crypto.AuthKey, initialSalt int64) (*Connection, error) {
	if authKey == nil {
		return nil, errors.New("connection: authKey must not be nil")
	}

	// Generate random 64-bit session ID
	var sessBytes [8]byte
	if _, err := rand.Read(sessBytes[:]); err != nil {
		return nil, fmt.Errorf("connection: failed to generate session ID: %w", err)
	}
	sessionID := int64(binary.LittleEndian.Uint64(sessBytes[:]))

	c := &Connection{
		transport:  tr,
		authKey:    authKey,
		sessionID:  sessionID,
		serverSalt: initialSalt,
		msgIDs:     protocol.NewMessageIDGenerator(),
		seqNos:     protocol.NewSequenceGenerator(),
		rpc:        rpc.NewEngine(),
		closed:     make(chan struct{}),
	}

	// Start background reader and keepalive loops
	c.wg.Add(2)
	go c.readLoop()
	go c.keepaliveLoop()

	return c, nil
}

// Invoke serializes an RPC request, assigns it message ID and sequence number,
// encrypts it, transmits it to the server, and awaits the correlated response.
func (c *Connection) Invoke(ctx context.Context, req tl.Object, timeout time.Duration) ([]byte, error) {
	select {
	case <-c.closed:
		return nil, transport.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Encode the request
	enc := encoder.New()
	if err := req.Encode(enc); err != nil {
		return nil, fmt.Errorf("connection: failed to encode request: %w", err)
	}

	msgID := c.msgIDs.Next()
	seqNo := c.seqNos.Next(true)

	// Register with RPC engine before transmitting
	ch := c.rpc.Register(msgID)

	c.saltMu.RLock()
	salt := c.serverSalt
	c.saltMu.RUnlock()

	encrypted, err := protocol.PackEncrypted(c.authKey, salt, c.sessionID, msgID, seqNo, enc.Bytes())
	if err != nil {
		c.rpc.Cancel(msgID)
		return nil, fmt.Errorf("connection: packet encryption failed: %w", err)
	}

	if err := c.transport.Send(ctx, encrypted); err != nil {
		c.rpc.Cancel(msgID)
		return nil, fmt.Errorf("connection: send failed: %w", err)
	}

	return c.rpc.Await(ctx, msgID, ch, timeout)
}

func (c *Connection) readLoop() {
	defer c.wg.Done()

	var buf [65536]byte // Reusable stack buffer for incoming packets

	for {
		select {
		case <-c.closed:
			return
		default:
		}

		rawPacket, err := c.transport.Recv(context.Background(), buf[:])
		if err != nil {
			select {
			case <-c.closed:
			default:
				c.Close()
			}
			return
		}

		salt, msgID, _, payload, err := protocol.UnpackEncrypted(c.authKey, c.sessionID, rawPacket)
		if err != nil {
			// Malformed or corrupted packet, skip
			continue
		}

		if salt != 0 {
			c.saltMu.Lock()
			c.serverSalt = salt
			c.saltMu.Unlock()
		}

		// Dispatch through RPC engine
		_ = c.rpc.Dispatch(msgID, payload)
	}
}

func (c *Connection) keepaliveLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.closed:
			return
		case <-ticker.C:
			c.flushAcksAndPing()
		}
	}
}

func (c *Connection) flushAcksAndPing() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Flush queued acknowledgements
	acks := c.rpc.PopAcks()
	if len(acks) > 0 {
		ackPayload := rpc.PackMsgsAck(acks)
		msgID := c.msgIDs.Next()
		seqNo := c.seqNos.Next(false) // Service message, no ACK needed

		c.saltMu.RLock()
		salt := c.serverSalt
		c.saltMu.RUnlock()

		if packet, err := protocol.PackEncrypted(c.authKey, salt, c.sessionID, msgID, seqNo, ackPayload); err == nil {
			_ = c.transport.Send(ctx, packet)
		}
	}

	// 2. Send keepalive ping (ping#7abe77ec ping_id:long = Pong)
	var pingBuf [12]byte
	binary.LittleEndian.PutUint32(pingBuf[0:4], 0x7abe77ec)
	binary.LittleEndian.PutUint64(pingBuf[4:12], uint64(time.Now().UnixNano()))

	msgID := c.msgIDs.Next()
	seqNo := c.seqNos.Next(false)

	c.saltMu.RLock()
	salt := c.serverSalt
	c.saltMu.RUnlock()

	if packet, err := protocol.PackEncrypted(c.authKey, salt, c.sessionID, msgID, seqNo, pingBuf[:]); err == nil {
		_ = c.transport.Send(ctx, packet)
	}
}

// Close terminates the connection, background workers, and underlying transport.
func (c *Connection) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		c.rpc.Close()
		err = c.transport.Close()
	})
	c.wg.Wait()
	return err
}
