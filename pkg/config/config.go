package config

import (
	"flag"
	_ "glog"
	"fmt"
)

type config struct {
	NodeID string
	MqttURL string
	MqttUsername string
	MqttPassword string
	CheckCameraSec int
	MqttRetries int
	Remote bool
}

var CKconfig config
var (
	TopicGetDevices = fmt.Sprintf("node/%s/membership/get", CKconfig.NodeID)
	TopicGetDevicesResult = fmt.Sprintf("node/%s/membership/get/result", CKconfig.NodeID)
	TopicUpdatedDevices = fmt.Sprintf("node/%s/update", CKconfig.NodeID)
)

func init() {
	flag.StringVar(&CKconfig.NodeID, "node_id", "", "node id.")
	flag.StringVar(&CKconfig.MqttURL, "mqtt-url", "127.0.0.1:1883", "mqtt url, default 127.0.0.1:1883.")
	flag.IntVar(&CKconfig.CheckCameraSec, "check-camera-interval", 20, "camera checker server interval.")
	flag.IntVar(&CKconfig.MqttRetries, "mqtt-retry-time", 60, "mqtt client retry times.")
	flag.BoolVar(&CKconfig.Remote, "check-remote-camera", false, "check real camera.")
	flag.Parse()
	TopicGetDevices = fmt.Sprintf("node/%s/membership/get", CKconfig.NodeID)
	TopicGetDevicesResult = fmt.Sprintf("node/%s/membership/get/result", CKconfig.NodeID)
	TopicUpdatedDevices = fmt.Sprintf("node/%s/update", CKconfig.NodeID)
}

var (

	TopicUpdatedDevice = "device/%s/update"
	TopicDeletedDevice = "device/%s/delete"

	TopicUpdateTwinDevice = "device/%s/twin/update"

	DeviceTwinEventType = "device_twin"
	UpdatedOperationType = "update"
	GroupEventType = "node"
)

const (
	CameraStatusOn = "Online"
	CameraStatusOff = "Offline"
)
