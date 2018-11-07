package config

import (
	"flag"
	_ "glog"
	)

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
	flag.StringVar(&CKconfig.EdgeGroupID, "node_id", "", "node id.")
	flag.StringVar(&CKconfig.MqttURL, "mqtt-url", "127.0.0.1:1883", "mqtt url, default 127.0.0.1:1883.")
	flag.StringVar(&CKconfig.MqttUsername, "mqtt-user", "qwq", "mqtt user name.")
	flag.StringVar(&CKconfig.MqttPassword, "mqtt-pwd", "qwq", "mqtt password.")
	flag.IntVar(&CKconfig.CheckCameraSec, "check-camera-interval", 60, "camera checker server interval.")
	flag.IntVar(&CKconfig.MqttRetries, "mqtt-retry-time", 60, "mqtt client retry times.")
	flag.BoolVar(&CKconfig.Remote, "check-remote-camera", false, "check real camera.")
	flag.Parse()
}

const (
	TopicGetDevices = "node/<edgeGroupID>/membership/get"
	TopicGetDevicesResult = "node/<edgeGroupID>/membership/get/result"
	TopicUpdatedDevices = "node/<edgeGroupID>/update"

	TopicUpdatedDevice = "device/<deviceID>/update"
	TopicDeletedDevice = "device/<deviceID>/delete"

	TopicUpdateTwinDevice = ""

	DeviceTwinEventType = "device_twin"
	UpdatedOperationType = "update"
	GroupEventType = "node"
)
