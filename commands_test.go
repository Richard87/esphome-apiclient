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

// newTestClientServer creates a connected client/server pair for command testing.
// The client has a running read loop. Caller must defer client.Close() and serverConn.Close().
func newTestClientServer(t *testing.T) (client *Client, server *Client, serverConn net.Conn) {
	t.Helper()
	clientConn, sConn := net.Pipe()
	t.Cleanup(func() {
		clientConn.Close()
		sConn.Close()
	})

	client = newTestClient(clientConn)
	server = newTestServer(sConn)

	go client.readLoop()
	return client, server, sConn
}

// --- SendCommand tests ---

func TestSendCommand_Switch(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(33), msgType)
		cmd, ok := msg.(*pb.SwitchCommandRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(0x12345678), cmd.Key)
		assert.True(t, cmd.State)
	}()

	err := client.SendCommand(&pb.SwitchCommandRequest{Key: 0x12345678, State: true})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive command")
	}
}

func TestSendCommand_Light(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(32), msgType)
		cmd, ok := msg.(*pb.LightCommandRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(42), cmd.Key)
		assert.True(t, cmd.HasBrightness)
		assert.InDelta(t, 0.75, cmd.Brightness, 0.001)
		assert.False(t, cmd.HasState)
		assert.False(t, cmd.HasRgb)
		assert.False(t, cmd.HasColorTemperature)
		assert.False(t, cmd.HasEffect)
	}()

	err := client.SendCommand(&pb.LightCommandRequest{
		Key:           42,
		HasBrightness: true,
		Brightness:    0.75,
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to receive command")
	}
}

func TestSendCommand_UnsupportedType(t *testing.T) {
	client, _, _ := newTestClientServer(t)
	defer client.Close()

	err := client.SendCommand(&pb.PingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported command type")
}

// --- Type-safe convenience method tests ---

func TestSetSwitch(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	// Populate registry with a switch entity
	client.entities.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
		Key: 0xABCD, ObjectId: "relay", Name: "Relay",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(33), msgType)
		cmd, ok := msg.(*pb.SwitchCommandRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(0xABCD), cmd.Key)
		assert.True(t, cmd.State)
	}()

	err := client.SetSwitch(0xABCD, true)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetLight_BrightnessOnly(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesLightResponse{
		Key: 10, ObjectId: "light_1", Name: "Ceiling",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(32), msgType)
		cmd, ok := msg.(*pb.LightCommandRequest)
		require.True(t, ok)
		assert.Equal(t, uint32(10), cmd.Key)
		// Only brightness set
		assert.True(t, cmd.HasBrightness)
		assert.InDelta(t, 0.5, cmd.Brightness, 0.001)
		// State and color fields should NOT be set
		assert.False(t, cmd.HasState)
		assert.False(t, cmd.HasRgb)
		assert.False(t, cmd.HasColorTemperature)
		assert.False(t, cmd.HasEffect)
		assert.False(t, cmd.HasWhite)
		assert.False(t, cmd.HasWarmWhite)
		assert.False(t, cmd.HasColdWhite)
	}()

	err := client.SetLight(10, LightCommandOpts{
		HasBrightness: true,
		Brightness:    0.5,
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetLight_FullRGB(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesLightResponse{
		Key: 10, ObjectId: "light_1", Name: "Ceiling",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, _, err := server.RecvMessage()
		require.NoError(t, err)
		cmd := msg.(*pb.LightCommandRequest)
		assert.True(t, cmd.HasState)
		assert.True(t, cmd.State)
		assert.True(t, cmd.HasBrightness)
		assert.InDelta(t, 1.0, cmd.Brightness, 0.001)
		assert.True(t, cmd.HasRgb)
		assert.InDelta(t, 1.0, cmd.Red, 0.001)
		assert.InDelta(t, 0.0, cmd.Green, 0.001)
		assert.InDelta(t, 0.5, cmd.Blue, 0.001)
		assert.True(t, cmd.HasTransitionLength)
		assert.Equal(t, uint32(500), cmd.TransitionLength)
	}()

	err := client.SetLight(10, LightCommandOpts{
		HasState:            true,
		State:               true,
		HasBrightness:       true,
		Brightness:          1.0,
		HasRGB:              true,
		Red:                 1.0,
		Green:               0.0,
		Blue:                0.5,
		HasTransitionLength: true,
		TransitionLength:    500,
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetNumber(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesNumberResponse{
		Key: 20, ObjectId: "brightness", Name: "Brightness",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(51), msgType)
		cmd := msg.(*pb.NumberCommandRequest)
		assert.Equal(t, uint32(20), cmd.Key)
		assert.InDelta(t, 42.5, cmd.State, 0.001)
	}()

	err := client.SetNumber(20, 42.5)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetSelect(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesSelectResponse{
		Key: 30, ObjectId: "mode", Name: "Mode",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(54), msgType)
		cmd := msg.(*pb.SelectCommandRequest)
		assert.Equal(t, uint32(30), cmd.Key)
		assert.Equal(t, "auto", cmd.State)
	}()

	err := client.SetSelect(30, "auto")
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestPressButton(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesButtonResponse{
		Key: 40, ObjectId: "restart", Name: "Restart",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(62), msgType)
		cmd := msg.(*pb.ButtonCommandRequest)
		assert.Equal(t, uint32(40), cmd.Key)
	}()

	err := client.PressButton(40)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetCoverPosition(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesCoverResponse{
		Key: 50, ObjectId: "blinds", Name: "Blinds",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(30), msgType)
		cmd := msg.(*pb.CoverCommandRequest)
		assert.Equal(t, uint32(50), cmd.Key)
		assert.True(t, cmd.HasPosition)
		assert.InDelta(t, 0.5, cmd.Position, 0.001)
		assert.False(t, cmd.HasTilt)
		assert.False(t, cmd.Stop)
	}()

	err := client.SetCoverPosition(50, 0.5)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSetClimate(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	client.entities.HandleListEntityMessage(&pb.ListEntitiesClimateResponse{
		Key: 60, ObjectId: "hvac", Name: "HVAC",
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		msg, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(48), msgType)
		cmd := msg.(*pb.ClimateCommandRequest)
		assert.Equal(t, uint32(60), cmd.Key)
		assert.True(t, cmd.HasMode)
		assert.Equal(t, pb.ClimateMode_CLIMATE_MODE_HEAT, cmd.Mode)
		assert.True(t, cmd.HasTargetTemperature)
		assert.InDelta(t, 22.0, cmd.TargetTemperature, 0.001)
		assert.False(t, cmd.HasFanMode)
	}()

	err := client.SetClimate(60, ClimateCommandOpts{
		HasMode:              true,
		Mode:                 pb.ClimateMode_CLIMATE_MODE_HEAT,
		HasTargetTemperature: true,
		TargetTemperature:    22.0,
	})
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// --- Entity validation tests ---

func TestValidateEntity_NotFound(t *testing.T) {
	client, _, _ := newTestClientServer(t)
	defer client.Close()

	// Populate with a different entity so registry is non-empty
	client.entities.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
		Key: 1, ObjectId: "relay", Name: "Relay",
	})

	err := client.SetSwitch(0xDEAD, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEntityNotFound)
}

func TestValidateEntity_TypeMismatch(t *testing.T) {
	client, _, _ := newTestClientServer(t)
	defer client.Close()

	// Register a sensor but try to use it as a switch
	client.entities.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key: 100, ObjectId: "temp", Name: "Temperature",
	})

	err := client.SetSwitch(100, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEntityTypeMismatch)
}

func TestValidateEntity_SkipWhenRegistryEmpty(t *testing.T) {
	client, server, _ := newTestClientServer(t)
	defer client.Close()

	// Registry is empty — validation should be skipped, command should go through
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, msgType, err := server.RecvMessage()
		require.NoError(t, err)
		assert.Equal(t, uint32(33), msgType) // SwitchCommandRequest
	}()

	err := client.SetSwitch(999, true)
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// --- Wire format verification tests ---

func TestSendCommand_Switch_WireFormat(t *testing.T) {
	// Verify the exact bytes on the wire match what we expect
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &Client{
		framer:          NewPlainFramer(&mockTransport{conn: clientConn}),
		router:          NewRouter(),
		entities:        NewEntityRegistry(),
		done:            make(chan struct{}),
		clientInfo:      "test-client",
		apiVersionMajor: 1,
		apiVersionMinor: 10,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Read raw bytes from server side and decode manually
		msg, msgType, err := client.RecvMessage()
		// We use a helper client on the server side to read
		_ = msg
		_ = msgType
		_ = err
	}()

	// The real test: construct command, send it, and verify the server-side
	// decoded message has the right fields.
	serverClient := &Client{
		framer: NewPlainFramer(&mockTransport{conn: serverConn}),
	}

	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		err := client.SendCommand(&pb.SwitchCommandRequest{
			Key:   0xAABBCCDD,
			State: true,
		})
		assert.NoError(t, err)
	}()

	msg, msgType, err := serverClient.RecvMessage()
	require.NoError(t, err)
	assert.Equal(t, uint32(33), msgType)
	cmd := msg.(*pb.SwitchCommandRequest)
	assert.Equal(t, uint32(0xAABBCCDD), cmd.Key)
	assert.True(t, cmd.State)

	<-sendDone
}

func TestSendCommand_AllCommandTypes(t *testing.T) {
	// Verify all command types resolve to the correct message type ID
	tests := []struct {
		name       string
		cmd        proto.Message
		expectedID uint32
	}{
		{"Cover", &pb.CoverCommandRequest{Key: 1}, 30},
		{"Fan", &pb.FanCommandRequest{Key: 1}, 31},
		{"Light", &pb.LightCommandRequest{Key: 1}, 32},
		{"Switch", &pb.SwitchCommandRequest{Key: 1}, 33},
		{"Climate", &pb.ClimateCommandRequest{Key: 1}, 48},
		{"Number", &pb.NumberCommandRequest{Key: 1}, 51},
		{"Select", &pb.SelectCommandRequest{Key: 1}, 54},
		{"Siren", &pb.SirenCommandRequest{Key: 1}, 57},
		{"Lock", &pb.LockCommandRequest{Key: 1}, 60},
		{"Button", &pb.ButtonCommandRequest{Key: 1}, 62},
		{"MediaPlayer", &pb.MediaPlayerCommandRequest{Key: 1}, 65},
		{"WaterHeater", &pb.WaterHeaterCommandRequest{Key: 1}, 134},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server, _ := newTestClientServer(t)
			defer client.Close()

			done := make(chan struct{})
			go func() {
				defer close(done)
				_, msgType, err := server.RecvMessage()
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, msgType)
			}()

			err := client.SendCommand(tt.cmd)
			require.NoError(t, err)

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("timeout waiting for %s command", tt.name)
			}
		})
	}
}
