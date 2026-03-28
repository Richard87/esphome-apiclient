package transport

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noiseTestServer simulates the ESPHome device side of a Noise handshake.
type noiseTestServer struct {
	conn      net.Conn
	encryptCS *noise.CipherState
	decryptCS *noise.CipherState
	name      string
	mac       string
}

func newNoiseTestServer(t *testing.T, conn net.Conn, psk []byte, name, mac string) *noiseTestServer {
	t.Helper()
	return &noiseTestServer{conn: conn, name: name, mac: mac}
}

// performHandshake runs the server side of the Noise handshake.
func (s *noiseTestServer) performHandshake(t *testing.T, psk []byte) {
	t.Helper()

	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             false,
		Prologue:              []byte("NoiseAPIInit\x00\x00"),
		PresharedKey:          psk,
		PresharedKeyPlacement: 0,
	})
	require.NoError(t, err)

	// Step 1: Read client hello (0x01 0x00 0x00)
	var clientHello [3]byte
	_, err = io.ReadFull(s.conn, clientHello[:])
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), clientHello[0])
	assert.Equal(t, byte(0x00), clientHello[1])
	assert.Equal(t, byte(0x00), clientHello[2])

	// Step 2: Read client handshake frame
	clientHandshakePayload := s.readNoiseFrame(t)
	require.NotEmpty(t, clientHandshakePayload)
	assert.Equal(t, byte(0x00), clientHandshakePayload[0])

	_, _, _, err = hs.ReadMessage(nil, clientHandshakePayload[1:])
	require.NoError(t, err)

	// Step 3: Send server hello
	serverHello := []byte{0x01}
	if s.name != "" {
		serverHello = append(serverHello, []byte(s.name)...)
		serverHello = append(serverHello, 0x00)
		if s.mac != "" {
			serverHello = append(serverHello, []byte(s.mac)...)
			serverHello = append(serverHello, 0x00)
		}
	}
	s.writeNoiseFrame(t, serverHello)

	// Step 4: Send handshake response
	handshakeResp, decryptCS, encryptCS, err := hs.WriteMessage(nil, nil)
	require.NoError(t, err)

	respPayload := append([]byte{0x00}, handshakeResp...)
	s.writeNoiseFrame(t, respPayload)

	// For the server, encrypt/decrypt are swapped relative to the client
	s.encryptCS = encryptCS
	s.decryptCS = decryptCS
}

func (s *noiseTestServer) writeNoiseFrame(t *testing.T, payload []byte) {
	t.Helper()
	header := [3]byte{0x01, byte(len(payload) >> 8), byte(len(payload) & 0xFF)}
	_, err := s.conn.Write(header[:])
	require.NoError(t, err)
	if len(payload) > 0 {
		_, err = s.conn.Write(payload)
		require.NoError(t, err)
	}
}

func (s *noiseTestServer) readNoiseFrame(t *testing.T) []byte {
	t.Helper()
	var header [3]byte
	_, err := io.ReadFull(s.conn, header[:])
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), header[0])
	length := int(header[1])<<8 | int(header[2])
	if length == 0 {
		return nil
	}
	payload := make([]byte, length)
	_, err = io.ReadFull(s.conn, payload)
	require.NoError(t, err)
	return payload
}

func (s *noiseTestServer) sendEncrypted(t *testing.T, msgType uint32, data []byte) {
	t.Helper()
	plaintext := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(plaintext[0:2], uint16(msgType))
	binary.BigEndian.PutUint16(plaintext[2:4], uint16(len(data)))
	copy(plaintext[4:], data)

	ciphertext, err := s.encryptCS.Encrypt(nil, nil, plaintext)
	require.NoError(t, err)
	s.writeNoiseFrame(t, ciphertext)
}

func (s *noiseTestServer) recvEncrypted(t *testing.T) (msgType uint32, data []byte) {
	t.Helper()
	ciphertext := s.readNoiseFrame(t)
	plaintext, err := s.decryptCS.Decrypt(nil, nil, ciphertext)
	require.NoError(t, err)
	require.True(t, len(plaintext) >= 4)

	msgType = uint32(binary.BigEndian.Uint16(plaintext[0:2]))
	data = plaintext[4:]
	return msgType, data
}

// --- Tests ---

func TestNoiseHandshake_Success(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "test-device", "AA:BB:CC:DD:EE:FF")

	var nt *NoiseTransport
	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		nt, clientErr = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
	}()

	server.performHandshake(t, psk)
	<-done

	require.NoError(t, clientErr)
	require.NotNil(t, nt)
	assert.Equal(t, "test-device", nt.ServerName)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", nt.ServerMAC)
}

func TestNoiseHandshake_WithExpectedName(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "my-device", "")

	var nt *NoiseTransport
	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		nt, clientErr = NewNoiseTransport(clientConn, psk, "my-device", 5*time.Second)
	}()

	server.performHandshake(t, psk)
	<-done

	require.NoError(t, clientErr)
	require.NotNil(t, nt)
	assert.Equal(t, "my-device", nt.ServerName)
}

func TestNoiseHandshake_NameMismatch(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		_, clientErr = NewNoiseTransport(clientConn, psk, "expected-device", 5*time.Second)
	}()

	// Server side: read client hello + handshake, send server hello with wrong name
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             false,
		Prologue:              []byte("NoiseAPIInit\x00\x00"),
		PresharedKey:          psk,
		PresharedKeyPlacement: 0,
	})
	require.NoError(t, err)

	var clientHello [3]byte
	_, err = io.ReadFull(serverConn, clientHello[:])
	require.NoError(t, err)

	var frameHeader [3]byte
	_, err = io.ReadFull(serverConn, frameHeader[:])
	require.NoError(t, err)
	length := int(frameHeader[1])<<8 | int(frameHeader[2])
	payload := make([]byte, length)
	_, err = io.ReadFull(serverConn, payload)
	require.NoError(t, err)
	_, _, _, err = hs.ReadMessage(nil, payload[1:])
	require.NoError(t, err)

	// Send server hello with "actual-device" name
	serverHello := []byte{0x01}
	serverHello = append(serverHello, []byte("actual-device")...)
	serverHello = append(serverHello, 0x00)
	shHeader := [3]byte{0x01, byte(len(serverHello) >> 8), byte(len(serverHello) & 0xFF)}
	_, _ = serverConn.Write(shHeader[:])
	_, _ = serverConn.Write(serverHello)

	<-done
	require.Error(t, clientErr)
	assert.Contains(t, clientErr.Error(), "name mismatch")
}

func TestNoiseHandshake_WrongPSK(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientPSK := make([]byte, 32)
	for i := range clientPSK {
		clientPSK[i] = byte(i)
	}
	serverPSK := make([]byte, 32)
	for i := range serverPSK {
		serverPSK[i] = byte(i + 100)
	}

	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		_, clientErr = NewNoiseTransport(clientConn, clientPSK, "", 5*time.Second)
	}()

	// Server with different PSK — handshake will fail
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             false,
		Prologue:              []byte("NoiseAPIInit\x00\x00"),
		PresharedKey:          serverPSK,
		PresharedKeyPlacement: 0,
	})
	require.NoError(t, err)

	var clientHello [3]byte
	_, _ = io.ReadFull(serverConn, clientHello[:])

	var header [3]byte
	_, _ = io.ReadFull(serverConn, header[:])
	length := int(header[1])<<8 | int(header[2])
	payload := make([]byte, length)
	_, _ = io.ReadFull(serverConn, payload)

	// ReadMessage will fail with mismatched PSK
	_, _, _, err = hs.ReadMessage(nil, payload[1:])

	// Send a server hello anyway so the client can proceed to the handshake step
	serverHello := []byte{0x01}
	shHeader := [3]byte{0x01, byte(len(serverHello) >> 8), byte(len(serverHello) & 0xFF)}
	_, _ = serverConn.Write(shHeader[:])
	_, _ = serverConn.Write(serverHello)

	// Send error response with MAC failure
	errPayload := append([]byte{0x01}, []byte("Handshake MAC failure")...)
	errHeader := [3]byte{0x01, byte(len(errPayload) >> 8), byte(len(errPayload) & 0xFF)}
	_, _ = serverConn.Write(errHeader[:])
	_, _ = serverConn.Write(errPayload)

	<-done
	require.Error(t, clientErr)
}

func TestNoiseHandshake_InvalidPSKLength(t *testing.T) {
	clientConn, _ := net.Pipe()
	defer clientConn.Close()

	shortPSK := make([]byte, 16)
	_, err := NewNoiseTransport(clientConn, shortPSK, "", 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestNoiseTransport_RoundTrip(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "test-device", "")

	var nt *NoiseTransport
	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		nt, clientErr = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
	}()

	server.performHandshake(t, psk)
	<-done
	require.NoError(t, clientErr)

	// Test client -> server
	testData := []byte{0x08, 0x01, 0x10, 0x0A}
	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		err := nt.WriteFrame(33, testData)
		assert.NoError(t, err)
	}()

	msgType, data := server.recvEncrypted(t)
	assert.Equal(t, uint32(33), msgType)
	assert.Equal(t, testData, data)
	<-sendDone

	// Test server -> client
	serverData := []byte{0x08, 0x02, 0x15, 0x00, 0x00, 0xBC, 0x41}
	go func() {
		server.sendEncrypted(t, 25, serverData)
	}()

	recvType, recvData, err := nt.ReadFrame()
	require.NoError(t, err)
	assert.Equal(t, uint32(25), recvType)
	assert.Equal(t, serverData, recvData)
}

func TestNoiseTransport_MultipleMessages(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "test-device", "")

	var nt *NoiseTransport
	done := make(chan struct{})

	go func() {
		defer close(done)
		var err error
		nt, err = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
		require.NoError(t, err)
	}()

	server.performHandshake(t, psk)
	<-done

	// Send 5 messages from client -> server
	for i := uint32(0); i < 5; i++ {
		data := []byte{byte(i), byte(i + 1)}
		sendDone := make(chan struct{})
		go func() {
			defer close(sendDone)
			err := nt.WriteFrame(i+1, data)
			assert.NoError(t, err)
		}()
		msgType, recvData := server.recvEncrypted(t)
		assert.Equal(t, i+1, msgType)
		assert.Equal(t, data, recvData)
		<-sendDone
	}

	// Send 5 messages from server -> client
	for i := uint32(0); i < 5; i++ {
		data := []byte{byte(i + 10), byte(i + 20)}
		go func() {
			server.sendEncrypted(t, i+100, data)
		}()

		msgType, recvData, err := nt.ReadFrame()
		require.NoError(t, err)
		assert.Equal(t, i+100, msgType)
		assert.Equal(t, data, recvData)
	}
}

func TestNoiseTransport_EmptyPayload(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "test-device", "")

	var nt *NoiseTransport
	done := make(chan struct{})

	go func() {
		defer close(done)
		var err error
		nt, err = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
		require.NoError(t, err)
	}()

	server.performHandshake(t, psk)
	<-done

	// Send empty payload message (e.g. PingResponse)
	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		err := nt.WriteFrame(8, nil)
		assert.NoError(t, err)
	}()

	msgType, data := server.recvEncrypted(t)
	assert.Equal(t, uint32(8), msgType)
	assert.Empty(t, data)
	<-sendDone
}

func TestNoiseHandshake_ServerError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)

	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		_, clientErr = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
	}()

	// Server: go through the motions but send "Handshake MAC failure"
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             false,
		Prologue:              []byte("NoiseAPIInit\x00\x00"),
		PresharedKey:          psk,
		PresharedKeyPlacement: 0,
	})
	require.NoError(t, err)

	var hdr [3]byte
	_, _ = io.ReadFull(serverConn, hdr[:])

	var fh [3]byte
	_, _ = io.ReadFull(serverConn, fh[:])
	length := int(fh[1])<<8 | int(fh[2])
	p := make([]byte, length)
	_, _ = io.ReadFull(serverConn, p)
	_, _, _, _ = hs.ReadMessage(nil, p[1:])

	// Send server hello
	serverHello := []byte{0x01, 't', 'e', 's', 't', 0x00}
	shHeader := [3]byte{0x01, byte(len(serverHello) >> 8), byte(len(serverHello) & 0xFF)}
	_, _ = serverConn.Write(shHeader[:])
	_, _ = serverConn.Write(serverHello)

	// Send MAC failure error
	errPayload := append([]byte{0x01}, []byte("Handshake MAC failure")...)
	errHeader := [3]byte{0x01, byte(len(errPayload) >> 8), byte(len(errPayload) & 0xFF)}
	_, _ = serverConn.Write(errHeader[:])
	_, _ = serverConn.Write(errPayload)

	<-done
	require.Error(t, clientErr)
	assert.Contains(t, clientErr.Error(), "invalid encryption key")
}

func TestNoiseHandshake_GenericError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	psk := make([]byte, 32)

	var clientErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		_, clientErr = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
	}()

	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           cs,
		Pattern:               noise.HandshakeNN,
		Initiator:             false,
		Prologue:              []byte("NoiseAPIInit\x00\x00"),
		PresharedKey:          psk,
		PresharedKeyPlacement: 0,
	})
	require.NoError(t, err)

	var h [3]byte
	_, _ = io.ReadFull(serverConn, h[:])
	var fh [3]byte
	_, _ = io.ReadFull(serverConn, fh[:])
	length := int(fh[1])<<8 | int(fh[2])
	p := make([]byte, length)
	_, _ = io.ReadFull(serverConn, p)
	_, _, _, _ = hs.ReadMessage(nil, p[1:])

	// Send server hello
	serverHello := []byte{0x01}
	shH := [3]byte{0x01, 0x00, byte(len(serverHello))}
	_, _ = serverConn.Write(shH[:])
	_, _ = serverConn.Write(serverHello)

	// Send generic error
	errPayload := append([]byte{0x01}, []byte("Something went wrong")...)
	errH := [3]byte{0x01, byte(len(errPayload) >> 8), byte(len(errPayload) & 0xFF)}
	_, _ = serverConn.Write(errH[:])
	_, _ = serverConn.Write(errPayload)

	<-done
	require.Error(t, clientErr)
	assert.Contains(t, clientErr.Error(), "handshake error")
	assert.Contains(t, clientErr.Error(), "Something went wrong")
}

func TestNoiseTransport_Close(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	psk := make([]byte, 32)
	for i := range psk {
		psk[i] = byte(i)
	}

	server := newNoiseTestServer(t, serverConn, psk, "test-device", "")

	var nt *NoiseTransport
	done := make(chan struct{})

	go func() {
		defer close(done)
		var err error
		nt, err = NewNoiseTransport(clientConn, psk, "", 5*time.Second)
		require.NoError(t, err)
	}()

	server.performHandshake(t, psk)
	<-done

	err := nt.Close()
	require.NoError(t, err)

	// Subsequent reads should fail
	_, _, err = nt.ReadFrame()
	require.Error(t, err)
}
