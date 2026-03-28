package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/flynn/noise"
)

const (
	// noisePreamble is the frame preamble byte for noise-encrypted connections.
	noisePreamble byte = 0x01

	// noiseMaxFrameSize is the maximum encrypted frame payload size.
	noiseMaxFrameSize = 65535

	// noisePrologue is the Noise protocol prologue used by ESPHome.
	noisePrologue = "NoiseAPIInit\x00\x00"
)

// NoiseTransport implements encrypted communication with an ESPHome device
// using the Noise_NNpsk0_25519_ChaChaPoly_SHA256 protocol.
//
// After the Noise handshake completes, all reads and writes are transparently
// encrypted/decrypted. The transport uses ESPHome's noise frame format:
//
//	┌──────────┬───────────────────────┬─────────────┐
//	│  0x01    │ Length (big-endian u16)│   Payload   │
//	└──────────┴───────────────────────┴─────────────┘
//
// Inside the encrypted payload, each message is structured as:
//
//	┌──────────────────┬────────────────────┬──────────────┐
//	│ msg_type (BE u16)│ msg_length (BE u16)│ protobuf data│
//	└──────────────────┴────────────────────┴──────────────┘
type NoiseTransport struct {
	conn      net.Conn
	encryptCS *noise.CipherState
	decryptCS *noise.CipherState
	writeMu   sync.Mutex

	// Server info extracted during handshake
	ServerName string
	ServerMAC  string
}

// DialNoise connects to the specified address and performs the Noise handshake.
// The psk must be exactly 32 bytes (decoded from base64).
// expectedName, if non-empty, is validated against the server's reported name.
func DialNoise(address string, timeout time.Duration, psk []byte, expectedName string) (*NoiseTransport, error) {
	if len(psk) != 32 {
		return nil, fmt.Errorf("noise PSK must be 32 bytes, got %d", len(psk))
	}

	conn, err := dialTCP(address, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	nt := &NoiseTransport{conn: conn}

	if err := nt.performHandshake(psk, expectedName, timeout); err != nil {
		conn.Close()
		return nil, fmt.Errorf("noise handshake failed: %w", err)
	}

	return nt, nil
}

// NewNoiseTransport creates a NoiseTransport from an existing net.Conn and
// performs the Noise handshake. Useful for testing with net.Pipe().
func NewNoiseTransport(conn net.Conn, psk []byte, expectedName string, timeout time.Duration) (*NoiseTransport, error) {
	if len(psk) != 32 {
		return nil, fmt.Errorf("noise PSK must be 32 bytes, got %d", len(psk))
	}

	nt := &NoiseTransport{conn: conn}

	if err := nt.performHandshake(psk, expectedName, timeout); err != nil {
		return nil, fmt.Errorf("noise handshake failed: %w", err)
	}

	return nt, nil
}

func (nt *NoiseTransport) performHandshake(psk []byte, expectedName string, timeout time.Duration) error {
	if timeout > 0 {
		if err := nt.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
		defer nt.conn.SetDeadline(time.Time{})
	}

	// Set up the Noise handshake state
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             true,
		Prologue:              []byte(noisePrologue),
		PresharedKey:          psk,
		PresharedKeyPlacement: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to create handshake state: %w", err)
	}

	// Step 1: Send client hello (0x01 0x00 0x00)
	if _, err := nt.conn.Write([]byte{0x01, 0x00, 0x00}); err != nil {
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	// Step 2: Send handshake message
	handshakeMsg, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return fmt.Errorf("failed to generate handshake message: %w", err)
	}

	// Payload: 0x00 (protocol indicator) + handshake message
	payload := make([]byte, 1+len(handshakeMsg))
	payload[0] = 0x00 // protocol indicator
	copy(payload[1:], handshakeMsg)

	if err := nt.writeNoiseFrame(payload); err != nil {
		return fmt.Errorf("failed to send handshake frame: %w", err)
	}

	// Step 3: Read server hello
	serverHello, err := nt.readNoiseFrame()
	if err != nil {
		return fmt.Errorf("failed to read server hello: %w", err)
	}

	if len(serverHello) == 0 {
		return fmt.Errorf("server hello is empty")
	}

	// First byte is the chosen protocol (must be 0x01)
	chosenProto := serverHello[0]
	if chosenProto != 0x01 {
		return fmt.Errorf("server chose unsupported protocol: 0x%02X", chosenProto)
	}

	// Extract server name (null-terminated after protocol byte)
	if nameEnd := bytes.IndexByte(serverHello[1:], 0x00); nameEnd != -1 {
		nt.ServerName = string(serverHello[1 : 1+nameEnd])

		if expectedName != "" && nt.ServerName != expectedName {
			return fmt.Errorf("server name mismatch: expected %q, got %q", expectedName, nt.ServerName)
		}

		// Extract MAC address (null-terminated after server name)
		macStart := 1 + nameEnd + 1
		if macEnd := bytes.IndexByte(serverHello[macStart:], 0x00); macEnd != -1 {
			nt.ServerMAC = string(serverHello[macStart : macStart+macEnd])
		}
	}

	// Step 4: Read server handshake response
	handshakeResp, err := nt.readNoiseFrame()
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %w", err)
	}

	if len(handshakeResp) == 0 {
		return fmt.Errorf("handshake response is empty")
	}

	// First byte 0x00 means success; otherwise it's an error
	if handshakeResp[0] != 0x00 {
		explanation := string(handshakeResp[1:])
		if explanation == "Handshake MAC failure" {
			return fmt.Errorf("invalid encryption key: %s", explanation)
		}
		return fmt.Errorf("handshake error: %s", explanation)
	}

	// Process the handshake response (remaining bytes after the 0x00 indicator)
	_, encryptCS, decryptCS, err := hs.ReadMessage(nil, handshakeResp[1:])
	if err != nil {
		return fmt.Errorf("failed to process handshake response: %w", err)
	}

	nt.encryptCS = encryptCS
	nt.decryptCS = decryptCS

	return nil
}

// WriteFrame encrypts and sends a protobuf message using noise framing.
// The plaintext format is: msg_type (BE u16) + msg_length (BE u16) + protobuf data.
// It is safe to call from multiple goroutines.
func (nt *NoiseTransport) WriteFrame(msgType uint32, data []byte) error {
	// Build inner plaintext: type(2) + length(2) + data
	plaintext := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(plaintext[0:2], uint16(msgType))
	binary.BigEndian.PutUint16(plaintext[2:4], uint16(len(data)))
	copy(plaintext[4:], data)

	// Encrypt
	nt.writeMu.Lock()
	defer nt.writeMu.Unlock()

	ciphertext, err := nt.encryptCS.Encrypt(nil, nil, plaintext)
	if err != nil {
		return fmt.Errorf("noise encrypt failed: %w", err)
	}

	return nt.writeNoiseFrame(ciphertext)
}

// ReadFrame reads and decrypts a noise frame, returning the message type and payload.
func (nt *NoiseTransport) ReadFrame() (msgType uint32, data []byte, err error) {
	ciphertext, err := nt.readNoiseFrame()
	if err != nil {
		return 0, nil, err
	}

	plaintext, err := nt.decryptCS.Decrypt(nil, nil, ciphertext)
	if err != nil {
		return 0, nil, fmt.Errorf("noise decrypt failed: %w", err)
	}

	if len(plaintext) < 4 {
		return 0, nil, fmt.Errorf("decrypted message too short: %d bytes", len(plaintext))
	}

	// Parse inner header
	msgType = uint32(binary.BigEndian.Uint16(plaintext[0:2]))
	// Ignore msg_length field (bytes 2:4) — use actual length like the Python implementation
	data = plaintext[4:]

	return msgType, data, nil
}

// Close closes the underlying TCP connection.
func (nt *NoiseTransport) Close() error {
	return nt.conn.Close()
}

// SetDeadline sets the deadline on the underlying connection.
func (nt *NoiseTransport) SetDeadline(t time.Time) error {
	return nt.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
func (nt *NoiseTransport) SetReadDeadline(t time.Time) error {
	return nt.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
func (nt *NoiseTransport) SetWriteDeadline(t time.Time) error {
	return nt.conn.SetWriteDeadline(t)
}

// writeNoiseFrame writes a noise frame: 0x01 + BE16(len) + payload.
func (nt *NoiseTransport) writeNoiseFrame(payload []byte) error {
	if len(payload) > noiseMaxFrameSize {
		return fmt.Errorf("noise frame payload too large: %d bytes (max %d)", len(payload), noiseMaxFrameSize)
	}

	header := [3]byte{noisePreamble, byte(len(payload) >> 8), byte(len(payload) & 0xFF)}
	if _, err := nt.conn.Write(header[:]); err != nil {
		return fmt.Errorf("failed to write noise frame header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := nt.conn.Write(payload); err != nil {
			return fmt.Errorf("failed to write noise frame payload: %w", err)
		}
	}
	return nil
}

// readNoiseFrame reads a noise frame: 0x01 + BE16(len) + payload.
func (nt *NoiseTransport) readNoiseFrame() ([]byte, error) {
	var header [3]byte
	if _, err := io.ReadFull(nt.conn, header[:]); err != nil {
		return nil, err
	}

	if header[0] != noisePreamble {
		if header[0] == 0x00 {
			return nil, fmt.Errorf("device is using plaintext protocol; enable encryption on device or disable on client")
		}
		return nil, fmt.Errorf("invalid noise frame preamble: 0x%02X", header[0])
	}

	length := int(header[1])<<8 | int(header[2])
	if length == 0 {
		return nil, nil
	}
	if length > noiseMaxFrameSize {
		return nil, fmt.Errorf("noise frame too large: %d bytes", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(nt.conn, payload); err != nil {
		return nil, fmt.Errorf("failed to read noise frame payload: %w", err)
	}

	return payload, nil
}
