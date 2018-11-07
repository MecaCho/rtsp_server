package config

import "flag"

type config struct {
	EdgeGroupID string
	MqttURL string
	MqttUsername string
	MqttPassword string
	CheckCameraSec int
	MqttRetries int
	Remote bool
}

var CKconfig config
func init() {

	flag.StringVar()
	flag.Parse()
}

const (
	TopicUpdatedDevice = ""
	TopicDeletedDevice = ""
	TopicUpdatedDevices = ""
	TopicUpdateTwinDevice = ""

	DeviceTwinEventType = ""
	UpdatedOperationType = ""
	GroupEventType = ""
)
