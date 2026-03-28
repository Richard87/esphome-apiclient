package esphome_apiclient

import (
	"github.com/richard87/esphome-apiclient/codec"
	"github.com/richard87/esphome-apiclient/transport"
)

// Framer reads and writes ESPHome message frames. Two implementations exist:
//   - PlainFramer for unencrypted connections (0x00 preamble, varint framing)
//   - NoiseFramer for Noise-encrypted connections (0x01 preamble, encrypted framing)
type Framer interface {
	// WriteFrame encodes and sends a message frame.
	WriteFrame(msgType uint32, data []byte) error
	// ReadFrame receives and decodes a message frame.
	ReadFrame() (msgType uint32, data []byte, err error)
	// Close closes the underlying connection.
	Close() error
}

// PlainFramer wraps a plain transport with the standard ESPHome codec.
type PlainFramer struct {
	transport transport.Transport
}

// NewPlainFramer creates a PlainFramer from a transport.
func NewPlainFramer(t transport.Transport) *PlainFramer {
	return &PlainFramer{transport: t}
}

// WriteFrame writes an unencrypted ESPHome frame (0x00 + varint size + varint type + data).
func (f *PlainFramer) WriteFrame(msgType uint32, data []byte) error {
	return codec.WriteFrame(f.transport, msgType, data)
}

// ReadFrame reads an unencrypted ESPHome frame.
func (f *PlainFramer) ReadFrame() (uint32, []byte, error) {
	return codec.ReadFrame(f.transport)
}

// Close closes the underlying transport.
func (f *PlainFramer) Close() error {
	return f.transport.Close()
}

// NoiseFramer wraps a NoiseTransport to implement the Framer interface.
// The NoiseTransport already handles encryption and the noise frame format.
type NoiseFramer struct {
	transport *transport.NoiseTransport
}

// NewNoiseFramer creates a NoiseFramer from a NoiseTransport.
func NewNoiseFramer(nt *transport.NoiseTransport) *NoiseFramer {
	return &NoiseFramer{transport: nt}
}

// WriteFrame encrypts and sends a protobuf message using noise framing.
func (f *NoiseFramer) WriteFrame(msgType uint32, data []byte) error {
	return f.transport.WriteFrame(msgType, data)
}

// ReadFrame reads and decrypts a noise frame, returning the message type and payload.
func (f *NoiseFramer) ReadFrame() (uint32, []byte, error) {
	return f.transport.ReadFrame()
}

// Close closes the underlying noise transport.
func (f *NoiseFramer) Close() error {
	return f.transport.Close()
}
