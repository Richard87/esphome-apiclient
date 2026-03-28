package esphome_apiclient

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Log Streaming Tests ----

func TestSubscribeLogs(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	var received []*pb.SubscribeLogsResponse
	var mu sync.Mutex
	done := make(chan struct{}, 3)

	go func() {
		// Read the SubscribeLogsRequest
		msg, msgType, err := server.RecvMessage()
		if err != nil {
			return
		}
		assert.Equal(t, uint32(28), msgType)
		req, ok := msg.(*pb.SubscribeLogsRequest)
		assert.True(t, ok)
		assert.Equal(t, pb.LogLevel_LOG_LEVEL_DEBUG, req.Level)

		// Send log messages
		_ = server.SendMessage(&pb.SubscribeLogsResponse{
			Level:   pb.LogLevel_LOG_LEVEL_INFO,
			Message: []byte("Boot complete"),
		}, 29)
		_ = server.SendMessage(&pb.SubscribeLogsResponse{
			Level:   pb.LogLevel_LOG_LEVEL_DEBUG,
			Message: []byte("Sensor update"),
		}, 29)
		_ = server.SendMessage(&pb.SubscribeLogsResponse{
			Level:   pb.LogLevel_LOG_LEVEL_ERROR,
			Message: []byte("Connection failed"),
		}, 29)
	}()

	unsub, err := client.SubscribeLogs(pb.LogLevel_LOG_LEVEL_DEBUG, func(msg *pb.SubscribeLogsResponse) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
		done <- struct{}{}
	})
	require.NoError(t, err)
	defer unsub()

	// Wait for all 3 messages
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for log message %d", i+1)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 3)
	assert.Equal(t, pb.LogLevel_LOG_LEVEL_INFO, received[0].Level)
	assert.Equal(t, "Boot complete", string(received[0].Message))
	assert.Equal(t, pb.LogLevel_LOG_LEVEL_DEBUG, received[1].Level)
	assert.Equal(t, "Sensor update", string(received[1].Message))
	assert.Equal(t, pb.LogLevel_LOG_LEVEL_ERROR, received[2].Level)
	assert.Equal(t, "Connection failed", string(received[2].Message))
}

func TestSubscribeLogs_Unsubscribe(t *testing.T) {
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
		_, _, _ = server.RecvMessage() // SubscribeLogsRequest
	}()

	unsub, err := client.SubscribeLogs(pb.LogLevel_LOG_LEVEL_DEBUG, func(msg *pb.SubscribeLogsResponse) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	require.NoError(t, err)

	// Unsubscribe and send a log — handler should NOT fire
	unsub()

	_ = server.SendMessage(&pb.SubscribeLogsResponse{
		Level:   pb.LogLevel_LOG_LEVEL_INFO,
		Message: []byte("Should not arrive"),
	}, 29)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 0, callCount)
	mu.Unlock()
}

// ---- Service Tests ----

func TestServiceRegistry(t *testing.T) {
	reg := NewServiceRegistry()

	reg.HandleServiceDefinition(&pb.ListEntitiesServicesResponse{
		Name: "restart",
		Key:  0xABCD1234,
		Args: []*pb.ListEntitiesServicesArgument{},
	})

	reg.HandleServiceDefinition(&pb.ListEntitiesServicesResponse{
		Name: "set_value",
		Key:  0xBBBB2222,
		Args: []*pb.ListEntitiesServicesArgument{
			{Name: "value", Type: pb.ServiceArgType_SERVICE_ARG_TYPE_FLOAT},
			{Name: "label", Type: pb.ServiceArgType_SERVICE_ARG_TYPE_STRING},
		},
	})

	// ByKey
	svc := reg.ByKey(0xABCD1234)
	require.NotNil(t, svc)
	assert.Equal(t, "restart", svc.Name)
	assert.Empty(t, svc.Args)

	// ByName
	svc2 := reg.ByName("set_value")
	require.NotNil(t, svc2)
	assert.Equal(t, uint32(0xBBBB2222), svc2.Key)
	assert.Len(t, svc2.Args, 2)
	assert.Equal(t, "value", svc2.Args[0].Name)
	assert.Equal(t, pb.ServiceArgType_SERVICE_ARG_TYPE_FLOAT, svc2.Args[0].Type)
	assert.Equal(t, "label", svc2.Args[1].Name)
	assert.Equal(t, pb.ServiceArgType_SERVICE_ARG_TYPE_STRING, svc2.Args[1].Type)

	// All
	all := reg.All()
	assert.Len(t, all, 2)

	// Missing
	assert.Nil(t, reg.ByKey(0x99999999))
	assert.Nil(t, reg.ByName("nonexistent"))

	// Clear
	reg.Clear()
	assert.Len(t, reg.All(), 0)
}

func TestExecuteService(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(42), msgType) // ExecuteServiceRequest
		req, ok := msg.(*pb.ExecuteServiceRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(0xABCD1234), req.Key)
		require.Len(t, req.Args, 3)
		// Verify mixed argument types
		assert.True(t, req.Args[0].Bool_)
		assert.InDelta(t, float32(3.14), req.Args[1].Float_, 0.01)
		assert.Equal(t, "hello", req.Args[2].String_)
	}()

	err := client.ExecuteService(0xABCD1234, []*pb.ExecuteServiceArgument{
		{Bool_: true},
		{Float_: 3.14},
		{String_: "hello"},
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive service call")
	}
}

func TestExecuteServiceByName(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()
	defer client.Close()

	// Register a service
	client.services.HandleServiceDefinition(&pb.ListEntitiesServicesResponse{
		Name: "test_service",
		Key:  0x11112222,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(42), msgType)
		req, ok := msg.(*pb.ExecuteServiceRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(0x11112222), req.Key)
	}()

	err := client.ExecuteServiceByName("test_service", nil)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for service execution")
	}
}

func TestExecuteServiceByName_NotFound(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)

	go client.readLoop()
	defer client.Close()

	err := client.ExecuteServiceByName("nonexistent", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListEntities_DiscoverServices(t *testing.T) {
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
			// Send a sensor
			_ = server.SendMessage(&pb.ListEntitiesSensorResponse{
				ObjectId: "sensor_1", Key: 1, Name: "Sensor 1",
			}, 16)
			// Send a service
			_ = server.SendMessage(&pb.ListEntitiesServicesResponse{
				Name: "my_service",
				Key:  0x55556666,
				Args: []*pb.ListEntitiesServicesArgument{
					{Name: "param1", Type: pb.ServiceArgType_SERVICE_ARG_TYPE_INT},
				},
			}, 41)
			_ = server.SendMessage(&pb.ListEntitiesDoneResponse{}, 19)
		}
	}()

	entities, err := client.ListEntities()
	require.NoError(t, err)
	assert.Len(t, entities, 2) // sensor + service

	// Verify service was added to the service registry
	svc := client.Services().ByName("my_service")
	require.NotNil(t, svc)
	assert.Equal(t, uint32(0x55556666), svc.Key)
	assert.Len(t, svc.Args, 1)
	assert.Equal(t, "param1", svc.Args[0].Name)
}
