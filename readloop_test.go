package esphome_apiclient

import (
	"net"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestClient_ReadLoop_Dispatch(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	ch := make(chan proto.Message, 1)
	client.On(10, func(msg proto.Message) {
		ch <- msg
	})

	err := server.SendMessage(&pb.DeviceInfoResponse{Name: "test-device"}, 10)
	require.NoError(t, err)

	select {
	case msg := <-ch:
		resp, ok := msg.(*pb.DeviceInfoResponse)
		require.True(t, ok)
		assert.Equal(t, "test-device", resp.Name)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message dispatch")
	}
}

func TestClient_ReadLoop_FanOut(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	ch1 := make(chan proto.Message, 1)
	ch2 := make(chan proto.Message, 1)

	client.On(10, func(msg proto.Message) { ch1 <- msg })
	client.On(10, func(msg proto.Message) { ch2 <- msg })

	err := server.SendMessage(&pb.DeviceInfoResponse{Name: "test-device"}, 10)
	require.NoError(t, err)

	select {
	case <-ch1:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message dispatch 1")
	}
	select {
	case <-ch2:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message dispatch 2")
	}
}

func TestClient_ReadLoop_UnknownType(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	// Send an unknown type, say 9999
	// Use raw encoding since SendMessage might complain
	_, err := serverConn.Write([]byte{0, 2, 0x8f, 0x4e, 0x00, 0x00}) // fake frame for 9999
	require.NoError(t, err)

	ch := make(chan proto.Message, 1)
	client.On(10, func(msg proto.Message) { ch <- msg })

	// Send a valid message afterwards to ensure loop continues
	err = server.SendMessage(&pb.DeviceInfoResponse{Name: "test-device"}, 10)
	require.NoError(t, err)

	select {
	case msg := <-ch:
		resp, ok := msg.(*pb.DeviceInfoResponse)
		require.True(t, ok)
		assert.Equal(t, "test-device", resp.Name)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message dispatch")
	}
}

func TestClient_ReadLoop_CleanShutdown(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	client := newTestClient(clientConn)

	go client.readLoop()

	client.Close()

	select {
	case <-client.Done():
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for read loop to exit")
	}
}
