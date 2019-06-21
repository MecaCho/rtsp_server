package model

type Device struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	State        string               `json:"state"`
	CameraStatus string               `json:"camera_status"`
	Attributes   map[string]Attribute `json:"attributes"`
}
type Attribute struct {
	Value     string `json:"value"`
	Optional  bool   `json:"optional"`
	IsEncrypt bool   `json:"is_encrypt"`
}

type GroupMembershipEvent struct {
	BaseEvent
	MemberShip
}

type MemberShip struct {
	Devices        []Device `json:"devices, omitempty"`
	AddedDevices   []Device `json:"added_devices, omitempty"`
	RemovedDevices []Device `json:"removed_devices, omitempty"`
}

type BaseEvent struct {
	EventType string `json:"event_type"`
}

type GroupEventType struct {
	BaseEvent
}

type DeviceEvent struct {
	BaseEvent
	DeviceName string               `json:"device_name"`
	Attributes map[string]Attribute `json:"attributes"`
}

type DeviceTwinEvent struct {
	EventType  string          `json:"event_type"`
	DeviceName string          `json:"device_name"`
	DeviceID   string          `json:"device_id"`
	Operation  string          `json:"operation"`
	Timestamp  int64           `json:"timestamp"`
	Twin       map[string]Twin `json:"twin"`
}

type Twin struct {
	Actual map[string]string `json:"actual"`
}

type EdgeGet struct {
	EventID string `json:"event_id"`
}
