package esphome_apiclient

import (
	"fmt"

	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
)

// commandTypeIDs maps command request proto types to their message type IDs.
var commandTypeIDs = map[string]uint32{
	"CoverCommandRequest":       30,
	"FanCommandRequest":         31,
	"LightCommandRequest":       32,
	"SwitchCommandRequest":      33,
	"ClimateCommandRequest":     48,
	"NumberCommandRequest":      51,
	"SelectCommandRequest":      54,
	"SirenCommandRequest":       57,
	"LockCommandRequest":        60,
	"ButtonCommandRequest":      62,
	"MediaPlayerCommandRequest": 65,
	"WaterHeaterCommandRequest": 134,
}

// ErrEntityNotFound is returned when an entity key is not in the registry.
var ErrEntityNotFound = fmt.Errorf("entity not found")

// ErrEntityTypeMismatch is returned when an entity key exists but belongs to a different domain.
var ErrEntityTypeMismatch = fmt.Errorf("entity type mismatch")

// SendCommand sends a command request protobuf message to the device. It resolves
// the message type ID automatically from the proto type. It is safe to call from
// multiple goroutines.
func (c *Client) SendCommand(cmd proto.Message) error {
	var msgType uint32
	switch cmd.(type) {
	case *pb.CoverCommandRequest:
		msgType = 30
	case *pb.FanCommandRequest:
		msgType = 31
	case *pb.LightCommandRequest:
		msgType = 32
	case *pb.SwitchCommandRequest:
		msgType = 33
	case *pb.ClimateCommandRequest:
		msgType = 48
	case *pb.NumberCommandRequest:
		msgType = 51
	case *pb.SelectCommandRequest:
		msgType = 54
	case *pb.SirenCommandRequest:
		msgType = 57
	case *pb.LockCommandRequest:
		msgType = 60
	case *pb.ButtonCommandRequest:
		msgType = 62
	case *pb.MediaPlayerCommandRequest:
		msgType = 65
	case *pb.WaterHeaterCommandRequest:
		msgType = 134
	default:
		return fmt.Errorf("unsupported command type: %T", cmd)
	}
	return c.SendMessage(cmd, msgType)
}

// --- Type-safe convenience methods ---

// SetSwitch turns a switch entity on or off.
func (c *Client) SetSwitch(key uint32, state bool) error {
	if err := c.validateEntity(key, DomainSwitch); err != nil {
		return err
	}
	return c.SendCommand(&pb.SwitchCommandRequest{
		Key:   key,
		State: state,
	})
}

// LightCommandOpts holds optional parameters for a light command. Only fields
// whose corresponding Has* flag is true will be sent.
type LightCommandOpts struct {
	HasState            bool
	State               bool
	HasBrightness       bool
	Brightness          float32
	HasColorMode        bool
	ColorMode           pb.ColorMode
	HasColorBrightness  bool
	ColorBrightness     float32
	HasRGB              bool
	Red                 float32
	Green               float32
	Blue                float32
	HasWhite            bool
	White               float32
	HasColorTemperature bool
	ColorTemperature    float32
	HasColdWhite        bool
	ColdWhite           float32
	HasWarmWhite        bool
	WarmWhite           float32
	HasTransitionLength bool
	TransitionLength    uint32
	HasFlashLength      bool
	FlashLength         uint32
	HasEffect           bool
	Effect              string
}

// SetLight sends a light command with the given options.
func (c *Client) SetLight(key uint32, opts LightCommandOpts) error {
	if err := c.validateEntity(key, DomainLight); err != nil {
		return err
	}
	return c.SendCommand(&pb.LightCommandRequest{
		Key:                 key,
		HasState:            opts.HasState,
		State:               opts.State,
		HasBrightness:       opts.HasBrightness,
		Brightness:          opts.Brightness,
		HasColorMode:        opts.HasColorMode,
		ColorMode:           opts.ColorMode,
		HasColorBrightness:  opts.HasColorBrightness,
		ColorBrightness:     opts.ColorBrightness,
		HasRgb:              opts.HasRGB,
		Red:                 opts.Red,
		Green:               opts.Green,
		Blue:                opts.Blue,
		HasWhite:            opts.HasWhite,
		White:               opts.White,
		HasColorTemperature: opts.HasColorTemperature,
		ColorTemperature:    opts.ColorTemperature,
		HasColdWhite:        opts.HasColdWhite,
		ColdWhite:           opts.ColdWhite,
		HasWarmWhite:        opts.HasWarmWhite,
		WarmWhite:           opts.WarmWhite,
		HasTransitionLength: opts.HasTransitionLength,
		TransitionLength:    opts.TransitionLength,
		HasFlashLength:      opts.HasFlashLength,
		FlashLength:         opts.FlashLength,
		HasEffect:           opts.HasEffect,
		Effect:              opts.Effect,
	})
}

// ClimateCommandOpts holds optional parameters for a climate command.
type ClimateCommandOpts struct {
	HasMode                  bool
	Mode                     pb.ClimateMode
	HasTargetTemperature     bool
	TargetTemperature        float32
	HasTargetTemperatureLow  bool
	TargetTemperatureLow     float32
	HasTargetTemperatureHigh bool
	TargetTemperatureHigh    float32
	HasFanMode               bool
	FanMode                  pb.ClimateFanMode
	HasSwingMode             bool
	SwingMode                pb.ClimateSwingMode
	HasCustomFanMode         bool
	CustomFanMode            string
	HasPreset                bool
	Preset                   pb.ClimatePreset
	HasCustomPreset          bool
	CustomPreset             string
	HasTargetHumidity        bool
	TargetHumidity           float32
}

// SetClimate sends a climate command with the given options.
func (c *Client) SetClimate(key uint32, opts ClimateCommandOpts) error {
	if err := c.validateEntity(key, DomainClimate); err != nil {
		return err
	}
	return c.SendCommand(&pb.ClimateCommandRequest{
		Key:                      key,
		HasMode:                  opts.HasMode,
		Mode:                     opts.Mode,
		HasTargetTemperature:     opts.HasTargetTemperature,
		TargetTemperature:        opts.TargetTemperature,
		HasTargetTemperatureLow:  opts.HasTargetTemperatureLow,
		TargetTemperatureLow:     opts.TargetTemperatureLow,
		HasTargetTemperatureHigh: opts.HasTargetTemperatureHigh,
		TargetTemperatureHigh:    opts.TargetTemperatureHigh,
		HasFanMode:               opts.HasFanMode,
		FanMode:                  opts.FanMode,
		HasSwingMode:             opts.HasSwingMode,
		SwingMode:                opts.SwingMode,
		HasCustomFanMode:         opts.HasCustomFanMode,
		CustomFanMode:            opts.CustomFanMode,
		HasPreset:                opts.HasPreset,
		Preset:                   opts.Preset,
		HasCustomPreset:          opts.HasCustomPreset,
		CustomPreset:             opts.CustomPreset,
		HasTargetHumidity:        opts.HasTargetHumidity,
		TargetHumidity:           opts.TargetHumidity,
	})
}

// SetNumber sets the value of a number entity.
func (c *Client) SetNumber(key uint32, value float32) error {
	if err := c.validateEntity(key, DomainNumber); err != nil {
		return err
	}
	return c.SendCommand(&pb.NumberCommandRequest{
		Key:   key,
		State: value,
	})
}

// SetSelect sets the value of a select entity.
func (c *Client) SetSelect(key uint32, value string) error {
	if err := c.validateEntity(key, DomainSelect); err != nil {
		return err
	}
	return c.SendCommand(&pb.SelectCommandRequest{
		Key:   key,
		State: value,
	})
}

// PressButton triggers a button entity.
func (c *Client) PressButton(key uint32) error {
	if err := c.validateEntity(key, DomainButton); err != nil {
		return err
	}
	return c.SendCommand(&pb.ButtonCommandRequest{
		Key: key,
	})
}

// CoverCommandOpts holds optional parameters for a cover command.
type CoverCommandOpts struct {
	HasPosition bool
	Position    float32
	HasTilt     bool
	Tilt        float32
	Stop        bool
}

// SetCover sends a cover command with the given options.
func (c *Client) SetCover(key uint32, opts CoverCommandOpts) error {
	if err := c.validateEntity(key, DomainCover); err != nil {
		return err
	}
	return c.SendCommand(&pb.CoverCommandRequest{
		Key:         key,
		HasPosition: opts.HasPosition,
		Position:    opts.Position,
		HasTilt:     opts.HasTilt,
		Tilt:        opts.Tilt,
		Stop:        opts.Stop,
	})
}

// SetCoverPosition is a convenience for setting only the cover position (0.0 = closed, 1.0 = open).
func (c *Client) SetCoverPosition(key uint32, position float32) error {
	return c.SetCover(key, CoverCommandOpts{HasPosition: true, Position: position})
}

// FanCommandOpts holds optional parameters for a fan command.
type FanCommandOpts struct {
	HasState       bool
	State          bool
	HasOscillating bool
	Oscillating    bool
	HasDirection   bool
	Direction      pb.FanDirection
	HasSpeedLevel  bool
	SpeedLevel     int32
	HasPresetMode  bool
	PresetMode     string
}

// SetFan sends a fan command with the given options.
func (c *Client) SetFan(key uint32, opts FanCommandOpts) error {
	if err := c.validateEntity(key, DomainFan); err != nil {
		return err
	}
	return c.SendCommand(&pb.FanCommandRequest{
		Key:            key,
		HasState:       opts.HasState,
		State:          opts.State,
		HasOscillating: opts.HasOscillating,
		Oscillating:    opts.Oscillating,
		HasDirection:   opts.HasDirection,
		Direction:      opts.Direction,
		HasSpeedLevel:  opts.HasSpeedLevel,
		SpeedLevel:     opts.SpeedLevel,
		HasPresetMode:  opts.HasPresetMode,
		PresetMode:     opts.PresetMode,
	})
}

// SirenCommandOpts holds optional parameters for a siren command.
type SirenCommandOpts struct {
	HasState    bool
	State       bool
	HasTone     bool
	Tone        string
	HasDuration bool
	Duration    uint32
	HasVolume   bool
	Volume      float32
}

// SetSiren sends a siren command with the given options.
func (c *Client) SetSiren(key uint32, opts SirenCommandOpts) error {
	if err := c.validateEntity(key, DomainSiren); err != nil {
		return err
	}
	return c.SendCommand(&pb.SirenCommandRequest{
		Key:         key,
		HasState:    opts.HasState,
		State:       opts.State,
		HasTone:     opts.HasTone,
		Tone:        opts.Tone,
		HasDuration: opts.HasDuration,
		Duration:    opts.Duration,
		HasVolume:   opts.HasVolume,
		Volume:      opts.Volume,
	})
}

// SetLock sends a lock command.
func (c *Client) SetLock(key uint32, command pb.LockCommand, code string) error {
	if err := c.validateEntity(key, DomainLock); err != nil {
		return err
	}
	cmd := &pb.LockCommandRequest{
		Key:     key,
		Command: command,
	}
	if code != "" {
		cmd.HasCode = true
		cmd.Code = code
	}
	return c.SendCommand(cmd)
}

// MediaPlayerCommandOpts holds optional parameters for a media player command.
type MediaPlayerCommandOpts struct {
	HasCommand      bool
	Command         pb.MediaPlayerCommand
	HasVolume       bool
	Volume          float32
	HasMediaUrl     bool
	MediaUrl        string
	HasAnnouncement bool
	Announcement    bool
}

// SetMediaPlayer sends a media player command with the given options.
func (c *Client) SetMediaPlayer(key uint32, opts MediaPlayerCommandOpts) error {
	if err := c.validateEntity(key, DomainMediaPlayer); err != nil {
		return err
	}
	return c.SendCommand(&pb.MediaPlayerCommandRequest{
		Key:             key,
		HasCommand:      opts.HasCommand,
		Command:         opts.Command,
		HasVolume:       opts.HasVolume,
		Volume:          opts.Volume,
		HasMediaUrl:     opts.HasMediaUrl,
		MediaUrl:        opts.MediaUrl,
		HasAnnouncement: opts.HasAnnouncement,
		Announcement:    opts.Announcement,
	})
}

// --- Entity key validation ---

// validateEntity checks that the given key exists in the entity registry and
// matches the expected domain. Returns nil if the entity registry is empty
// (entities may not have been listed yet) to allow raw sends.
func (c *Client) validateEntity(key uint32, expected EntityDomain) error {
	if c.entities.Len() == 0 {
		// Registry not populated — skip validation
		return nil
	}
	entity := c.entities.ByKey(key)
	if entity == nil {
		return fmt.Errorf("%w: key 0x%08X", ErrEntityNotFound, key)
	}
	if entity.GetDomain() != expected {
		return fmt.Errorf("%w: key 0x%08X is %s, expected %s", ErrEntityTypeMismatch, key, entity.GetDomain(), expected)
	}
	return nil
}
