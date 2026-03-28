package esphome_apiclient

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// ---- Population & Query Tests ----

func TestEntityRegistry_PopulateSensors(t *testing.T) {
	reg := NewEntityRegistry()

	handled := reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key:               0x1001,
		ObjectId:          "temperature",
		Name:              "Temperature",
		Icon:              "mdi:thermometer",
		UnitOfMeasurement: "°C",
		AccuracyDecimals:  1,
		DeviceClass:       "temperature",
		StateClass:        pb.SensorStateClass_STATE_CLASS_MEASUREMENT,
		EntityCategory:    pb.EntityCategory_ENTITY_CATEGORY_NONE,
		DeviceId:          42,
	})
	assert.True(t, handled)

	sensors := reg.Sensors()
	require.Len(t, sensors, 1)
	s := sensors[0]
	assert.Equal(t, uint32(0x1001), s.Key)
	assert.Equal(t, "temperature", s.ObjectID)
	assert.Equal(t, "Temperature", s.Name)
	assert.Equal(t, "°C", s.UnitOfMeasurement)
	assert.Equal(t, int32(1), s.AccuracyDecimals)
	assert.Equal(t, "temperature", s.DeviceClass)
	assert.Equal(t, pb.SensorStateClass_STATE_CLASS_MEASUREMENT, s.StateClass)
	assert.True(t, s.MissingState, "should be missing_state=true before any state update")
}

func TestEntityRegistry_PopulateMultipleTypes(t *testing.T) {
	reg := NewEntityRegistry()

	// Add 3 sensors
	for i := uint32(1); i <= 3; i++ {
		reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
			Key:  i,
			Name: "sensor_" + string(rune('a'+i-1)),
		})
	}

	// Add 2 switches
	for i := uint32(100); i <= 101; i++ {
		reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
			Key:  i,
			Name: "switch_" + string(rune('a'+i-100)),
		})
	}

	// Add 1 binary sensor
	reg.HandleListEntityMessage(&pb.ListEntitiesBinarySensorResponse{
		Key:  200,
		Name: "motion",
	})

	assert.Len(t, reg.Sensors(), 3)
	assert.Len(t, reg.Switches(), 2)
	assert.Len(t, reg.BinarySensors(), 1)
	assert.Equal(t, 6, reg.Len())
}

func TestEntityRegistry_ByKey(t *testing.T) {
	reg := NewEntityRegistry()

	reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key:  0xABCD,
		Name: "Temperature",
	})
	reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
		Key:  0x1234,
		Name: "Relay",
	})

	// Found
	e := reg.ByKey(0xABCD)
	require.NotNil(t, e)
	assert.Equal(t, "Temperature", e.GetName())
	assert.Equal(t, DomainSensor, e.GetDomain())

	e2 := reg.ByKey(0x1234)
	require.NotNil(t, e2)
	assert.Equal(t, "Relay", e2.GetName())
	assert.Equal(t, DomainSwitch, e2.GetDomain())

	// Not found
	assert.Nil(t, reg.ByKey(0x9999))
}

func TestEntityRegistry_ByName(t *testing.T) {
	reg := NewEntityRegistry()

	reg.HandleListEntityMessage(&pb.ListEntitiesLightResponse{
		Key:  1,
		Name: "Living Room Light",
	})

	e := reg.ByName("Living Room Light")
	require.NotNil(t, e)
	assert.Equal(t, uint32(1), e.GetKey())
	assert.Equal(t, DomainLight, e.GetDomain())

	assert.Nil(t, reg.ByName("Nonexistent"))
}

func TestEntityRegistry_UnknownMessage(t *testing.T) {
	reg := NewEntityRegistry()

	// A non-entity message should not be handled
	handled := reg.HandleListEntityMessage(&pb.PingRequest{})
	assert.False(t, handled)
	assert.Equal(t, 0, reg.Len())
}

// ---- All Entity Domain Population Tests ----

func TestEntityRegistry_PopulateBinarySensor(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesBinarySensorResponse{
		Key:         1,
		Name:        "Motion",
		ObjectId:    "motion",
		DeviceClass: "motion",
	})
	entities := reg.BinarySensors()
	require.Len(t, entities, 1)
	assert.Equal(t, "Motion", entities[0].Name)
	assert.Equal(t, "motion", entities[0].DeviceClass)
	assert.True(t, entities[0].MissingState)
}

func TestEntityRegistry_PopulateCover(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesCoverResponse{
		Key:              1,
		Name:             "Garage Door",
		SupportsPosition: true,
		SupportsTilt:     false,
		SupportsStop:     true,
	})
	entities := reg.Covers()
	require.Len(t, entities, 1)
	assert.True(t, entities[0].SupportsPosition)
	assert.False(t, entities[0].SupportsTilt)
	assert.True(t, entities[0].SupportsStop)
}

func TestEntityRegistry_PopulateFan(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesFanResponse{
		Key:                  1,
		Name:                 "Ceiling Fan",
		SupportsOscillation:  true,
		SupportsDirection:    true,
		SupportedSpeedCount:  4,
		SupportedPresetModes: []string{"eco", "sleep"},
	})
	fans := reg.Fans()
	require.Len(t, fans, 1)
	assert.True(t, fans[0].SupportsOscillation)
	assert.Equal(t, int32(4), fans[0].SupportedSpeedCount)
	assert.Equal(t, []string{"eco", "sleep"}, fans[0].SupportedPresetModes)
}

func TestEntityRegistry_PopulateLight(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesLightResponse{
		Key:       1,
		Name:      "Desk Lamp",
		MinMireds: 153,
		MaxMireds: 500,
		Effects:   []string{"rainbow", "strobe"},
	})
	lights := reg.Lights()
	require.Len(t, lights, 1)
	assert.Equal(t, float32(153), lights[0].MinMireds)
	assert.Equal(t, float32(500), lights[0].MaxMireds)
	assert.Equal(t, []string{"rainbow", "strobe"}, lights[0].Effects)
}

func TestEntityRegistry_PopulateSwitch(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
		Key:          1,
		Name:         "Relay",
		AssumedState: true,
	})
	switches := reg.Switches()
	require.Len(t, switches, 1)
	assert.True(t, switches[0].AssumedState)
}

func TestEntityRegistry_PopulateTextSensor(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesTextSensorResponse{
		Key:  1,
		Name: "WiFi SSID",
	})
	entities := reg.TextSensors()
	require.Len(t, entities, 1)
	assert.Equal(t, "WiFi SSID", entities[0].Name)
	assert.True(t, entities[0].MissingState)
}

func TestEntityRegistry_PopulateCamera(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesCameraResponse{
		Key:  1,
		Name: "Front Door Camera",
	})
	cameras := reg.Cameras()
	require.Len(t, cameras, 1)
	assert.Equal(t, "Front Door Camera", cameras[0].Name)
}

func TestEntityRegistry_PopulateClimate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesClimateResponse{
		Key:                  1,
		Name:                 "HVAC",
		VisualMinTemperature: 16,
		VisualMaxTemperature: 30,
		SupportedModes:       []pb.ClimateMode{pb.ClimateMode_CLIMATE_MODE_OFF, pb.ClimateMode_CLIMATE_MODE_HEAT, pb.ClimateMode_CLIMATE_MODE_COOL},
	})
	climates := reg.Climates()
	require.Len(t, climates, 1)
	assert.Equal(t, float32(16), climates[0].VisualMinTemperature)
	assert.Equal(t, float32(30), climates[0].VisualMaxTemperature)
	assert.Len(t, climates[0].SupportedModes, 3)
}

func TestEntityRegistry_PopulateNumber(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesNumberResponse{
		Key:               1,
		Name:              "Volume",
		MinValue:          0,
		MaxValue:          100,
		Step:              1,
		UnitOfMeasurement: "%",
	})
	numbers := reg.Numbers()
	require.Len(t, numbers, 1)
	assert.Equal(t, float32(0), numbers[0].MinValue)
	assert.Equal(t, float32(100), numbers[0].MaxValue)
	assert.True(t, numbers[0].MissingState)
}

func TestEntityRegistry_PopulateSelect(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSelectResponse{
		Key:     1,
		Name:    "Mode",
		Options: []string{"auto", "manual", "eco"},
	})
	selects := reg.Selects()
	require.Len(t, selects, 1)
	assert.Equal(t, []string{"auto", "manual", "eco"}, selects[0].Options)
	assert.True(t, selects[0].MissingState)
}

func TestEntityRegistry_PopulateSiren(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSirenResponse{
		Key:   1,
		Name:  "Alarm",
		Tones: []string{"beep", "siren"},
	})
	sirens := reg.Sirens()
	require.Len(t, sirens, 1)
	assert.Equal(t, []string{"beep", "siren"}, sirens[0].Tones)
}

func TestEntityRegistry_PopulateLock(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesLockResponse{
		Key:          1,
		Name:         "Front Door",
		SupportsOpen: true,
		RequiresCode: true,
		CodeFormat:   `^\d{4}$`,
	})
	locks := reg.Locks()
	require.Len(t, locks, 1)
	assert.True(t, locks[0].SupportsOpen)
	assert.True(t, locks[0].RequiresCode)
	assert.Equal(t, `^\d{4}$`, locks[0].CodeFormat)
}

func TestEntityRegistry_PopulateButton(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesButtonResponse{
		Key:         1,
		Name:        "Restart",
		DeviceClass: "restart",
	})
	buttons := reg.Buttons()
	require.Len(t, buttons, 1)
	assert.Equal(t, "Restart", buttons[0].Name)
	assert.Equal(t, "restart", buttons[0].DeviceClass)
}

func TestEntityRegistry_PopulateMediaPlayer(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesMediaPlayerResponse{
		Key:           1,
		Name:          "Speaker",
		SupportsPause: true,
	})
	players := reg.MediaPlayers()
	require.Len(t, players, 1)
	assert.True(t, players[0].SupportsPause)
}

func TestEntityRegistry_PopulateWaterHeater(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesWaterHeaterResponse{
		Key:            1,
		Name:           "Boiler",
		MinTemperature: 30,
		MaxTemperature: 80,
	})
	heaters := reg.WaterHeaters()
	require.Len(t, heaters, 1)
	assert.Equal(t, float32(30), heaters[0].MinTemperature)
	assert.Equal(t, float32(80), heaters[0].MaxTemperature)
}

// ---- State Update Tests ----

func TestEntityRegistry_SensorStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()

	reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key:  1,
		Name: "Temperature",
	})

	// Initially missing
	sensors := reg.Sensors()
	require.Len(t, sensors, 1)
	assert.True(t, sensors[0].MissingState)

	// Update state
	handled := reg.HandleStateMessage(&pb.SensorStateResponse{
		Key:          1,
		State:        22.5,
		MissingState: false,
	})
	assert.True(t, handled)

	sensors = reg.Sensors()
	assert.Equal(t, float32(22.5), sensors[0].State)
	assert.False(t, sensors[0].MissingState)
}

func TestEntityRegistry_SensorStateMissing(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{Key: 1, Name: "T"})

	// First: normal state
	reg.HandleStateMessage(&pb.SensorStateResponse{Key: 1, State: 20.0, MissingState: false})
	assert.False(t, reg.Sensors()[0].MissingState)
	assert.Equal(t, float32(20.0), reg.Sensors()[0].State)

	// Second: missing state
	reg.HandleStateMessage(&pb.SensorStateResponse{Key: 1, State: 0, MissingState: true})
	assert.True(t, reg.Sensors()[0].MissingState)
}

func TestEntityRegistry_BinarySensorStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesBinarySensorResponse{Key: 1, Name: "Motion"})

	reg.HandleStateMessage(&pb.BinarySensorStateResponse{Key: 1, State: true, MissingState: false})
	assert.True(t, reg.BinarySensors()[0].State)
	assert.False(t, reg.BinarySensors()[0].MissingState)

	reg.HandleStateMessage(&pb.BinarySensorStateResponse{Key: 1, State: false})
	assert.False(t, reg.BinarySensors()[0].State)
}

func TestEntityRegistry_CoverStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesCoverResponse{Key: 1, Name: "Garage"})

	reg.HandleStateMessage(&pb.CoverStateResponse{
		Key:              1,
		Position:         0.75,
		Tilt:             0.5,
		CurrentOperation: pb.CoverOperation_COVER_OPERATION_IS_OPENING,
	})
	cover := reg.Covers()[0]
	assert.Equal(t, float32(0.75), cover.Position)
	assert.Equal(t, float32(0.5), cover.Tilt)
	assert.Equal(t, pb.CoverOperation_COVER_OPERATION_IS_OPENING, cover.CurrentOperation)
}

func TestEntityRegistry_FanStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesFanResponse{Key: 1, Name: "Fan"})

	reg.HandleStateMessage(&pb.FanStateResponse{
		Key:         1,
		State:       true,
		Oscillating: true,
		Direction:   pb.FanDirection_FAN_DIRECTION_REVERSE,
		SpeedLevel:  3,
		PresetMode:  "eco",
	})
	fan := reg.Fans()[0]
	assert.True(t, fan.State)
	assert.True(t, fan.Oscillating)
	assert.Equal(t, pb.FanDirection_FAN_DIRECTION_REVERSE, fan.Direction)
	assert.Equal(t, int32(3), fan.SpeedLevel)
	assert.Equal(t, "eco", fan.PresetMode)
}

func TestEntityRegistry_LightStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesLightResponse{Key: 1, Name: "Lamp"})

	reg.HandleStateMessage(&pb.LightStateResponse{
		Key:              1,
		State:            true,
		Brightness:       0.8,
		Red:              1.0,
		Green:            0.5,
		Blue:             0.0,
		ColorTemperature: 300,
		Effect:           "rainbow",
	})
	light := reg.Lights()[0]
	assert.True(t, light.State)
	assert.Equal(t, float32(0.8), light.Brightness)
	assert.Equal(t, float32(1.0), light.Red)
	assert.Equal(t, float32(0.5), light.Green)
	assert.Equal(t, float32(0.0), light.Blue)
	assert.Equal(t, float32(300), light.ColorTemperature)
	assert.Equal(t, "rainbow", light.Effect)
}

func TestEntityRegistry_SwitchStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{Key: 1, Name: "Relay"})

	reg.HandleStateMessage(&pb.SwitchStateResponse{Key: 1, State: true})
	assert.True(t, reg.Switches()[0].State)

	reg.HandleStateMessage(&pb.SwitchStateResponse{Key: 1, State: false})
	assert.False(t, reg.Switches()[0].State)
}

func TestEntityRegistry_TextSensorStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesTextSensorResponse{Key: 1, Name: "WiFi"})

	reg.HandleStateMessage(&pb.TextSensorStateResponse{Key: 1, State: "MyNetwork", MissingState: false})
	ts := reg.TextSensors()[0]
	assert.Equal(t, "MyNetwork", ts.State)
	assert.False(t, ts.MissingState)
}

func TestEntityRegistry_CameraStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesCameraResponse{Key: 1, Name: "Cam"})

	// First chunk
	reg.HandleStateMessage(&pb.CameraImageResponse{Key: 1, Data: []byte{0xFF, 0xD8}, Done: false})
	cam := reg.Cameras()[0]
	assert.Equal(t, []byte{0xFF, 0xD8}, cam.ImageData)
	assert.False(t, cam.ImageDone)

	// Second chunk
	reg.HandleStateMessage(&pb.CameraImageResponse{Key: 1, Data: []byte{0xFF, 0xD9}, Done: true})
	cam = reg.Cameras()[0]
	assert.Equal(t, []byte{0xFF, 0xD8, 0xFF, 0xD9}, cam.ImageData)
	assert.True(t, cam.ImageDone)

	// New frame after done — should reset
	reg.HandleStateMessage(&pb.CameraImageResponse{Key: 1, Data: []byte{0xAA}, Done: false})
	cam = reg.Cameras()[0]
	assert.Equal(t, []byte{0xAA}, cam.ImageData)
	assert.False(t, cam.ImageDone)
}

func TestEntityRegistry_ClimateStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesClimateResponse{Key: 1, Name: "HVAC"})

	reg.HandleStateMessage(&pb.ClimateStateResponse{
		Key:                1,
		Mode:               pb.ClimateMode_CLIMATE_MODE_HEAT,
		CurrentTemperature: 21.5,
		TargetTemperature:  24.0,
		Action:             pb.ClimateAction_CLIMATE_ACTION_HEATING,
	})
	climate := reg.Climates()[0]
	assert.Equal(t, pb.ClimateMode_CLIMATE_MODE_HEAT, climate.Mode)
	assert.Equal(t, float32(21.5), climate.CurrentTemperature)
	assert.Equal(t, float32(24.0), climate.TargetTemperature)
	assert.Equal(t, pb.ClimateAction_CLIMATE_ACTION_HEATING, climate.Action)
}

func TestEntityRegistry_NumberStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesNumberResponse{Key: 1, Name: "Vol"})

	reg.HandleStateMessage(&pb.NumberStateResponse{Key: 1, State: 75, MissingState: false})
	assert.Equal(t, float32(75), reg.Numbers()[0].State)
	assert.False(t, reg.Numbers()[0].MissingState)
}

func TestEntityRegistry_SelectStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSelectResponse{Key: 1, Name: "Mode"})

	reg.HandleStateMessage(&pb.SelectStateResponse{Key: 1, State: "eco", MissingState: false})
	assert.Equal(t, "eco", reg.Selects()[0].State)
	assert.False(t, reg.Selects()[0].MissingState)
}

func TestEntityRegistry_SirenStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesSirenResponse{Key: 1, Name: "Alarm"})

	reg.HandleStateMessage(&pb.SirenStateResponse{Key: 1, State: true})
	assert.True(t, reg.Sirens()[0].State)
}

func TestEntityRegistry_LockStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesLockResponse{Key: 1, Name: "Lock"})

	reg.HandleStateMessage(&pb.LockStateResponse{Key: 1, State: pb.LockState_LOCK_STATE_LOCKED})
	assert.Equal(t, pb.LockState_LOCK_STATE_LOCKED, reg.Locks()[0].State)
}

func TestEntityRegistry_MediaPlayerStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesMediaPlayerResponse{Key: 1, Name: "Speaker"})

	reg.HandleStateMessage(&pb.MediaPlayerStateResponse{
		Key:    1,
		State:  pb.MediaPlayerState_MEDIA_PLAYER_STATE_PLAYING,
		Volume: 0.6,
		Muted:  false,
	})
	mp := reg.MediaPlayers()[0]
	assert.Equal(t, pb.MediaPlayerState_MEDIA_PLAYER_STATE_PLAYING, mp.State)
	assert.Equal(t, float32(0.6), mp.Volume)
	assert.False(t, mp.Muted)
}

func TestEntityRegistry_WaterHeaterStateUpdate(t *testing.T) {
	reg := NewEntityRegistry()
	reg.HandleListEntityMessage(&pb.ListEntitiesWaterHeaterResponse{Key: 1, Name: "Boiler"})

	reg.HandleStateMessage(&pb.WaterHeaterStateResponse{
		Key:                1,
		CurrentTemperature: 55,
		TargetTemperature:  60,
		Mode:               pb.WaterHeaterMode_WATER_HEATER_MODE_HEAT_PUMP,
	})
	wh := reg.WaterHeaters()[0]
	assert.Equal(t, float32(55), wh.CurrentTemperature)
	assert.Equal(t, float32(60), wh.TargetTemperature)
	assert.Equal(t, pb.WaterHeaterMode_WATER_HEATER_MODE_HEAT_PUMP, wh.Mode)
}

// ---- State update for unknown key ----

func TestEntityRegistry_StateUpdateUnknownKey(t *testing.T) {
	reg := NewEntityRegistry()

	// No entities registered — state update for unknown key should return false
	handled := reg.HandleStateMessage(&pb.SensorStateResponse{Key: 999, State: 42})
	assert.False(t, handled)

	handled = reg.HandleStateMessage(&pb.SwitchStateResponse{Key: 999, State: true})
	assert.False(t, handled)
}

// ---- State update for unhandled message type ----

func TestEntityRegistry_StateUpdateUnhandledMessage(t *testing.T) {
	reg := NewEntityRegistry()

	handled := reg.HandleStateMessage(&pb.PingResponse{})
	assert.False(t, handled)
}

// ---- Clear Test ----

func TestEntityRegistry_Clear(t *testing.T) {
	reg := NewEntityRegistry()

	reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{Key: 1, Name: "T"})
	reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{Key: 2, Name: "S"})
	assert.Equal(t, 2, reg.Len())

	reg.Clear()
	assert.Equal(t, 0, reg.Len())
	assert.Empty(t, reg.Sensors())
	assert.Empty(t, reg.Switches())
	assert.Nil(t, reg.ByKey(1))
	assert.Nil(t, reg.ByName("T"))
}

// ---- Entity Interface Tests ----

func TestEntityDomainString(t *testing.T) {
	tests := []struct {
		domain EntityDomain
		want   string
	}{
		{DomainSensor, "sensor"},
		{DomainBinarySensor, "binary_sensor"},
		{DomainCover, "cover"},
		{DomainFan, "fan"},
		{DomainLight, "light"},
		{DomainSwitch, "switch"},
		{DomainTextSensor, "text_sensor"},
		{DomainCamera, "camera"},
		{DomainClimate, "climate"},
		{DomainNumber, "number"},
		{DomainSelect, "select"},
		{DomainSiren, "siren"},
		{DomainLock, "lock"},
		{DomainButton, "button"},
		{DomainMediaPlayer, "media_player"},
		{DomainWaterHeater, "water_heater"},
		{EntityDomain(99), "unknown(99)"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.domain.String())
	}
}

func TestEntity_InterfaceMethods(t *testing.T) {
	reg := NewEntityRegistry()

	reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key:      0x42,
		ObjectId: "temp",
		Name:     "Temperature",
	})

	e := reg.ByKey(0x42)
	require.NotNil(t, e)
	assert.Equal(t, uint32(0x42), e.GetKey())
	assert.Equal(t, "Temperature", e.GetName())
	assert.Equal(t, "temp", e.GetObjectID())
	assert.Equal(t, DomainSensor, e.GetDomain())
}

// ---- Concurrent Access Tests ----

func TestEntityRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewEntityRegistry()

	// Pre-populate some entities
	for i := uint32(0); i < 100; i++ {
		reg.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
			Key:  i,
			Name: "sensor_" + string(rune(i)),
		})
	}

	var wg sync.WaitGroup

	// Writers: update state concurrently
	for i := uint32(0); i < 100; i++ {
		wg.Add(1)
		go func(key uint32) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				reg.HandleStateMessage(&pb.SensorStateResponse{
					Key:   key,
					State: float32(j),
				})
			}
		}(i)
	}

	// Readers: read state concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = reg.Sensors()
				_ = reg.ByKey(uint32(j))
				_ = reg.ByName("sensor_" + string(rune(j)))
				_ = reg.Len()
			}
		}()
	}

	// More writers: add new entities concurrently
	for i := uint32(200); i < 250; i++ {
		wg.Add(1)
		go func(key uint32) {
			defer wg.Done()
			reg.HandleListEntityMessage(&pb.ListEntitiesSwitchResponse{
				Key:  key,
				Name: "switch",
			})
		}(i)
	}

	wg.Wait()

	// Sanity check: all entities should be present
	assert.Equal(t, 150, reg.Len())
	assert.Len(t, reg.Sensors(), 100)
	assert.Len(t, reg.Switches(), 50)
}

// ---- Integration with Client ListEntities ----

func TestClient_ListEntities_PopulatesRegistry(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	go client.readLoop()

	go func() {
		// Read the ListEntitiesRequest from client
		_, _, err := server.RecvMessage()
		require.NoError(t, err)

		// Send entity responses
		server.SendMessage(&pb.ListEntitiesSensorResponse{
			Key:               0x1001,
			Name:              "Temperature",
			UnitOfMeasurement: "°C",
		}, 16)

		server.SendMessage(&pb.ListEntitiesSwitchResponse{
			Key:  0x2001,
			Name: "Relay",
		}, 17)

		server.SendMessage(&pb.ListEntitiesBinarySensorResponse{
			Key:  0x3001,
			Name: "Motion",
		}, 12)

		// Send ListEntitiesDoneResponse
		server.SendMessage(&pb.ListEntitiesDoneResponse{}, 19)
	}()

	msgs, err := client.ListEntitiesWithTimeout(5 * time.Second)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)

	// Verify registry was populated
	reg := client.Entities()
	assert.Equal(t, 3, reg.Len())

	sensors := reg.Sensors()
	require.Len(t, sensors, 1)
	assert.Equal(t, "Temperature", sensors[0].Name)

	switches := reg.Switches()
	require.Len(t, switches, 1)
	assert.Equal(t, "Relay", switches[0].Name)

	binarySensors := reg.BinarySensors()
	require.Len(t, binarySensors, 1)
	assert.Equal(t, "Motion", binarySensors[0].Name)
}

// ---- Integration with Client SubscribeStates ----

func TestClient_SubscribeStates_UpdatesRegistry(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := newTestClient(clientConn)
	server := newTestServer(serverConn)

	// Pre-populate registry
	client.entities.HandleListEntityMessage(&pb.ListEntitiesSensorResponse{
		Key:  0x1001,
		Name: "Temperature",
	})

	go client.readLoop()

	// Start server goroutine before SubscribeStates to avoid deadlock:
	// net.Pipe() is synchronous, so SendMessage in SubscribeStates blocks
	// until the server reads the SubscribeStatesRequest from its end.
	serverDone := make(chan error, 1)
	go func() {
		// Read the SubscribeStatesRequest from the server side
		_, _, err := server.RecvMessage()
		if err != nil {
			serverDone <- err
			return
		}

		// Server sends state update
		err = server.SendMessage(&pb.SensorStateResponse{
			Key:          0x1001,
			State:        23.5,
			MissingState: false,
		}, 25)
		serverDone <- err
	}()

	received := make(chan proto.Message, 1)
	_, err := client.SubscribeStates(func(msg proto.Message) {
		received <- msg
	})
	require.NoError(t, err)

	// Wait for server goroutine to finish
	require.NoError(t, <-serverDone)

	// Wait for handler
	select {
	case msg := <-received:
		resp, ok := msg.(*pb.SensorStateResponse)
		require.True(t, ok)
		assert.Equal(t, float32(23.5), resp.State)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for state update")
	}

	// Verify entity registry was updated
	sensors := client.Entities().Sensors()
	require.Len(t, sensors, 1)
	assert.Equal(t, float32(23.5), sensors[0].State)
	assert.False(t, sensors[0].MissingState)
}
