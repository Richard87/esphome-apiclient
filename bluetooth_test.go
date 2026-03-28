package esphome_apiclient

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func createBluetoothTestClient(t *testing.T) (*Client, *PlainFramer, func()) {
	clientConn, serverConn := net.Pipe()

	c := &Client{
		router:   NewRouter(),
		framer:   NewPlainFramer(clientConn),
		done:     make(chan struct{}),
		entities: NewEntityRegistry(),
		services: NewServiceRegistry(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel

	go c.readLoop()

	fServer := NewPlainFramer(serverConn)

	cleanup := func() {
		c.Close()
		serverConn.Close()
	}

	return c, fServer, cleanup
}

func TestBluetoothAdvertisements(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	received := make(chan proto.Message, 2)

	// Server side
	go func() {
		for {
			msgType, _, err := fServer.ReadFrame()
			if err != nil {
				return
			}
			if msgType == 66 {
				// Send legacy advertisement
				adv1 := &pb.BluetoothLEAdvertisementResponse{
					Address: 12345,
					Name:    []byte("Test Device"),
				}
				data1, _ := proto.Marshal(adv1)
				_ = fServer.WriteFrame(67, data1)

				// Send raw advertisement
				adv2 := &pb.BluetoothLERawAdvertisementsResponse{
					Advertisements: []*pb.BluetoothLERawAdvertisement{
						{
							Address: 67890,
							Rssi:    -50,
						},
					},
				}
				data2, _ := proto.Marshal(adv2)
				_ = fServer.WriteFrame(93, data2)
			}
		}
	}()

	unsubscribe, err := c.SubscribeBluetoothAdvertisements(func(msg proto.Message) {
		received <- msg
	})
	require.NoError(t, err)
	defer unsubscribe()

	select {
	case msg := <-received:
		resp := msg.(*pb.BluetoothLEAdvertisementResponse)
		assert.Equal(t, uint64(12345), resp.Address)
		assert.Equal(t, []byte("Test Device"), resp.Name)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for advertisement")
	}

	select {
	case msg := <-received:
		resp := msg.(*pb.BluetoothLERawAdvertisementsResponse)
		assert.Len(t, resp.Advertisements, 1)
		assert.Equal(t, uint64(67890), resp.Advertisements[0].Address)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for raw advertisement")
	}
}

func TestBluetoothConnectDisconnect(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	// Server side
	go func() {
		for {
			msgType, _, err := fServer.ReadFrame()
			if err != nil {
				return
			}
			if msgType == 68 {
				resp := &pb.BluetoothDeviceConnectionResponse{
					Address:   123,
					Connected: true,
				}
				// Simulate disconnect for the second call
				data, _ := proto.Marshal(resp)
				_ = fServer.WriteFrame(69, data)

				// Wait for next request
				msgType, data, err = fServer.ReadFrame()
				if err != nil {
					return
				}
				if msgType == 68 {
					var req pb.BluetoothDeviceRequest
					_ = proto.Unmarshal(data, &req)
					if req.RequestType == pb.BluetoothDeviceRequestType_BLUETOOTH_DEVICE_REQUEST_TYPE_DISCONNECT {
						resp.Connected = false
						data, _ = proto.Marshal(resp)
						_ = fServer.WriteFrame(69, data)
					}
				}
				return
			}
		}
	}()

	err := c.BluetoothConnect(123)
	assert.NoError(t, err)

	err = c.BluetoothDisconnect(123)
	assert.NoError(t, err)
}

func TestBluetoothGATTGetServices(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	go func() {
		for {
			msgType, _, err := fServer.ReadFrame()
			if err != nil {
				return
			}
			if msgType == 70 {
				// Send services
				resp1 := &pb.BluetoothGATTGetServicesResponse{
					Address: 123,
					Services: []*pb.BluetoothGATTService{
						{ShortUuid: 0x1800, Handle: 1},
					},
				}
				data1, _ := proto.Marshal(resp1)
				_ = fServer.WriteFrame(71, data1)

				// Send done
				resp2 := &pb.BluetoothGATTGetServicesDoneResponse{
					Address: 123,
				}
				data2, _ := proto.Marshal(resp2)
				_ = fServer.WriteFrame(72, data2)
			}
		}
	}()

	services, err := c.BluetoothGATTGetServices(123)
	assert.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, uint32(0x1800), services[0].ShortUuid)
}

func TestBluetoothGATTRead(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	// Test successful read
	go func() {
		_, _, _ = fServer.ReadFrame()
		resp := &pb.BluetoothGATTReadResponse{
			Address: 123,
			Handle:  45,
			Data:    []byte("hello"),
		}
		data, _ := proto.Marshal(resp)
		_ = fServer.WriteFrame(74, data)

		// Second call for error
		_, _, _ = fServer.ReadFrame()
		respErr := &pb.BluetoothGATTErrorResponse{
			Address: 123,
			Handle:  46,
			Error:   13,
		}
		dataErr, _ := proto.Marshal(respErr)
		_ = fServer.WriteFrame(82, dataErr)
	}()

	val, err := c.BluetoothGATTRead(123, 45)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), val)

	_, err = c.BluetoothGATTRead(123, 46)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GATT error")
}

func TestBluetoothGATTWrite(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	go func() {
		// Response for first write
		_, _, _ = fServer.ReadFrame()
		resp := &pb.BluetoothGATTWriteResponse{
			Address: 123,
			Handle:  45,
		}
		data, _ := proto.Marshal(resp)
		_ = fServer.WriteFrame(83, data)

		// Second write (no response)
		_, _, _ = fServer.ReadFrame()
	}()

	err := c.BluetoothGATTWrite(123, 45, []byte("world"), true)
	assert.NoError(t, err)

	err = c.BluetoothGATTWrite(123, 45, []byte("no-resp"), false)
	assert.NoError(t, err)
}

func TestBluetoothGATTNotify(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	received := make(chan []byte, 1)
	go func() {
		_, _, _ = fServer.ReadFrame()
		// Send response to enable notify
		resp := &pb.BluetoothGATTNotifyResponse{
			Address: 123,
			Handle:  45,
		}
		data, _ := proto.Marshal(resp)
		_ = fServer.WriteFrame(84, data)

		// Send notification data
		notif := &pb.BluetoothGATTNotifyDataResponse{
			Address: 123,
			Handle:  45,
			Data:    []byte("notify-data"),
		}
		data2, _ := proto.Marshal(notif)
		_ = fServer.WriteFrame(79, data2)
	}()

	unsubscribe, err := c.BluetoothGATTNotify(123, 45, true, func(data []byte) {
		received <- data
	})
	assert.NoError(t, err)
	defer unsubscribe()

	select {
	case data := <-received:
		assert.Equal(t, []byte("notify-data"), data)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for notification")
	}
}

func TestBluetoothConnectionsFree(t *testing.T) {
	c, fServer, cleanup := createBluetoothTestClient(t)
	defer cleanup()

	received := make(chan *pb.BluetoothConnectionsFreeResponse, 1)

	go func() {
		_, _, _ = fServer.ReadFrame()
		resp := &pb.BluetoothConnectionsFreeResponse{
			Free:  2,
			Limit: 3,
		}
		data, _ := proto.Marshal(resp)
		_ = fServer.WriteFrame(81, data)
	}()

	unsubscribe, err := c.SubscribeBluetoothConnectionsFree(func(resp *pb.BluetoothConnectionsFreeResponse) {
		received <- resp
	})
	assert.NoError(t, err)
	defer unsubscribe()

	select {
	case resp := <-received:
		assert.Equal(t, uint32(2), resp.Free)
		assert.Equal(t, uint32(3), resp.Limit)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for connections free")
	}
}
