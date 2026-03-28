package esphome_apiclient

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Keepalive Tests ----

func TestKeepalive_Fires(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newTestClient(clientConn)
	client.ctx = ctx
	client.cancel = cancel
	client.keepaliveInterval = 100 * time.Millisecond
	client.keepaliveTimeout = 500 * time.Millisecond
	client.connected.Store(true)

	server := newTestServer(serverConn)

	go client.readLoop()

	// Server: respond to pings
	var pingCount atomic.Int32
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for {
			msg, msgType, err := server.RecvMessage()
			if err != nil {
				return
			}
			_ = msg
			if msgType == 7 { // PingRequest
				pingCount.Add(1)
				_ = server.SendMessage(&pb.PingResponse{}, 8)
			}
		}
	}()

	go client.keepaliveLoop()

	// Wait for at least 2 pings
	time.Sleep(350 * time.Millisecond)
	cancel()
	// Close pipes to unblock goroutines stuck on reads
	clientConn.Close()
	serverConn.Close()
	<-serverDone

	assert.GreaterOrEqual(t, pingCount.Load(), int32(2), "expected at least 2 keepalive pings")
}

func TestKeepalive_DeadConnectionDetection(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newTestClient(clientConn)
	client.ctx = ctx
	client.cancel = cancel
	client.keepaliveInterval = 50 * time.Millisecond
	client.keepaliveTimeout = 50 * time.Millisecond
	client.connected.Store(true)

	disconnected := make(chan struct{})
	client.onDisconnect = func() {
		close(disconnected)
	}

	go client.readLoop()

	// Server: drain but never respond to pings
	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := serverConn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	go client.keepaliveLoop()

	// Wait for disconnect detection
	select {
	case <-disconnected:
		// Success — dead connection detected
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: dead connection was not detected")
	}

	assert.False(t, client.Connected())
}

func TestContextCancellation_StopsKeepalive(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	ctx, cancel := context.WithCancel(context.Background())

	client := newTestClient(clientConn)
	client.ctx = ctx
	client.cancel = cancel
	client.keepaliveInterval = 50 * time.Millisecond
	client.keepaliveTimeout = 50 * time.Millisecond
	client.connected.Store(true)

	go client.readLoop()

	// Server: respond to pings
	go func() {
		server := newTestServer(serverConn)
		for {
			_, msgType, err := server.RecvMessage()
			if err != nil {
				return
			}
			if msgType == 7 {
				_ = server.SendMessage(&pb.PingResponse{}, 8)
			}
		}
	}()

	keepaliveDone := make(chan struct{})
	go func() {
		client.keepaliveLoop()
		close(keepaliveDone)
	}()

	// Let it ping a couple times
	time.Sleep(150 * time.Millisecond)

	// Cancel context — keepalive should exit
	cancel()

	select {
	case <-keepaliveDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Keepalive goroutine did not exit on context cancellation")
	}
}

func TestConnected_State(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		server := newTestServer(conn)
		_, _, _ = server.RecvMessage()
		_ = server.SendMessage(&pb.HelloResponse{
			ApiVersionMajor: 1,
			ApiVersionMinor: 10,
			ServerInfo:      "test-server",
			Name:            "test-device",
		}, 2)
		// Keep connection alive
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	client, err := Dial(ln.Addr().String(), 1*time.Second,
		WithKeepalive(0), // disable keepalive for this test
	)
	require.NoError(t, err)
	defer client.Close()

	assert.True(t, client.Connected())
	assert.Equal(t, "test-device", client.Name())
	assert.Equal(t, "test-server", client.ServerInfo())

	major, minor := client.APIVersion()
	assert.Equal(t, uint32(1), major)
	assert.Equal(t, uint32(10), minor)

	client.Close()
	// After close, connected should go to false
	time.Sleep(50 * time.Millisecond)
	assert.False(t, client.Connected())
}

func TestReconnect_ReEstablishesConnection(t *testing.T) {
	// Create a server that accepts two connections
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	connectionCount := atomic.Int32{}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			connectionCount.Add(1)
			go func(c net.Conn) {
				defer c.Close()
				server := newTestServer(c)
				// Handle HelloRequest
				_, _, err := server.RecvMessage()
				if err != nil {
					return
				}
				_ = server.SendMessage(&pb.HelloResponse{
					ApiVersionMajor: 1,
					ApiVersionMinor: 10,
					Name:            "test-device",
				}, 2)
				// Respond to messages (ListEntities, SubscribeStates, Ping)
				for {
					_, msgType, err := server.RecvMessage()
					if err != nil {
						return
					}
					switch msgType {
					case 7: // PingRequest
						_ = server.SendMessage(&pb.PingResponse{}, 8)
					case 11: // ListEntitiesRequest
						_ = server.SendMessage(&pb.ListEntitiesDoneResponse{}, 19)
					case 20: // SubscribeStatesRequest
						// OK, just accept
					}
				}
			}(conn)
		}
	}()

	reconnected := make(chan struct{}, 1)
	client, err := Dial(ln.Addr().String(), 1*time.Second,
		WithKeepalive(0), // disable keepalive
		WithReconnect(50*time.Millisecond),
		WithOnConnect(func() {
			select {
			case reconnected <- struct{}{}:
			default:
			}
		}),
	)
	require.NoError(t, err)
	defer client.Close()

	// Drain reconnected for initial connect
	<-reconnected

	assert.Equal(t, int32(1), connectionCount.Load())

	// Force a disconnect by closing the framer
	client.framer.Close()

	// Wait for reconnect
	select {
	case <-reconnected:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for reconnect")
	}

	assert.Equal(t, int32(2), connectionCount.Load())
	assert.True(t, client.Connected())
}
