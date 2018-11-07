package model

type GroupMembershipEvent struct {
	MemberShip
}

type Device struct {
	ID string `json:"id"`
	Name string `json:"name"`
	State bool `json:"state"`
	CameraStatus string `json:"camera_status"`
	Attributes []Attribute `json:"attributes"`
}
type Attribute struct {
	Key string `json:"key"`
	Value string `json:"value"`
	IsEncrypt bool `json:"is_encrypt"`
}

type MemberShip struct {
	Devices []Device `json:"devices"`
	AddedDevices []Device `json:"added_devices"`
	RemovedDevices []Device `json:"removed_devices"`
}

type BaseEvent struct {
	EventType string `json:"event_type"`
}

type GroupEventType struct {
	BaseEvent

}

type DeviceEvent struct {
	BaseEvent
	DeviceName string `json:"device_name"`
	Attributes []Attribute `json:"attributes"`
}

type DeviceTwinEvent struct {
	EventType string `json:"event_type"`
	DeviceName string `json:"device_name"`
	DeviceID string `json:"device_id"`
	Operation string `json:"operation"`
	Timestamp int64 `json:"timestamp"`
	Twin Twin `json:"twin"`
}

type Twin struct {
	Actual map[string]string `json:"actual"`

}

type EdgeGet struct {
	EventID string `json:"event_id"`
}