package esphome_apiclient

import (
	"fmt"
	"sync"

	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
)

// EntityDomain represents the type of an ESPHome entity.
type EntityDomain int

const (
	DomainSensor EntityDomain = iota
	DomainBinarySensor
	DomainCover
	DomainFan
	DomainLight
	DomainSwitch
	DomainTextSensor
	DomainCamera
	DomainClimate
	DomainNumber
	DomainSelect
	DomainSiren
	DomainLock
	DomainButton
	DomainMediaPlayer
	DomainWaterHeater
)

func (d EntityDomain) String() string {
	switch d {
	case DomainSensor:
		return "sensor"
	case DomainBinarySensor:
		return "binary_sensor"
	case DomainCover:
		return "cover"
	case DomainFan:
		return "fan"
	case DomainLight:
		return "light"
	case DomainSwitch:
		return "switch"
	case DomainTextSensor:
		return "text_sensor"
	case DomainCamera:
		return "camera"
	case DomainClimate:
		return "climate"
	case DomainNumber:
		return "number"
	case DomainSelect:
		return "select"
	case DomainSiren:
		return "siren"
	case DomainLock:
		return "lock"
	case DomainButton:
		return "button"
	case DomainMediaPlayer:
		return "media_player"
	case DomainWaterHeater:
		return "water_heater"
	default:
		return fmt.Sprintf("unknown(%d)", int(d))
	}
}

// Entity is the common interface for all entity types.
type Entity interface {
	GetKey() uint32
	GetName() string
	GetObjectID() string
	GetDomain() EntityDomain
}

// ---- Sensor ----

// SensorEntity represents a sensor entity with metadata and cached state.
type SensorEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	UnitOfMeasurement string
	AccuracyDecimals  int32
	ForceUpdate       bool
	DeviceClass       string
	StateClass        pb.SensorStateClass
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State        float32
	MissingState bool
}

func (e *SensorEntity) GetKey() uint32        { return e.Key }
func (e *SensorEntity) GetName() string        { return e.Name }
func (e *SensorEntity) GetObjectID() string    { return e.ObjectID }
func (e *SensorEntity) GetDomain() EntityDomain { return DomainSensor }

// ---- BinarySensor ----

// BinarySensorEntity represents a binary sensor entity with metadata and cached state.
type BinarySensorEntity struct {
	Key                  uint32
	ObjectID             string
	Name                 string
	Icon                 string
	DeviceClass          string
	IsStatusBinarySensor bool
	DisabledByDefault    bool
	EntityCategory       pb.EntityCategory
	DeviceID             uint32

	// Cached state
	State        bool
	MissingState bool
}

func (e *BinarySensorEntity) GetKey() uint32        { return e.Key }
func (e *BinarySensorEntity) GetName() string        { return e.Name }
func (e *BinarySensorEntity) GetObjectID() string    { return e.ObjectID }
func (e *BinarySensorEntity) GetDomain() EntityDomain { return DomainBinarySensor }

// ---- Cover ----

// CoverEntity represents a cover entity with metadata and cached state.
type CoverEntity struct {
	Key              uint32
	ObjectID         string
	Name             string
	Icon             string
	AssumedState     bool
	SupportsPosition bool
	SupportsTilt     bool
	SupportsStop     bool
	DeviceClass      string
	DisabledByDefault bool
	EntityCategory   pb.EntityCategory
	DeviceID         uint32

	// Cached state
	Position         float32
	Tilt             float32
	CurrentOperation pb.CoverOperation
}

func (e *CoverEntity) GetKey() uint32        { return e.Key }
func (e *CoverEntity) GetName() string        { return e.Name }
func (e *CoverEntity) GetObjectID() string    { return e.ObjectID }
func (e *CoverEntity) GetDomain() EntityDomain { return DomainCover }

// ---- Fan ----

// FanEntity represents a fan entity with metadata and cached state.
type FanEntity struct {
	Key                 uint32
	ObjectID            string
	Name                string
	Icon                string
	SupportsOscillation bool
	SupportsSpeed       bool
	SupportsDirection   bool
	SupportedSpeedCount int32
	SupportedPresetModes []string
	DisabledByDefault   bool
	EntityCategory      pb.EntityCategory
	DeviceID            uint32

	// Cached state
	State      bool
	Oscillating bool
	Direction  pb.FanDirection
	SpeedLevel int32
	PresetMode string
}

func (e *FanEntity) GetKey() uint32        { return e.Key }
func (e *FanEntity) GetName() string        { return e.Name }
func (e *FanEntity) GetObjectID() string    { return e.ObjectID }
func (e *FanEntity) GetDomain() EntityDomain { return DomainFan }

// ---- Light ----

// LightEntity represents a light entity with metadata and cached state.
type LightEntity struct {
	Key                uint32
	ObjectID           string
	Name               string
	Icon               string
	SupportedColorModes []pb.ColorMode
	MinMireds          float32
	MaxMireds          float32
	Effects            []string
	DisabledByDefault  bool
	EntityCategory     pb.EntityCategory
	DeviceID           uint32

	// Cached state
	State            bool
	Brightness       float32
	ColorMode        pb.ColorMode
	ColorBrightness  float32
	Red              float32
	Green            float32
	Blue             float32
	White            float32
	ColorTemperature float32
	ColdWhite        float32
	WarmWhite        float32
	Effect           string
}

func (e *LightEntity) GetKey() uint32        { return e.Key }
func (e *LightEntity) GetName() string        { return e.Name }
func (e *LightEntity) GetObjectID() string    { return e.ObjectID }
func (e *LightEntity) GetDomain() EntityDomain { return DomainLight }

// ---- Switch ----

// SwitchEntity represents a switch entity with metadata and cached state.
type SwitchEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	AssumedState      bool
	DeviceClass       string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State bool
}

func (e *SwitchEntity) GetKey() uint32        { return e.Key }
func (e *SwitchEntity) GetName() string        { return e.Name }
func (e *SwitchEntity) GetObjectID() string    { return e.ObjectID }
func (e *SwitchEntity) GetDomain() EntityDomain { return DomainSwitch }

// ---- TextSensor ----

// TextSensorEntity represents a text sensor entity with metadata and cached state.
type TextSensorEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	DeviceClass       string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State        string
	MissingState bool
}

func (e *TextSensorEntity) GetKey() uint32        { return e.Key }
func (e *TextSensorEntity) GetName() string        { return e.Name }
func (e *TextSensorEntity) GetObjectID() string    { return e.ObjectID }
func (e *TextSensorEntity) GetDomain() EntityDomain { return DomainTextSensor }

// ---- Camera ----

// CameraEntity represents a camera entity with metadata and cached state.
type CameraEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state: latest image frame
	ImageData []byte
	ImageDone bool
}

func (e *CameraEntity) GetKey() uint32        { return e.Key }
func (e *CameraEntity) GetName() string        { return e.Name }
func (e *CameraEntity) GetObjectID() string    { return e.ObjectID }
func (e *CameraEntity) GetDomain() EntityDomain { return DomainCamera }

// ---- Climate ----

// ClimateEntity represents a climate entity with metadata and cached state.
type ClimateEntity struct {
	Key                              uint32
	ObjectID                         string
	Name                             string
	Icon                             string
	SupportsCurrentTemperature       bool
	SupportsTwoPointTargetTemperature bool
	SupportedModes                   []pb.ClimateMode
	VisualMinTemperature             float32
	VisualMaxTemperature             float32
	VisualTargetTemperatureStep      float32
	VisualCurrentTemperatureStep     float32
	SupportsAction                   bool
	SupportedFanModes                []pb.ClimateFanMode
	SupportedSwingModes              []pb.ClimateSwingMode
	SupportedCustomFanModes          []string
	SupportedPresets                 []pb.ClimatePreset
	SupportedCustomPresets           []string
	SupportsCurrentHumidity          bool
	SupportsTargetHumidity           bool
	VisualMinHumidity                float32
	VisualMaxHumidity                float32
	DisabledByDefault                bool
	EntityCategory                   pb.EntityCategory
	DeviceID                         uint32

	// Cached state
	Mode                 pb.ClimateMode
	CurrentTemperature   float32
	TargetTemperature    float32
	TargetTemperatureLow  float32
	TargetTemperatureHigh float32
	Action               pb.ClimateAction
	FanMode              pb.ClimateFanMode
	SwingMode            pb.ClimateSwingMode
	CustomFanMode        string
	Preset               pb.ClimatePreset
	CustomPreset         string
	CurrentHumidity      float32
	TargetHumidity       float32
}

func (e *ClimateEntity) GetKey() uint32        { return e.Key }
func (e *ClimateEntity) GetName() string        { return e.Name }
func (e *ClimateEntity) GetObjectID() string    { return e.ObjectID }
func (e *ClimateEntity) GetDomain() EntityDomain { return DomainClimate }

// ---- Number ----

// NumberEntity represents a number entity with metadata and cached state.
type NumberEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	MinValue          float32
	MaxValue          float32
	Step              float32
	UnitOfMeasurement string
	Mode              pb.NumberMode
	DeviceClass       string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State        float32
	MissingState bool
}

func (e *NumberEntity) GetKey() uint32        { return e.Key }
func (e *NumberEntity) GetName() string        { return e.Name }
func (e *NumberEntity) GetObjectID() string    { return e.ObjectID }
func (e *NumberEntity) GetDomain() EntityDomain { return DomainNumber }

// ---- Select ----

// SelectEntity represents a select entity with metadata and cached state.
type SelectEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	Options           []string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State        string
	MissingState bool
}

func (e *SelectEntity) GetKey() uint32        { return e.Key }
func (e *SelectEntity) GetName() string        { return e.Name }
func (e *SelectEntity) GetObjectID() string    { return e.ObjectID }
func (e *SelectEntity) GetDomain() EntityDomain { return DomainSelect }

// ---- Siren ----

// SirenEntity represents a siren entity with metadata and cached state.
type SirenEntity struct {
	Key              uint32
	ObjectID         string
	Name             string
	Icon             string
	Tones            []string
	SupportsDuration bool
	SupportsVolume   bool
	DisabledByDefault bool
	EntityCategory   pb.EntityCategory
	DeviceID         uint32

	// Cached state
	State bool
}

func (e *SirenEntity) GetKey() uint32        { return e.Key }
func (e *SirenEntity) GetName() string        { return e.Name }
func (e *SirenEntity) GetObjectID() string    { return e.ObjectID }
func (e *SirenEntity) GetDomain() EntityDomain { return DomainSiren }

// ---- Lock ----

// LockEntity represents a lock entity with metadata and cached state.
type LockEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	AssumedState      bool
	SupportsOpen      bool
	RequiresCode      bool
	CodeFormat        string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State pb.LockState
}

func (e *LockEntity) GetKey() uint32        { return e.Key }
func (e *LockEntity) GetName() string        { return e.Name }
func (e *LockEntity) GetObjectID() string    { return e.ObjectID }
func (e *LockEntity) GetDomain() EntityDomain { return DomainLock }

// ---- Button ----

// ButtonEntity represents a button entity with metadata (buttons are stateless).
type ButtonEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	DeviceClass       string
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32
}

func (e *ButtonEntity) GetKey() uint32        { return e.Key }
func (e *ButtonEntity) GetName() string        { return e.Name }
func (e *ButtonEntity) GetObjectID() string    { return e.ObjectID }
func (e *ButtonEntity) GetDomain() EntityDomain { return DomainButton }

// ---- MediaPlayer ----

// MediaPlayerEntity represents a media player entity with metadata and cached state.
type MediaPlayerEntity struct {
	Key               uint32
	ObjectID          string
	Name              string
	Icon              string
	SupportsPause     bool
	DisabledByDefault bool
	EntityCategory    pb.EntityCategory
	DeviceID          uint32

	// Cached state
	State  pb.MediaPlayerState
	Volume float32
	Muted  bool
}

func (e *MediaPlayerEntity) GetKey() uint32        { return e.Key }
func (e *MediaPlayerEntity) GetName() string        { return e.Name }
func (e *MediaPlayerEntity) GetObjectID() string    { return e.ObjectID }
func (e *MediaPlayerEntity) GetDomain() EntityDomain { return DomainMediaPlayer }

// ---- WaterHeater ----

// WaterHeaterEntity represents a water heater entity with metadata and cached state.
type WaterHeaterEntity struct {
	Key                   uint32
	ObjectID              string
	Name                  string
	Icon                  string
	MinTemperature        float32
	MaxTemperature        float32
	TargetTemperatureStep float32
	SupportedModes        []pb.WaterHeaterMode
	SupportedFeatures     uint32
	DisabledByDefault     bool
	EntityCategory        pb.EntityCategory
	DeviceID              uint32

	// Cached state
	CurrentTemperature    float32
	TargetTemperature     float32
	TargetTemperatureLow  float32
	TargetTemperatureHigh float32
	Mode                  pb.WaterHeaterMode
	State                 uint32
}

func (e *WaterHeaterEntity) GetKey() uint32        { return e.Key }
func (e *WaterHeaterEntity) GetName() string        { return e.Name }
func (e *WaterHeaterEntity) GetObjectID() string    { return e.ObjectID }
func (e *WaterHeaterEntity) GetDomain() EntityDomain { return DomainWaterHeater }

// ---- EntityRegistry ----

// EntityRegistry caches entity metadata from ListEntities*Response messages
// and the latest state from *StateResponse messages. Thread-safe for concurrent access.
type EntityRegistry struct {
	mu sync.RWMutex

	sensors       map[uint32]*SensorEntity
	binarySensors map[uint32]*BinarySensorEntity
	covers        map[uint32]*CoverEntity
	fans          map[uint32]*FanEntity
	lights        map[uint32]*LightEntity
	switches      map[uint32]*SwitchEntity
	textSensors   map[uint32]*TextSensorEntity
	cameras       map[uint32]*CameraEntity
	climates      map[uint32]*ClimateEntity
	numbers       map[uint32]*NumberEntity
	selects       map[uint32]*SelectEntity
	sirens        map[uint32]*SirenEntity
	locks         map[uint32]*LockEntity
	buttons       map[uint32]*ButtonEntity
	mediaPlayers  map[uint32]*MediaPlayerEntity
	waterHeaters  map[uint32]*WaterHeaterEntity

	// byKey provides a unified lookup across all domains.
	byKey  map[uint32]Entity
	byName map[string]Entity
}

// NewEntityRegistry creates an empty EntityRegistry.
func NewEntityRegistry() *EntityRegistry {
	return &EntityRegistry{
		sensors:       make(map[uint32]*SensorEntity),
		binarySensors: make(map[uint32]*BinarySensorEntity),
		covers:        make(map[uint32]*CoverEntity),
		fans:          make(map[uint32]*FanEntity),
		lights:        make(map[uint32]*LightEntity),
		switches:      make(map[uint32]*SwitchEntity),
		textSensors:   make(map[uint32]*TextSensorEntity),
		cameras:       make(map[uint32]*CameraEntity),
		climates:      make(map[uint32]*ClimateEntity),
		numbers:       make(map[uint32]*NumberEntity),
		selects:       make(map[uint32]*SelectEntity),
		sirens:        make(map[uint32]*SirenEntity),
		locks:         make(map[uint32]*LockEntity),
		buttons:       make(map[uint32]*ButtonEntity),
		mediaPlayers:  make(map[uint32]*MediaPlayerEntity),
		waterHeaters:  make(map[uint32]*WaterHeaterEntity),
		byKey:         make(map[uint32]Entity),
		byName:        make(map[string]Entity),
	}
}

// --- Accessors (read-locked) ---

// Sensors returns a copy of all sensor entities.
func (r *EntityRegistry) Sensors() []*SensorEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*SensorEntity, 0, len(r.sensors))
	for _, e := range r.sensors {
		result = append(result, e)
	}
	return result
}

// BinarySensors returns a copy of all binary sensor entities.
func (r *EntityRegistry) BinarySensors() []*BinarySensorEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*BinarySensorEntity, 0, len(r.binarySensors))
	for _, e := range r.binarySensors {
		result = append(result, e)
	}
	return result
}

// Covers returns a copy of all cover entities.
func (r *EntityRegistry) Covers() []*CoverEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*CoverEntity, 0, len(r.covers))
	for _, e := range r.covers {
		result = append(result, e)
	}
	return result
}

// Fans returns a copy of all fan entities.
func (r *EntityRegistry) Fans() []*FanEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*FanEntity, 0, len(r.fans))
	for _, e := range r.fans {
		result = append(result, e)
	}
	return result
}

// Lights returns a copy of all light entities.
func (r *EntityRegistry) Lights() []*LightEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*LightEntity, 0, len(r.lights))
	for _, e := range r.lights {
		result = append(result, e)
	}
	return result
}

// Switches returns a copy of all switch entities.
func (r *EntityRegistry) Switches() []*SwitchEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*SwitchEntity, 0, len(r.switches))
	for _, e := range r.switches {
		result = append(result, e)
	}
	return result
}

// TextSensors returns a copy of all text sensor entities.
func (r *EntityRegistry) TextSensors() []*TextSensorEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*TextSensorEntity, 0, len(r.textSensors))
	for _, e := range r.textSensors {
		result = append(result, e)
	}
	return result
}

// Cameras returns a copy of all camera entities.
func (r *EntityRegistry) Cameras() []*CameraEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*CameraEntity, 0, len(r.cameras))
	for _, e := range r.cameras {
		result = append(result, e)
	}
	return result
}

// Climates returns a copy of all climate entities.
func (r *EntityRegistry) Climates() []*ClimateEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ClimateEntity, 0, len(r.climates))
	for _, e := range r.climates {
		result = append(result, e)
	}
	return result
}

// Numbers returns a copy of all number entities.
func (r *EntityRegistry) Numbers() []*NumberEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*NumberEntity, 0, len(r.numbers))
	for _, e := range r.numbers {
		result = append(result, e)
	}
	return result
}

// Selects returns a copy of all select entities.
func (r *EntityRegistry) Selects() []*SelectEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*SelectEntity, 0, len(r.selects))
	for _, e := range r.selects {
		result = append(result, e)
	}
	return result
}

// Sirens returns a copy of all siren entities.
func (r *EntityRegistry) Sirens() []*SirenEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*SirenEntity, 0, len(r.sirens))
	for _, e := range r.sirens {
		result = append(result, e)
	}
	return result
}

// Locks returns a copy of all lock entities.
func (r *EntityRegistry) Locks() []*LockEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*LockEntity, 0, len(r.locks))
	for _, e := range r.locks {
		result = append(result, e)
	}
	return result
}

// Buttons returns a copy of all button entities.
func (r *EntityRegistry) Buttons() []*ButtonEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ButtonEntity, 0, len(r.buttons))
	for _, e := range r.buttons {
		result = append(result, e)
	}
	return result
}

// MediaPlayers returns a copy of all media player entities.
func (r *EntityRegistry) MediaPlayers() []*MediaPlayerEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*MediaPlayerEntity, 0, len(r.mediaPlayers))
	for _, e := range r.mediaPlayers {
		result = append(result, e)
	}
	return result
}

// WaterHeaters returns a copy of all water heater entities.
func (r *EntityRegistry) WaterHeaters() []*WaterHeaterEntity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*WaterHeaterEntity, 0, len(r.waterHeaters))
	for _, e := range r.waterHeaters {
		result = append(result, e)
	}
	return result
}

// ByKey returns an entity by its key, or nil if not found.
func (r *EntityRegistry) ByKey(key uint32) Entity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byKey[key]
}

// ByName returns an entity by its name, or nil if not found.
func (r *EntityRegistry) ByName(name string) Entity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

// Len returns the total number of entities across all domains.
func (r *EntityRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byKey)
}

// --- Population from ListEntities*Response messages ---

// HandleListEntityMessage processes a ListEntities*Response protobuf message and
// populates the registry. Returns true if the message was handled.
func (r *EntityRegistry) HandleListEntityMessage(msg proto.Message) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch m := msg.(type) {
	case *pb.ListEntitiesSensorResponse:
		e := &SensorEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			UnitOfMeasurement: m.UnitOfMeasurement,
			AccuracyDecimals:  m.AccuracyDecimals,
			ForceUpdate:       m.ForceUpdate,
			DeviceClass:       m.DeviceClass,
			StateClass:        m.StateClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
			MissingState:      true, // no state received yet
		}
		r.sensors[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesBinarySensorResponse:
		e := &BinarySensorEntity{
			Key:                  m.Key,
			ObjectID:             m.ObjectId,
			Name:                 m.Name,
			Icon:                 m.Icon,
			DeviceClass:          m.DeviceClass,
			IsStatusBinarySensor: m.IsStatusBinarySensor,
			DisabledByDefault:    m.DisabledByDefault,
			EntityCategory:       m.EntityCategory,
			DeviceID:             m.DeviceId,
			MissingState:         true,
		}
		r.binarySensors[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesCoverResponse:
		e := &CoverEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			AssumedState:      m.AssumedState,
			SupportsPosition:  m.SupportsPosition,
			SupportsTilt:      m.SupportsTilt,
			SupportsStop:      m.SupportsStop,
			DeviceClass:       m.DeviceClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.covers[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesFanResponse:
		e := &FanEntity{
			Key:                  m.Key,
			ObjectID:             m.ObjectId,
			Name:                 m.Name,
			Icon:                 m.Icon,
			SupportsOscillation:  m.SupportsOscillation,
			SupportsSpeed:        m.SupportsSpeed,
			SupportsDirection:    m.SupportsDirection,
			SupportedSpeedCount:  m.SupportedSpeedCount,
			SupportedPresetModes: m.SupportedPresetModes,
			DisabledByDefault:    m.DisabledByDefault,
			EntityCategory:       m.EntityCategory,
			DeviceID:             m.DeviceId,
		}
		r.fans[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesLightResponse:
		e := &LightEntity{
			Key:                 m.Key,
			ObjectID:            m.ObjectId,
			Name:                m.Name,
			Icon:                m.Icon,
			SupportedColorModes: m.SupportedColorModes,
			MinMireds:           m.MinMireds,
			MaxMireds:           m.MaxMireds,
			Effects:             m.Effects,
			DisabledByDefault:   m.DisabledByDefault,
			EntityCategory:      m.EntityCategory,
			DeviceID:            m.DeviceId,
		}
		r.lights[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesSwitchResponse:
		e := &SwitchEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			AssumedState:      m.AssumedState,
			DeviceClass:       m.DeviceClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.switches[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesTextSensorResponse:
		e := &TextSensorEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			DeviceClass:       m.DeviceClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
			MissingState:      true,
		}
		r.textSensors[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesCameraResponse:
		e := &CameraEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.cameras[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesClimateResponse:
		e := &ClimateEntity{
			Key:                              m.Key,
			ObjectID:                         m.ObjectId,
			Name:                             m.Name,
			Icon:                             m.Icon,
			SupportsCurrentTemperature:       m.SupportsCurrentTemperature,
			SupportsTwoPointTargetTemperature: m.SupportsTwoPointTargetTemperature,
			SupportedModes:                   m.SupportedModes,
			VisualMinTemperature:             m.VisualMinTemperature,
			VisualMaxTemperature:             m.VisualMaxTemperature,
			VisualTargetTemperatureStep:      m.VisualTargetTemperatureStep,
			VisualCurrentTemperatureStep:     m.VisualCurrentTemperatureStep,
			SupportsAction:                   m.SupportsAction,
			SupportedFanModes:                m.SupportedFanModes,
			SupportedSwingModes:              m.SupportedSwingModes,
			SupportedCustomFanModes:          m.SupportedCustomFanModes,
			SupportedPresets:                 m.SupportedPresets,
			SupportedCustomPresets:           m.SupportedCustomPresets,
			SupportsCurrentHumidity:          m.SupportsCurrentHumidity,
			SupportsTargetHumidity:           m.SupportsTargetHumidity,
			VisualMinHumidity:                m.VisualMinHumidity,
			VisualMaxHumidity:                m.VisualMaxHumidity,
			DisabledByDefault:                m.DisabledByDefault,
			EntityCategory:                   m.EntityCategory,
			DeviceID:                         m.DeviceId,
		}
		r.climates[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesNumberResponse:
		e := &NumberEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			MinValue:          m.MinValue,
			MaxValue:          m.MaxValue,
			Step:              m.Step,
			UnitOfMeasurement: m.UnitOfMeasurement,
			Mode:              m.Mode,
			DeviceClass:       m.DeviceClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
			MissingState:      true,
		}
		r.numbers[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesSelectResponse:
		e := &SelectEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			Options:           m.Options,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
			MissingState:      true,
		}
		r.selects[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesSirenResponse:
		e := &SirenEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			Tones:             m.Tones,
			SupportsDuration:  m.SupportsDuration,
			SupportsVolume:    m.SupportsVolume,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.sirens[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesLockResponse:
		e := &LockEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			AssumedState:      m.AssumedState,
			SupportsOpen:      m.SupportsOpen,
			RequiresCode:      m.RequiresCode,
			CodeFormat:        m.CodeFormat,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.locks[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesButtonResponse:
		e := &ButtonEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			DeviceClass:       m.DeviceClass,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.buttons[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesMediaPlayerResponse:
		e := &MediaPlayerEntity{
			Key:               m.Key,
			ObjectID:          m.ObjectId,
			Name:              m.Name,
			Icon:              m.Icon,
			SupportsPause:     m.SupportsPause,
			DisabledByDefault: m.DisabledByDefault,
			EntityCategory:    m.EntityCategory,
			DeviceID:          m.DeviceId,
		}
		r.mediaPlayers[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	case *pb.ListEntitiesWaterHeaterResponse:
		e := &WaterHeaterEntity{
			Key:                   m.Key,
			ObjectID:              m.ObjectId,
			Name:                  m.Name,
			Icon:                  m.Icon,
			MinTemperature:        m.MinTemperature,
			MaxTemperature:        m.MaxTemperature,
			TargetTemperatureStep: m.TargetTemperatureStep,
			SupportedModes:        m.SupportedModes,
			SupportedFeatures:     m.SupportedFeatures,
			DisabledByDefault:     m.DisabledByDefault,
			EntityCategory:        m.EntityCategory,
			DeviceID:              m.DeviceId,
		}
		r.waterHeaters[m.Key] = e
		r.byKey[m.Key] = e
		r.byName[m.Name] = e

	default:
		return false
	}

	return true
}

// --- State Updates from *StateResponse messages ---

// HandleStateMessage processes a *StateResponse protobuf message and updates
// the cached state of the corresponding entity. Returns true if the message was handled.
func (r *EntityRegistry) HandleStateMessage(msg proto.Message) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch m := msg.(type) {
	case *pb.SensorStateResponse:
		if e, ok := r.sensors[m.Key]; ok {
			e.State = m.State
			e.MissingState = m.MissingState
			return true
		}

	case *pb.BinarySensorStateResponse:
		if e, ok := r.binarySensors[m.Key]; ok {
			e.State = m.State
			e.MissingState = m.MissingState
			return true
		}

	case *pb.CoverStateResponse:
		if e, ok := r.covers[m.Key]; ok {
			e.Position = m.Position
			e.Tilt = m.Tilt
			e.CurrentOperation = m.CurrentOperation
			return true
		}

	case *pb.FanStateResponse:
		if e, ok := r.fans[m.Key]; ok {
			e.State = m.State
			e.Oscillating = m.Oscillating
			e.Direction = m.Direction
			e.SpeedLevel = m.SpeedLevel
			e.PresetMode = m.PresetMode
			return true
		}

	case *pb.LightStateResponse:
		if e, ok := r.lights[m.Key]; ok {
			e.State = m.State
			e.Brightness = m.Brightness
			e.ColorMode = m.ColorMode
			e.ColorBrightness = m.ColorBrightness
			e.Red = m.Red
			e.Green = m.Green
			e.Blue = m.Blue
			e.White = m.White
			e.ColorTemperature = m.ColorTemperature
			e.ColdWhite = m.ColdWhite
			e.WarmWhite = m.WarmWhite
			e.Effect = m.Effect
			return true
		}

	case *pb.SwitchStateResponse:
		if e, ok := r.switches[m.Key]; ok {
			e.State = m.State
			return true
		}

	case *pb.TextSensorStateResponse:
		if e, ok := r.textSensors[m.Key]; ok {
			e.State = m.State
			e.MissingState = m.MissingState
			return true
		}

	case *pb.CameraImageResponse:
		if e, ok := r.cameras[m.Key]; ok {
			if e.ImageDone || e.ImageData == nil {
				// Start of a new image frame
				e.ImageData = m.Data
			} else {
				// Continuation of the same frame
				e.ImageData = append(e.ImageData, m.Data...)
			}
			e.ImageDone = m.Done
			return true
		}

	case *pb.ClimateStateResponse:
		if e, ok := r.climates[m.Key]; ok {
			e.Mode = m.Mode
			e.CurrentTemperature = m.CurrentTemperature
			e.TargetTemperature = m.TargetTemperature
			e.TargetTemperatureLow = m.TargetTemperatureLow
			e.TargetTemperatureHigh = m.TargetTemperatureHigh
			e.Action = m.Action
			e.FanMode = m.FanMode
			e.SwingMode = m.SwingMode
			e.CustomFanMode = m.CustomFanMode
			e.Preset = m.Preset
			e.CustomPreset = m.CustomPreset
			e.CurrentHumidity = m.CurrentHumidity
			e.TargetHumidity = m.TargetHumidity
			return true
		}

	case *pb.NumberStateResponse:
		if e, ok := r.numbers[m.Key]; ok {
			e.State = m.State
			e.MissingState = m.MissingState
			return true
		}

	case *pb.SelectStateResponse:
		if e, ok := r.selects[m.Key]; ok {
			e.State = m.State
			e.MissingState = m.MissingState
			return true
		}

	case *pb.SirenStateResponse:
		if e, ok := r.sirens[m.Key]; ok {
			e.State = m.State
			return true
		}

	case *pb.LockStateResponse:
		if e, ok := r.locks[m.Key]; ok {
			e.State = m.State
			return true
		}

	case *pb.MediaPlayerStateResponse:
		if e, ok := r.mediaPlayers[m.Key]; ok {
			e.State = m.State
			e.Volume = m.Volume
			e.Muted = m.Muted
			return true
		}

	case *pb.WaterHeaterStateResponse:
		if e, ok := r.waterHeaters[m.Key]; ok {
			e.CurrentTemperature = m.CurrentTemperature
			e.TargetTemperature = m.TargetTemperature
			e.TargetTemperatureLow = m.TargetTemperatureLow
			e.TargetTemperatureHigh = m.TargetTemperatureHigh
			e.Mode = m.Mode
			e.State = m.State
			return true
		}
	}

	return false
}

// Clear removes all entities from the registry.
func (r *EntityRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sensors = make(map[uint32]*SensorEntity)
	r.binarySensors = make(map[uint32]*BinarySensorEntity)
	r.covers = make(map[uint32]*CoverEntity)
	r.fans = make(map[uint32]*FanEntity)
	r.lights = make(map[uint32]*LightEntity)
	r.switches = make(map[uint32]*SwitchEntity)
	r.textSensors = make(map[uint32]*TextSensorEntity)
	r.cameras = make(map[uint32]*CameraEntity)
	r.climates = make(map[uint32]*ClimateEntity)
	r.numbers = make(map[uint32]*NumberEntity)
	r.selects = make(map[uint32]*SelectEntity)
	r.sirens = make(map[uint32]*SirenEntity)
	r.locks = make(map[uint32]*LockEntity)
	r.buttons = make(map[uint32]*ButtonEntity)
	r.mediaPlayers = make(map[uint32]*MediaPlayerEntity)
	r.waterHeaters = make(map[uint32]*WaterHeaterEntity)
	r.byKey = make(map[uint32]Entity)
	r.byName = make(map[string]Entity)
}
