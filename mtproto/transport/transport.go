package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Common transport errors.
var (
	ErrInvalidLength = errors.New("transport: invalid packet length")
	ErrCRCMismatch   = errors.New("transport: CRC32 checksum mismatch")
	ErrClosed        = errors.New("transport: connection is closed")
)

// Transport defines the framing codec for sending and receiving MTProto packets
// over a streaming connection.
type Transport interface {
	// Handshake writes the transport-specific protocol tag to the connection.
	Handshake(w io.Writer) error

	// WritePacket frames and writes an MTProto payload to the writer.
	WritePacket(w io.Writer, payload []byte) error

	// ReadPacket reads a single framed MTProto payload from the reader.
	// If buf has sufficient capacity, it is reused to avoid allocations.
	ReadPacket(r io.Reader, buf []byte) ([]byte, error)
}

// Conn represents an active MTProto network transport connection.
// It wraps a net.Conn with a transport framing codec, timeout management,
// and synchronized I/O.
type Conn struct {
	raw       net.Conn
	codec     Transport
	readMu    sync.Mutex
	writeMu   sync.Mutex
	closed    chan struct{}
	closeOnce sync.Once

	readTimeout  time.Duration
	writeTimeout time.Duration
}

// WrapConn wraps an existing network connection with the specified transport codec.
// It executes the initial transport handshake on the connection.
func WrapConn(raw net.Conn, codec Transport) (*Conn, error) {
	if err := codec.Handshake(raw); err != nil {
		raw.Close()
		return nil, fmt.Errorf("transport handshake failed: %w", err)
	}

	return &Conn{
		raw:          raw,
		codec:        codec,
		closed:       make(chan struct{}),
		readTimeout:  60 * time.Second,
		writeTimeout: 30 * time.Second,
	}, nil
}

// Dial connects to a remote Telegram server address using TCP and initializes
// the given transport codec.
func Dial(ctx context.Context, network, addr string, codec Transport) (*Conn, error) {
	d := net.Dialer{}
	raw, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	conn, err := WrapConn(raw, codec)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// SetTimeouts configures read and write deadlines for packets.
func (c *Conn) SetTimeouts(read, write time.Duration) {
	c.readTimeout = read
	c.writeTimeout = write
}

// Send frames and transmits a packet to the remote endpoint.
func (c *Conn) Send(ctx context.Context, payload []byte) error {
	select {
	case <-c.closed:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.writeTimeout > 0 {
		c.raw.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	} else {
		c.raw.SetWriteDeadline(time.Time{})
	}

	return c.codec.WritePacket(c.raw, payload)
}

// Recv reads the next framed packet from the network. If buf is provided
// and has sufficient capacity, it will be used to hold the payload.
func (c *Conn) Recv(ctx context.Context, buf []byte) ([]byte, error) {
	select {
	case <-c.closed:
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.readMu.Lock()
	defer c.readMu.Unlock()

	if c.readTimeout > 0 {
		c.raw.SetReadDeadline(time.Now().Add(c.readTimeout))
	} else {
		c.raw.SetReadDeadline(time.Time{})
	}

	return c.codec.ReadPacket(c.raw, buf)
}

// Close gracefully closes the transport connection and releases resources.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.raw.Close()
	})
	return err
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.raw.RemoteAddr()
}
