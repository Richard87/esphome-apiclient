package esphome_apiclient

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// mockTransport implements Transport using an existing net.Conn for testing.
type mockTransport struct {
	conn net.Conn
}

func (m *mockTransport) Read(p []byte) (n int, err error) {
	return m.conn.Read(p)
}

func (m *mockTransport) Write(p []byte) (n int, err error) {
	return m.conn.Write(p)
}

func (m *mockTransport) Close() error {
	return m.conn.Close()
}

func (m *mockTransport) SetDeadline(t time.Time) error {
	return m.conn.SetDeadline(t)
}

func (m *mockTransport) SetReadDeadline(t time.Time) error {
	return m.conn.SetReadDeadline(t)
}

func (m *mockTransport) SetWriteDeadline(t time.Time) error {
	return m.conn.SetWriteDeadline(t)
}

// newTestClient creates a Client suitable for unit tests with all required fields.
func newTestClient(conn net.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		framer:          NewPlainFramer(&mockTransport{conn: conn}),
		router:          NewRouter(),
		entities:        NewEntityRegistry(),
		services:        NewServiceRegistry(),
		done:            make(chan struct{}),
		clientInfo:      "test-client",
		apiVersionMajor: 1,
		apiVersionMinor: 10,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// newTestServer creates a minimal Client that can send/receive messages (server side of tests).
func newTestServer(conn net.Conn) *Client {
	return &Client{
		framer: NewPlainFramer(&mockTransport{conn: conn}),
	}
}

func TestClient_SendRecv(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{
		framer: NewPlainFramer(&mockTransport{conn: clientConn}),
	}
	server := &Client{
		framer: NewPlainFramer(&mockTransport{conn: serverConn}),
	}

	go func() {
		err := client.SendMessage(&pb.HelloRequest{
			ClientInfo:      "test-client",
			ApiVersionMajor: 1,
			ApiVersionMinor: 10,
		}, 1)
		assert.NoError(t, err)
	}()

	msg, msgType, err := server.RecvMessage()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), msgType)

	helloReq, ok := msg.(*pb.HelloRequest)
	require.True(t, ok)
	assert.Equal(t, "test-client", helloReq.ClientInfo)
	assert.Equal(t, uint32(1), helloReq.ApiVersionMajor)
	assert.Equal(t, uint32(10), helloReq.ApiVersionMinor)
}

func TestHandshake_Success(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{
		framer:          NewPlainFramer(&mockTransport{conn: clientConn}),
		clientInfo:      "test-client",
		apiVersionMajor: 1,
		apiVersionMinor: 10,
	}
	server := &Client{
		framer: NewPlainFramer(&mockTransport{conn: serverConn}),
	}

	go func() {
		// Wait for HelloRequest
		msg, msgType, err := server.RecvMessage()
		assert.NoError(t, err)
		assert.Equal(t, uint32(1), msgType)
		req, ok := msg.(*pb.HelloRequest)
		assert.True(t, ok)
		assert.Equal(t, "test-client", req.ClientInfo)
		assert.Equal(t, uint32(1), req.ApiVersionMajor)

		// Send HelloResponse
		err = server.SendMessage(&pb.HelloResponse{
			ApiVersionMajor: 1,
			ApiVersionMinor: 10,
			ServerInfo:      "test-server",
			Name:            "my-device",
		}, 2)
		assert.NoError(t, err)
	}()

	err := client.handshake()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), client.apiVersionMajor)
	assert.Equal(t, uint32(10), client.apiVersionMinor)
	assert.Equal(t, "test-server", client.serverInfo)
	assert.Equal(t, "my-device", client.name)
}

func TestHandshake_VersionMismatch(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{
		framer:          NewPlainFramer(&mockTransport{conn: clientConn}),
		clientInfo:      "test-client",
		apiVersionMajor: 1,
		apiVersionMinor: 10,
	}
	server := &Client{
		framer: NewPlainFramer(&mockTransport{conn: serverConn}),
	}

	go func() {
		_, _, _ = server.RecvMessage()
		// Send HelloResponse with major version 99
		err := server.SendMessage(&pb.HelloResponse{
			ApiVersionMajor: 99,
			ApiVersionMinor: 0,
		}, 2)
		assert.NoError(t, err)
	}()

	err := client.handshake()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported API version 99")
}

func TestHandshake_UnexpectedMessage(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{
		framer:          NewPlainFramer(&mockTransport{conn: clientConn}),
		clientInfo:      "test-client",
		apiVersionMajor: 1,
		apiVersionMinor: 10,
	}
	server := &Client{
		framer: NewPlainFramer(&mockTransport{conn: serverConn}),
	}

	go func() {
		_, _, _ = server.RecvMessage()
		// Send PingRequest (id=7) instead of HelloResponse (id=2)
		err := server.SendMessage(&pb.PingRequest{}, 7)
		assert.NoError(t, err)
	}()

	err := client.handshake()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected HelloResponse (msgType 2), got 7")
}

func TestDial_Timeout(t *testing.T) {
	// A non-routable or dropped address typically. Let's use an address that drops packets or a reserved IP
	// 192.0.2.0/24 is TEST-NET-1, usually dropped/unroutable.
	start := time.Now()
	timeout := 100 * time.Millisecond
	_, err := Dial("192.0.2.1:12345", timeout)
	duration := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, duration, timeout+50*time.Millisecond) // Within reason
}

func TestDial_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		server := &Client{
			framer: NewPlainFramer(&mockTransport{conn: conn}),
		}
		_, _, _ = server.RecvMessage()
		_ = server.SendMessage(&pb.HelloResponse{
			ApiVersionMajor: 1,
			ApiVersionMinor: 10,
		}, 2)
	}()

	client, err := Dial(ln.Addr().String(), 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()
}

func TestClient_DeviceInfo(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 9 {
			_ = server.SendMessage(&pb.DeviceInfoResponse{
				Name:           "my-test-device",
				MacAddress:     "AA:BB:CC:DD:EE:FF",
				EsphomeVersion: "2023.12.0",
			}, 10)
		}
	}()

	info, err := client.DeviceInfo()
	require.NoError(t, err)
	assert.Equal(t, "my-test-device", info.Name)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", info.MacAddress)
	assert.Equal(t, "2023.12.0", info.EsphomeVersion)
}

func TestClient_ListEntities(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 11 { // ListEntitiesRequest
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_1",
				Key:      1,
				Name:     "Sensor 1",
			}, 16)
			_ = server.SendMessage(&pb.ListEntitiesSwitchResponse{
				ObjectId: "switch_1",
				Key:      2,
				Name:     "Switch 1",
			}, 17)
			_ = server.SendMessage(&pb.ListEntitiesDoneResponse{}, 19)
		}
	}()

	entities, err := client.ListEntities()
	require.NoError(t, err)
	assert.Len(t, entities, 2)
}

func TestClient_Ping(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 7 { // PingRequest
			_ = server.SendMessage(&pb.PingResponse{}, 8)
		}
	}()

	err := client.Ping()
	require.NoError(t, err)
}

func TestClient_PingTimeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)

	go client.readLoop()
	defer client.Close()

	// Drain the server side so writes don't block, but never respond
	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := serverConn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	// Server never responds — use short timeout
	err := client.PingWithTimeout(100 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestClient_Disconnect(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	disconnected := make(chan bool)
	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 5 { // DisconnectRequest
			_ = server.SendMessage(&pb.DisconnectResponse{}, 6)
			disconnected <- true
		}
	}()

	err := client.Disconnect()
	require.NoError(t, err)

	<-disconnected
}

func TestClient_ListEntities_FiveEntities(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 11 { // ListEntitiesRequest
			// 3 sensors
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_1", Key: 1, Name: "Temperature",
			}, 16)
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_2", Key: 2, Name: "Humidity",
			}, 16)
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_3", Key: 3, Name: "Pressure",
			}, 16)
			// 2 switches
			_ = server.SendMessage(&pb.ListEntitiesSwitchResponse{
				ObjectId: "switch_1", Key: 4, Name: "Relay 1",
			}, 17)
			_ = server.SendMessage(&pb.ListEntitiesSwitchResponse{
				ObjectId: "switch_2", Key: 5, Name: "Relay 2",
			}, 17)
			// Done
			_ = server.SendMessage(&pb.ListEntitiesDoneResponse{}, 19)
		}
	}()

	entities, err := client.ListEntities()
	require.NoError(t, err)
	assert.Len(t, entities, 5)

	// Verify types
	sensorCount := 0
	switchCount := 0
	for _, e := range entities {
		switch e.(type) {
		case *pb.ListEntitiesSensorResponse:
			sensorCount++
		case *pb.ListEntitiesSwitchResponse:
			switchCount++
		}
	}
	assert.Equal(t, 3, sensorCount)
	assert.Equal(t, 2, switchCount)
}

func TestClient_ListEntities_Timeout(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 11 {
			// Send some entities but never the Done response
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_1", Key: 1, Name: "Temperature",
			}, 16)
		}
	}()

	_, err := client.ListEntitiesWithTimeout(200 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestClient_SubscribeStates(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	var received []proto.Message
	var mu sync.Mutex
	done := make(chan struct{}, 3)

	go func() {
		// Read the SubscribeStatesRequest
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 20 {
			// Send a sensor state
			_ = server.SendMessage(&pb.SensorStateResponse{
				Key: 1, State: 23.5,
			}, 25)
			// Send a binary sensor state
			_ = server.SendMessage(&pb.BinarySensorStateResponse{
				Key: 2, State: true,
			}, 21)
			// Send a switch state
			_ = server.SendMessage(&pb.SwitchStateResponse{
				Key: 3, State: false,
			}, 26)
		}
	}()

	unsub, err := client.SubscribeStates(func(msg proto.Message) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		done <- struct{}{}
	})
	require.NoError(t, err)
	require.NotNil(t, unsub)
	defer unsub()

	// Wait for all 3 state messages
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for state message")
		}
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, received, 3)
}

func TestClient_SubscribeStates_Unsubscribe(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	callCount := 0
	var mu sync.Mutex

	go func() {
		_, _, _ = server.RecvMessage() // SubscribeStatesRequest
	}()

	unsub, err := client.SubscribeStates(func(msg proto.Message) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	require.NoError(t, err)

	// Unsubscribe immediately, then send a state — handler should NOT fire
	unsub()

	_ = server.SendMessage(&pb.SensorStateResponse{Key: 1, State: 42.0}, 25)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 0, callCount)
	mu.Unlock()
}

func TestClient_Disconnect_Sequence(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	requestReceived := make(chan bool, 1)
	responseBeforeClose := make(chan bool, 1)

	go func() {
		_, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		if msgType == 5 { // DisconnectRequest
			requestReceived <- true
			_ = server.SendMessage(&pb.DisconnectResponse{}, 6)
		}
	}()

	go func() {
		// Verify request was received before the client returns
		select {
		case <-requestReceived:
			responseBeforeClose <- true
		case <-time.After(2 * time.Second):
		}
	}()

	err := client.Disconnect()
	require.NoError(t, err)

	select {
	case ok := <-responseBeforeClose:
		assert.True(t, ok)
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not receive DisconnectRequest")
	}

	// Verify transport is closed — read loop should exit
	select {
	case <-client.Done():
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Read loop did not exit after Disconnect")
	}
}
