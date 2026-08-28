package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mrabhi2k3/telegofer/tl/decoder"
)

// Combinator IDs for MTProto service messages.
const (
	CRCGzipPacked       uint32 = 0x3072cfa1
	CRCRPCResult        uint32 = 0xf35c6d01
	CRCRPCError         uint32 = 0x2144ca19
	CRCMsgContainer     uint32 = 0x73f1f8dc
	CRCPong             uint32 = 0x347773c5
	CRCBadMsgNotice     uint32 = 0xa7eff811
	CRCBadServerSalt    uint32 = 0xedab447b
	CRCNewSessionCreate uint32 = 0x9ec20908
)

type pendingCall struct {
	done chan rpcResult
}

type rpcResult struct {
	payload []byte
	err     error
}

// Engine manages asynchronous, concurrent Telegram RPC request/response correlation,
// message acknowledgements, container unpacking, and error classification.
type Engine struct {
	mu      sync.Mutex
	pending map[int64]*pendingCall
	acksMu  sync.Mutex
	ackIDs  []int64
	closed  bool
}

// NewEngine creates an initialized RPC engine.
func NewEngine() *Engine {
	return &Engine{
		pending: make(map[int64]*pendingCall),
	}
}

// Register registers an outgoing request message ID to await its RPC response.
func (e *Engine) Register(msgID int64) chan rpcResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan rpcResult, 1)
	if e.closed {
		ch <- rpcResult{err: errors.New("rpc: engine is closed")}
		return ch
	}

	e.pending[msgID] = &pendingCall{done: ch}
	return ch
}

// Cancel unregisters a pending request (e.g. on context cancellation).
func (e *Engine) Cancel(msgID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.pending, msgID)
}

// Dispatch processes an incoming MTProto message payload, unpacking containers,
// handling gzip decompression, and routing RPC results to their waiting callers.
func (e *Engine) Dispatch(msgID int64, payload []byte) error {
	// Queue ACK for incoming server message ID
	e.acksMu.Lock()
	e.ackIDs = append(e.ackIDs, msgID)
	e.acksMu.Unlock()

	d := decoder.New(payload)
	crc, err := d.PeekUint()
	if err != nil {
		return err
	}

	switch crc {
	case CRCMsgContainer:
		return e.handleContainer(d)
	case CRCGzipPacked:
		decompressed, err := decompressGzip(d)
		if err != nil {
			return err
		}
		return e.Dispatch(msgID, decompressed)
	case CRCRPCResult:
		return e.handleRPCResult(d)
	case CRCPong:
		// Handled via RPC or keepalive
		return nil
	default:
		// Unsolicited updates or service notices
		return nil
	}
}

func (e *Engine) handleContainer(d *decoder.Decoder) error {
	_, _ = d.Uint() // consume CRCMsgContainer
	count, err := d.Int()
	if err != nil {
		return err
	}

	for i := int32(0); i < count; i++ {
		innerMsgID, err := d.Long()
		if err != nil {
			return err
		}
		_, err = d.Int() // seq_no
		if err != nil {
			return err
		}
		bytesLen, err := d.Int()
		if err != nil {
			return err
		}
		if bytesLen < 0 || int(bytesLen) > d.Remaining() {
			return errors.New("rpc: invalid container message length")
		}
		rawMsg, err := d.Raw(int(bytesLen))
		if err != nil {
			return err
		}

		if err := e.Dispatch(innerMsgID, rawMsg); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) handleRPCResult(d *decoder.Decoder) error {
	_, _ = d.Uint() // consume CRCRPCResult
	reqMsgID, err := d.Long()
	if err != nil {
		return err
	}

	innerCRC, err := d.PeekUint()
	if err != nil {
		return err
	}

	e.mu.Lock()
	call, ok := e.pending[reqMsgID]
	if ok {
		delete(e.pending, reqMsgID)
	}
	e.mu.Unlock()

	if !ok {
		// Response for unknown or timed-out request
		return nil
	}

	// Check if the result is an RPC error
	if innerCRC == CRCRPCError {
		_, _ = d.Uint() // consume CRCRPCError
		code, err := d.Int()
		if err != nil {
			call.done <- rpcResult{err: err}
			return err
		}
		msg, err := d.String()
		if err != nil {
			call.done <- rpcResult{err: err}
			return err
		}
		typedErr := ParseError(int(code), msg)
		call.done <- rpcResult{err: typedErr}
		return nil
	}

	// Check if the result is gzip-packed
	if innerCRC == CRCGzipPacked {
		decompressed, err := decompressGzip(d)
		if err != nil {
			call.done <- rpcResult{err: err}
			return err
		}
		call.done <- rpcResult{payload: decompressed}
		return nil
	}

	// Raw payload response
	call.done <- rpcResult{payload: d.BytesRemaining()}
	return nil
}

func decompressGzip(d *decoder.Decoder) ([]byte, error) {
	_, _ = d.Uint() // consume CRCGzipPacked
	compressed, err := d.Bytes()
	if err != nil {
		return nil, err
	}

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("rpc: gzip decompress error: %w", err)
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

// PopAcks retrieves and clears all pending message IDs awaiting ACK to Telegram.
func (e *Engine) PopAcks() []int64 {
	e.acksMu.Lock()
	defer e.acksMu.Unlock()

	if len(e.ackIDs) == 0 {
		return nil
	}

	out := make([]int64, len(e.ackIDs))
	copy(out, e.ackIDs)
	e.ackIDs = e.ackIDs[:0]
	return out
}

// Await waits for the response to a registered request with context cancellation.
func (e *Engine) Await(ctx context.Context, msgID int64, ch chan rpcResult, timeout time.Duration) ([]byte, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		e.Cancel(msgID)
		return nil, ctx.Err()
	case <-timer.C:
		e.Cancel(msgID)
		return nil, fmt.Errorf("rpc: request %d timed out after %s", msgID, timeout)
	case res := <-ch:
		return res.payload, res.err
	}
}

// Close terminates the engine and fails any outstanding pending requests.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.closed = true
	for id, call := range e.pending {
		call.done <- rpcResult{err: errors.New("rpc: engine closed")}
		delete(e.pending, id)
	}
}

// PackMsgsAck serializes a vector of message IDs into a msgs_ack payload.
func PackMsgsAck(msgIDs []int64) []byte {
	// msgs_ack#62d6b459 msg_ids:Vector<long> = MsgsAck
	out := make([]byte, 4+4+4+len(msgIDs)*8)
	binary.LittleEndian.PutUint32(out[0:4], 0x62d6b459)
	binary.LittleEndian.PutUint32(out[4:8], 0x1cb5c415) // Vector CRC
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(msgIDs)))
	for i, id := range msgIDs {
		binary.LittleEndian.PutUint64(out[12+i*8:20+i*8], uint64(id))
	}
	return out
}
