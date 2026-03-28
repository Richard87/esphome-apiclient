package transport

import (
	"fmt"
	"net"
	"time"
)

// Transport reads and writes raw bytes over the connection.
type Transport interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// PlainTransport wraps a bare net.Conn.
type PlainTransport struct {
	conn net.Conn
}

// Dial connects to the specified address with a timeout.
func Dial(address string, timeout time.Duration) (*PlainTransport, error) {
	conn, err := dialTCP(address, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &PlainTransport{
		conn: conn,
	}, nil
}

// Read reads raw bytes over the connection.
func (p *PlainTransport) Read(b []byte) (n int, err error) {
	return p.conn.Read(b)
}

// Write writes raw bytes over the connection.
func (p *PlainTransport) Write(b []byte) (n int, err error) {
	return p.conn.Write(b)
}

// Close closes the connection.
func (p *PlainTransport) Close() error {
	return p.conn.Close()
}

// SetDeadline sets the read and write deadlines associated with the connection.
func (p *PlainTransport) SetDeadline(t time.Time) error {
	return p.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.
func (p *PlainTransport) SetReadDeadline(t time.Time) error {
	return p.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.
func (p *PlainTransport) SetWriteDeadline(t time.Time) error {
	return p.conn.SetWriteDeadline(t)
}
