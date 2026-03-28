package esphome_apiclient

import (
	"fmt"
	"sync"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
)

// SubscribeBluetoothAdvertisements subscribes to Bluetooth LE advertisement packets.
// It handles both the legacy BluetoothLEAdvertisementResponse and the newer BluetoothLERawAdvertisementsResponse.
func (c *Client) SubscribeBluetoothAdvertisements(handler func(proto.Message)) (func(), error) {
	var removes []func()

	// Handle legacy advertisements
	removes = append(removes, c.On(67, func(msg proto.Message) {
		if handler != nil {
			handler(msg)
		}
	}))

	// Handle raw advertisements
	removes = append(removes, c.On(93, func(msg proto.Message) {
		if handler != nil {
			handler(msg)
		}
	}))

	if err := c.SendMessage(&pb.SubscribeBluetoothLEAdvertisementsRequest{}, 66); err != nil {
		for _, rem := range removes {
			rem()
		}
		return nil, fmt.Errorf("SubscribeBluetoothAdvertisements: %w", err)
	}

	unsubscribe := func() {
		for _, rem := range removes {
			rem()
		}
		// Try to unsubscribe on the server as well, but don't fail if it doesn't work
		_ = c.SendMessage(&pb.UnsubscribeBluetoothLEAdvertisementsRequest{}, 87)
	}

	return unsubscribe, nil
}

// BluetoothConnect connects to a Bluetooth device.
func (c *Client) BluetoothConnect(address uint64) error {
	// Using CONNECT_V3_WITH_CACHE as it's the modern way
	req := &pb.BluetoothDeviceRequest{
		Address:     address,
		RequestType: pb.BluetoothDeviceRequestType_BLUETOOTH_DEVICE_REQUEST_TYPE_CONNECT_V3_WITH_CACHE,
	}

	resp, err := c.requestResponse(req, 68, 69, 30*time.Second)
	if err != nil {
		return fmt.Errorf("BluetoothConnect: %w", err)
	}

	connResp := resp.(*pb.BluetoothDeviceConnectionResponse)
	if !connResp.Connected {
		return fmt.Errorf("BluetoothConnect: failed to connect (address: %d)", address)
	}

	return nil
}

// BluetoothDisconnect disconnects from a Bluetooth device.
func (c *Client) BluetoothDisconnect(address uint64) error {
	req := &pb.BluetoothDeviceRequest{
		Address:     address,
		RequestType: pb.BluetoothDeviceRequestType_BLUETOOTH_DEVICE_REQUEST_TYPE_DISCONNECT,
	}

	resp, err := c.requestResponse(req, 68, 69, 10*time.Second)
	if err != nil {
		return fmt.Errorf("BluetoothDisconnect: %w", err)
	}

	connResp := resp.(*pb.BluetoothDeviceConnectionResponse)
	if connResp.Connected {
		return fmt.Errorf("BluetoothDisconnect: failed to disconnect (address: %d)", address)
	}

	return nil
}

// BluetoothGATTGetServices retrieves the GATT services of a connected Bluetooth device.
func (c *Client) BluetoothGATTGetServices(address uint64) ([]*pb.BluetoothGATTService, error) {
	var services []*pb.BluetoothGATTService
	var mu sync.Mutex
	done := make(chan struct{})
	var errResult error

	// Register handler for services
	removeService := c.On(71, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTGetServicesResponse)
		if resp.Address != address {
			return
		}
		mu.Lock()
		services = append(services, resp.Services...)
		mu.Unlock()
	})
	defer removeService()

	// Register handler for done
	removeDone := c.On(72, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTGetServicesDoneResponse)
		if resp.Address != address {
			return
		}
		close(done)
	})
	defer removeDone()

	// Register handler for error
	removeError := c.On(82, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTErrorResponse)
		if resp.Address != address {
			return
		}
		errResult = fmt.Errorf("GATT error (code: %d)", resp.Error)
		close(done)
	})
	defer removeError()

	req := &pb.BluetoothGATTGetServicesRequest{
		Address: address,
	}

	if err := c.SendMessage(req, 70); err != nil {
		return nil, fmt.Errorf("BluetoothGATTGetServices: %w", err)
	}

	select {
	case <-done:
		if errResult != nil {
			return nil, errResult
		}
		return services, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("BluetoothGATTGetServices: timeout")
	case <-c.Done():
		return nil, fmt.Errorf("BluetoothGATTGetServices: client closed")
	}
}

// BluetoothGATTRead reads a GATT characteristic.
func (c *Client) BluetoothGATTRead(address uint64, handle uint32) ([]byte, error) {
	req := &pb.BluetoothGATTReadRequest{
		Address: address,
		Handle:  handle,
	}

	// We need to handle both response and error
	type result struct {
		data []byte
		err  error
	}
	resCh := make(chan result, 1)

	removeResp := c.On(74, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTReadResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{data: resp.Data}
		}
	})
	defer removeResp()

	removeErr := c.On(82, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTErrorResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{err: fmt.Errorf("GATT error (code: %d)", resp.Error)}
		}
	})
	defer removeErr()

	if err := c.SendMessage(req, 73); err != nil {
		return nil, fmt.Errorf("BluetoothGATTRead: %w", err)
	}

	select {
	case res := <-resCh:
		return res.data, res.err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("BluetoothGATTRead: timeout")
	case <-c.Done():
		return nil, fmt.Errorf("BluetoothGATTRead: client closed")
	}
}

// BluetoothGATTWrite writes a GATT characteristic.
func (c *Client) BluetoothGATTWrite(address uint64, handle uint32, data []byte, response bool) error {
	req := &pb.BluetoothGATTWriteRequest{
		Address:  address,
		Handle:   handle,
		Data:     data,
		Response: response,
	}

	if !response {
		return c.SendMessage(req, 75)
	}

	// Wait for response
	type result struct {
		err error
	}
	resCh := make(chan result, 1)

	removeResp := c.On(83, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTWriteResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{}
		}
	})
	defer removeResp()

	removeErr := c.On(82, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTErrorResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{err: fmt.Errorf("GATT error (code: %d)", resp.Error)}
		}
	})
	defer removeErr()

	if err := c.SendMessage(req, 75); err != nil {
		return fmt.Errorf("BluetoothGATTWrite: %w", err)
	}

	select {
	case res := <-resCh:
		return res.err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("BluetoothGATTWrite: timeout")
	case <-c.Done():
		return fmt.Errorf("BluetoothGATTWrite: client closed")
	}
}

// BluetoothGATTNotify enables or disables GATT notifications for a characteristic.
func (c *Client) BluetoothGATTNotify(address uint64, handle uint32, enable bool, handler func([]byte)) (func(), error) {
	req := &pb.BluetoothGATTNotifyRequest{
		Address: address,
		Handle:  handle,
		Enable:  enable,
	}

	type result struct {
		err error
	}
	resCh := make(chan result, 1)

	removeResp := c.On(84, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTNotifyResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{}
		}
	})
	defer removeResp()

	removeErr := c.On(82, func(msg proto.Message) {
		resp := msg.(*pb.BluetoothGATTErrorResponse)
		if resp.Address == address && resp.Handle == handle {
			resCh <- result{err: fmt.Errorf("GATT error (code: %d)", resp.Error)}
		}
	})
	defer removeErr()

	var removeData func()
	if enable && handler != nil {
		removeData = c.On(79, func(msg proto.Message) {
			resp := msg.(*pb.BluetoothGATTNotifyDataResponse)
			if resp.Address == address && resp.Handle == handle {
				handler(resp.Data)
			}
		})
	}

	if err := c.SendMessage(req, 78); err != nil {
		if removeData != nil {
			removeData()
		}
		return nil, fmt.Errorf("BluetoothGATTNotify: %w", err)
	}

	select {
	case res := <-resCh:
		if res.err != nil {
			if removeData != nil {
				removeData()
			}
			return nil, res.err
		}
		return removeData, nil
	case <-time.After(10 * time.Second):
		if removeData != nil {
			removeData()
		}
		return nil, fmt.Errorf("BluetoothGATTNotify: timeout")
	case <-c.Done():
		if removeData != nil {
			removeData()
		}
		return nil, fmt.Errorf("BluetoothGATTNotify: client closed")
	}
}

// SubscribeBluetoothConnectionsFree subscribes to updates on free Bluetooth connection slots.
func (c *Client) SubscribeBluetoothConnectionsFree(handler func(*pb.BluetoothConnectionsFreeResponse)) (func(), error) {
	remove := c.On(81, func(msg proto.Message) {
		if handler != nil {
			handler(msg.(*pb.BluetoothConnectionsFreeResponse))
		}
	})

	if err := c.SendMessage(&pb.SubscribeBluetoothConnectionsFreeRequest{}, 80); err != nil {
		remove()
		return nil, fmt.Errorf("SubscribeBluetoothConnectionsFree: %w", err)
	}

	return remove, nil
}

// BluetoothScannerSetMode sets the Bluetooth scanner mode.
func (c *Client) BluetoothScannerSetMode(mode pb.BluetoothScannerMode) error {
	req := &pb.BluetoothScannerSetModeRequest{
		Mode: mode,
	}

	if err := c.SendMessage(req, 127); err != nil {
		return fmt.Errorf("BluetoothScannerSetMode: %w", err)
	}

	return nil
}
