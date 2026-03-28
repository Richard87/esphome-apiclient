package pb

import (
	"google.golang.org/protobuf/proto"
)

// ListEntityResponseIDs contains message type IDs for all ListEntities*Response messages
// (excluding ListEntitiesDoneResponse at ID 19).
var ListEntityResponseIDs = []uint32{
	12,  // ListEntitiesBinarySensorResponse
	13,  // ListEntitiesCoverResponse
	14,  // ListEntitiesFanResponse
	15,  // ListEntitiesLightResponse
	16,  // ListEntitiesSensorResponse
	17,  // ListEntitiesSwitchResponse
	18,  // ListEntitiesTextSensorResponse
	41,  // ListEntitiesServicesResponse
	43,  // ListEntitiesCameraResponse
	46,  // ListEntitiesClimateResponse
	49,  // ListEntitiesNumberResponse
	52,  // ListEntitiesSelectResponse
	55,  // ListEntitiesSirenResponse
	58,  // ListEntitiesLockResponse
	61,  // ListEntitiesButtonResponse
	63,  // ListEntitiesMediaPlayerResponse
	132, // ListEntitiesWaterHeaterResponse
}

// StateResponseIDs contains message type IDs for all *StateResponse messages.
var StateResponseIDs = []uint32{
	21,  // BinarySensorStateResponse
	22,  // CoverStateResponse
	23,  // FanStateResponse
	24,  // LightStateResponse
	25,  // SensorStateResponse
	26,  // SwitchStateResponse
	27,  // TextSensorStateResponse
	44,  // CameraImageResponse
	47,  // ClimateStateResponse
	50,  // NumberStateResponse
	53,  // SelectStateResponse
	56,  // SirenStateResponse
	59,  // LockStateResponse
	64,  // MediaPlayerStateResponse
	133, // WaterHeaterStateResponse
}

// ListEntitiesDoneResponseID is the message type ID for ListEntitiesDoneResponse.
const ListEntitiesDoneResponseID uint32 = 19

var MessageRegistry = map[uint32]func() proto.Message{
	1:   func() proto.Message { return &HelloRequest{} },
	2:   func() proto.Message { return &HelloResponse{} },
	3:   func() proto.Message { return &AuthenticationRequest{} },
	4:   func() proto.Message { return &AuthenticationResponse{} },
	5:   func() proto.Message { return &DisconnectRequest{} },
	6:   func() proto.Message { return &DisconnectResponse{} },
	7:   func() proto.Message { return &PingRequest{} },
	8:   func() proto.Message { return &PingResponse{} },
	9:   func() proto.Message { return &DeviceInfoRequest{} },
	10:  func() proto.Message { return &DeviceInfoResponse{} },
	11:  func() proto.Message { return &ListEntitiesRequest{} },
	12:  func() proto.Message { return &ListEntitiesBinarySensorResponse{} },
	13:  func() proto.Message { return &ListEntitiesCoverResponse{} },
	14:  func() proto.Message { return &ListEntitiesFanResponse{} },
	15:  func() proto.Message { return &ListEntitiesLightResponse{} },
	16:  func() proto.Message { return &ListEntitiesSensorResponse{} },
	17:  func() proto.Message { return &ListEntitiesSwitchResponse{} },
	18:  func() proto.Message { return &ListEntitiesTextSensorResponse{} },
	19:  func() proto.Message { return &ListEntitiesDoneResponse{} },
	20:  func() proto.Message { return &SubscribeStatesRequest{} },
	21:  func() proto.Message { return &BinarySensorStateResponse{} },
	22:  func() proto.Message { return &CoverStateResponse{} },
	23:  func() proto.Message { return &FanStateResponse{} },
	24:  func() proto.Message { return &LightStateResponse{} },
	25:  func() proto.Message { return &SensorStateResponse{} },
	26:  func() proto.Message { return &SwitchStateResponse{} },
	27:  func() proto.Message { return &TextSensorStateResponse{} },
	28:  func() proto.Message { return &SubscribeLogsRequest{} },
	29:  func() proto.Message { return &SubscribeLogsResponse{} },
	30:  func() proto.Message { return &CoverCommandRequest{} },
	31:  func() proto.Message { return &FanCommandRequest{} },
	32:  func() proto.Message { return &LightCommandRequest{} },
	33:  func() proto.Message { return &SwitchCommandRequest{} },
	34:  func() proto.Message { return &SubscribeHomeassistantServicesRequest{} },
	35:  func() proto.Message { return &HomeassistantActionRequest{} },
	36:  func() proto.Message { return &GetTimeRequest{} },
	37:  func() proto.Message { return &GetTimeResponse{} },
	38:  func() proto.Message { return &SubscribeHomeAssistantStatesRequest{} },
	39:  func() proto.Message { return &SubscribeHomeAssistantStateResponse{} },
	40:  func() proto.Message { return &HomeAssistantStateResponse{} },
	41:  func() proto.Message { return &ListEntitiesServicesResponse{} },
	42:  func() proto.Message { return &ExecuteServiceRequest{} },
	43:  func() proto.Message { return &ListEntitiesCameraResponse{} },
	44:  func() proto.Message { return &CameraImageResponse{} },
	45:  func() proto.Message { return &CameraImageRequest{} },
	46:  func() proto.Message { return &ListEntitiesClimateResponse{} },
	47:  func() proto.Message { return &ClimateStateResponse{} },
	48:  func() proto.Message { return &ClimateCommandRequest{} },
	49:  func() proto.Message { return &ListEntitiesNumberResponse{} },
	50:  func() proto.Message { return &NumberStateResponse{} },
	51:  func() proto.Message { return &NumberCommandRequest{} },
	52:  func() proto.Message { return &ListEntitiesSelectResponse{} },
	53:  func() proto.Message { return &SelectStateResponse{} },
	54:  func() proto.Message { return &SelectCommandRequest{} },
	55:  func() proto.Message { return &ListEntitiesSirenResponse{} },
	56:  func() proto.Message { return &SirenStateResponse{} },
	57:  func() proto.Message { return &SirenCommandRequest{} },
	58:  func() proto.Message { return &ListEntitiesLockResponse{} },
	59:  func() proto.Message { return &LockStateResponse{} },
	60:  func() proto.Message { return &LockCommandRequest{} },
	61:  func() proto.Message { return &ListEntitiesButtonResponse{} },
	62:  func() proto.Message { return &ButtonCommandRequest{} },
	63:  func() proto.Message { return &ListEntitiesMediaPlayerResponse{} },
	64:  func() proto.Message { return &MediaPlayerStateResponse{} },
	65:  func() proto.Message { return &MediaPlayerCommandRequest{} },
	66:  func() proto.Message { return &SubscribeBluetoothLEAdvertisementsRequest{} },
	67:  func() proto.Message { return &BluetoothLEAdvertisementResponse{} },
	68:  func() proto.Message { return &BluetoothDeviceRequest{} },
	69:  func() proto.Message { return &BluetoothDeviceConnectionResponse{} },
	70:  func() proto.Message { return &BluetoothGATTGetServicesRequest{} },
	71:  func() proto.Message { return &BluetoothGATTGetServicesResponse{} },
	72:  func() proto.Message { return &BluetoothGATTGetServicesDoneResponse{} },
	73:  func() proto.Message { return &BluetoothGATTReadRequest{} },
	74:  func() proto.Message { return &BluetoothGATTReadResponse{} },
	75:  func() proto.Message { return &BluetoothGATTWriteRequest{} },
	76:  func() proto.Message { return &BluetoothGATTReadDescriptorRequest{} },
	77:  func() proto.Message { return &BluetoothGATTWriteDescriptorRequest{} },
	78:  func() proto.Message { return &BluetoothGATTNotifyRequest{} },
	79:  func() proto.Message { return &BluetoothGATTNotifyDataResponse{} },
	80:  func() proto.Message { return &SubscribeBluetoothConnectionsFreeRequest{} },
	81:  func() proto.Message { return &BluetoothConnectionsFreeResponse{} },
	82:  func() proto.Message { return &BluetoothGATTErrorResponse{} },
	83:  func() proto.Message { return &BluetoothGATTWriteResponse{} },
	84:  func() proto.Message { return &BluetoothGATTNotifyResponse{} },
	85:  func() proto.Message { return &BluetoothDevicePairingResponse{} },
	86:  func() proto.Message { return &BluetoothDeviceUnpairingResponse{} },
	87:  func() proto.Message { return &UnsubscribeBluetoothLEAdvertisementsRequest{} },
	88:  func() proto.Message { return &BluetoothDeviceClearCacheResponse{} },
	89:  func() proto.Message { return &SubscribeVoiceAssistantRequest{} },
	90:  func() proto.Message { return &VoiceAssistantRequest{} },
	91:  func() proto.Message { return &VoiceAssistantResponse{} },
	92:  func() proto.Message { return &VoiceAssistantEventResponse{} },
	93:  func() proto.Message { return &BluetoothLERawAdvertisementsResponse{} },
	106: func() proto.Message { return &VoiceAssistantAudio{} },
	115: func() proto.Message { return &VoiceAssistantTimerEventResponse{} },
	119: func() proto.Message { return &VoiceAssistantAnnounceRequest{} },
	120: func() proto.Message { return &VoiceAssistantAnnounceFinished{} },
	124: func() proto.Message { return &NoiseEncryptionSetKeyRequest{} },
	125: func() proto.Message { return &NoiseEncryptionSetKeyResponse{} },
	126: func() proto.Message { return &BluetoothScannerStateResponse{} },
	127: func() proto.Message { return &BluetoothScannerSetModeRequest{} },
	130: func() proto.Message { return &HomeassistantActionResponse{} },
	131: func() proto.Message { return &ExecuteServiceResponse{} },
	132: func() proto.Message { return &ListEntitiesWaterHeaterResponse{} },
	133: func() proto.Message { return &WaterHeaterStateResponse{} },
	134: func() proto.Message { return &WaterHeaterCommandRequest{} },
}
