package checker

import (
	"rtsp_server/pkg/config"
	"rtsp_server/pkg/mqtt_client"
	"rtsp_server/pkg/model"
	"rtsp_server/pkg/rtspclient"
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/robfig/cron"
	"math/rand"
	"strconv"
	"sync"
	"time"
	"glog"
	"rtsp_server/pkg/common"
)

var syncMap sync.Map

// Manager ...
type Manager struct {
	MQTTClient mqtt_client.Client
	FinishedInitLoad bool
}

func getDevice(deviceID string) model.Device {
	var nilDev model.Device
	dev, ok := syncMap.Load(deviceID)
	if ok{
		nilDev = dev.(model.Device)
	}
	return nilDev
}

func deleteDevice(deviceID string) {
	syncMap.Delete(deviceID)
}

func updateDevice(deviceID string, deviceEventData model.DeviceEvent) {
	var newDev model.Device
	dev, ok := syncMap.Load(deviceID)
	if ok{
		newDev = dev.(model.Device)
	}
	if deviceEventData.DeviceName != "" {
		newDev.Name = deviceEventData.DeviceName
	}
	for k, v := range(deviceEventData.Attributes){
		if newDev.Attributes == nil{
			newDev.Attributes = make(map[string]model.Attribute)
		}
		newDev.Attributes[k] = v
	}
	syncMap.Store(deviceID, newDev)
}

func addDevice(deviceID string, device model.Device){
	syncMap.Store(deviceID, device)
}


//GetDeviceAttributeValue ...
func GetDeviceAttributeValue(attribute model.Attribute) (string, error) {
	v := attribute
	if v.IsEncrypt {
		decryptValue, err := rtspclient.DecKeytool(v.Value)
		if err == nil {
			return decryptValue, nil
		}
		return "", err
	}
	return v.Value,nil
}

//CheckRealCamera ...
func CheckRealCamera(dev model.Device) (bool, string) {
	var checkingLog string
	address, err := GetDeviceAttributeValue(dev.Attributes["address"])
	if err != nil {
		checkingLog = "Can not Get Device Attribute Value (CameraURL) " + err.Error()
		return false, checkingLog
	}
	CameraURL, err := GetDeviceAttributeValue(dev.Attributes["CameraURL"])
	if err != nil {
		checkingLog = "Can not Get Device Attribute Value (CameraURL) " + err.Error()
		return false, checkingLog
	}
	checkingLog, err = rtspclient.CheckMain(address, CameraURL)
	if err != nil {
		return false, checkingLog + " Error : " + err.Error()
	}
	return true, checkingLog
}

// CheckCameraStatus ...
func (m *Manager) CheckCameraStatus() {
	fmtLog := common.FormatLog("          --------------- Checking device status ... --------------")
	glog.Infof(fmtLog)
	devNum := 0
	done := make(chan bool)
	syncMap.Range(func(key, value interface{}) bool {
		devNum++
		id := key.(string)
		device := value.(model.Device)
		go func(deviceId string, dev model.Device) {
			copyDev := getDevice(deviceId)
			var ret bool
			if config.CKconfig.Remote {
				var checkLog string
				copyDev.State, checkLog = CheckRealCamera(dev)
				fmtLog := common.FormatLog(fmt.Sprintf("Device(%q) , check result : (%t) , detail : %q ", dev.ID, copyDev.State, checkLog))
				glog.Infof(fmtLog)
			} else {
				statusCode := rand.Intn(100)
				fmtLog := common.FormatLog(fmt.Sprintf("Device (%q) response : (%d)", dev.ID, statusCode))
				glog.Infof(fmtLog)
				if statusCode%2 == 0 {
					ret = true
				} else {
					ret = false
				}
				copyDev.State = ret
			}
			addDevice(deviceId, copyDev)
			done <- ret
		}(id, device)
		return true
	})
	for i := 0; i < devNum; i ++ {
		 <-done
	}
}

//StartServer ...
func StartServer() {
	m := &Manager{
		MQTTClient:       nil,
		FinishedInitLoad: false,
	}
	m.serverInit()
	cronTask := cron.New()
	cameraCheckInterval := config.CKconfig.CheckCameraSec
	glog.Infof("Begining to check device every (%d) seconds. \n", cameraCheckInterval)
	detectSpec := fmt.Sprintf("*/%d * * * * ?", cameraCheckInterval)
	cronTask.AddFunc(detectSpec, m.CheckWork)
	cronTask.Start()
	select {}
}

func (m *Manager) serverInit() {
	glog.Infoln("CameraChecker Server init...")
	m.MQTTClient = mqtt_client.ConnectMQTTClient()
	NodeID := config.CKconfig.NodeID
	if NodeID == "" {
		glog.Fatalf("Node id is nil (%q) , server init failed", NodeID)
	}
	glog.Infof("Try to get membership of node (%q) Publishing topic: (%q) \n", NodeID, config.TopicGetDevices)
	var detailGet model.EdgeGet
	detailGet.EventID = common.GetUUID()
	cont, _ := json.Marshal(detailGet)
	time.Sleep(3*time.Second)
	m.MQTTClient.Publish(config.TopicGetDevices, string(cont))
	glog.Infof("Subscribing topic (%q) , try to geting devices info result \n", config.TopicGetDevicesResult)
	m.MQTTClient.Subscribe(config.TopicGetDevicesResult, func(mqtt MQTT.Client, msg MQTT.Message) {
		glog.Infof("Subscribed topic: (%q), with msg: (%q) successfully ,got devices info result , \n", config.TopicGetDevicesResult, msg.Payload())
		topic := msg.Topic()
		payload := msg.Payload()
		go m.DealCallbackMsg(topic, payload)
	})
	glog.Infoln("Loading devices info ...")
	RetryTime := 1
	for {
		time.Sleep(3 * 1e9)
		if m.FinishedInitLoad {
			glog.Infof("Finished Init Load Msg: %t ", m.FinishedInitLoad)
			break
		}
		fmt.Printf("Retry to loading devices info (%d) : ", RetryTime)
		go func() {
			token := m.MQTTClient.Publish(config.TopicGetDevices, string(cont))
			if token.Wait() && token.Error() != nil {
				glog.Errorf("Publish msg ERROR, topic: %s\n", config.TopicGetDevices)
			} else {
				glog.Infof("Published msg (%q) successfully", config.TopicGetDevices)
			}
		}()
		RetryTime++
	}
	glog.Infoln("Server initialed , finished load devices info")

	m.subscribeNodeUpdate(NodeID)
	glog.Infoln("Server finished initial. \n")
}

//CheckWork ...
func (m *Manager)CheckWork() {
	glog.Infoln(" ============================ Begin Checking ======================================")
	fmtLog := common.FormatLog(fmt.Sprintf(" %s Checking camera status scheduler timestamp (%d)", time.Now().String(), time.Now().Unix()))
	glog.Infof(fmtLog)
	m.CheckCameraStatus()
	fmtLog = common.FormatLog(fmt.Sprintf("        ------------ Updating Camera Devices Status : ------------"))
	glog.Infof(fmtLog)
	syncMap.Range(func(key, value interface{}) bool {
		k, v := key.(string), value.(model.Device)
		fmtLog := common.FormatLog(fmt.Sprintf("Device (%q) status : (%#v) ", k, v))
		glog.Infof(fmtLog)
		deviceTwinData := &model.DeviceTwinEvent{}
		deviceTwinData.EventType = config.DeviceTwinEventType
		deviceTwinData.DeviceName = v.Name
		deviceTwinData.DeviceID = v.ID
		deviceTwinData.Operation = config.UpdatedOperationType
		deviceTwinData.Timestamp = time.Now().UnixNano()
		deviceTwinData.Twin.Actual = make(map[string]string)
		deviceTwinData.Twin.Actual["state"] = strconv.FormatBool(v.State)

		deviceJSON, _ := json.Marshal(deviceTwinData)
		updatedDeviceTwinTopic := fmt.Sprintf(config.TopicUpdateTwinDevice, deviceTwinData.DeviceID)
		fmtLog = common.FormatLog(fmt.Sprintf("Publishing Devices Status , topic :(%q) msg: (%#v)", updatedDeviceTwinTopic, string(deviceJSON)))
		glog.Infof(fmtLog)
		go m.MQTTClient.Publish(updatedDeviceTwinTopic, string(deviceJSON))
		return true
	})
	glog.Infoln("============================== Finished Checking ===================================")
}

// DealCallbackMsg deal messages
func (m *Manager) DealCallbackMsg(topic string, payload []byte) error {
	baseEventData := &model.BaseEvent{}
	err := json.Unmarshal(payload, baseEventData)
	if err != nil {
		glog.Errorf("Json parse ERROR, topic: (%s) .", topic)
		return err
	}
	switch baseEventData.EventType {
	case config.GroupEventType:
		groupEventData := &model.GroupMembershipEvent{}
		err := json.Unmarshal(payload, groupEventData)
		if err != nil {
			glog.Errorf("json parse topic:%s data failed.", topic)
		}
		if groupEventData.MemberShip.Devices != nil {
			for i := range groupEventData.MemberShip.Devices {
				device := groupEventData.MemberShip.Devices[i]
				addDevice(device.ID, device)
				glog.Infof("Get devices msg , device : (%s).", device.ID)
				go m.subscribeDeviceUpdate(device.ID)
			}
			m.FinishedInitLoad = true
		}
		for i := range groupEventData.MemberShip.AddedDevices {
			device := groupEventData.MemberShip.AddedDevices[i]
			glog.Infof("Get add devices msg , device : (%s)", device.ID)
			addDevice(device.ID, device)
			go m.subscribeDeviceUpdate(device.ID)
		}
		for i := range groupEventData.MemberShip.RemovedDevices {
			device := groupEventData.MemberShip.RemovedDevices[i]
			glog.Infof("Get delete devices msg , device : (%s)", device.ID)
			deleteDevice(device.ID)
		}
		return err
	default:
		return nil
	}
}

func (m *Manager) subscribeDeviceUpdate(deviceID string) {
	updatedDeviceTopic := fmt.Sprintf(config.TopicUpdatedDevice, deviceID)
	deletedDeviceTopic := fmt.Sprintf(config.TopicDeletedDevice, deviceID)
	glog.Infof("Subscribing device update , updating device (%q) info , topic:(%q) and (%q)\n", deviceID, updatedDeviceTopic, deletedDeviceTopic)
	m.MQTTClient.Subscribe(updatedDeviceTopic, func(mqtt MQTT.Client, msg MQTT.Message) {
		m.DealUpdateDeviceMsg(msg.Payload(), deviceID)
	})
	m.MQTTClient.Subscribe(deletedDeviceTopic, func(mqtt MQTT.Client, msg MQTT.Message) {
		m.DealDeleteDeviceMsg(msg.Payload(), deviceID)
	})
}

//DealUpdateDeviceMsg ...
func (m *Manager)DealUpdateDeviceMsg(msg []byte, deviceID string){
	glog.Infof("Updating device (%q) , msg detail :(%q)\n", deviceID, string(msg))
	deviceEventData := model.DeviceEvent{}
	err := json.Unmarshal(msg, &deviceEventData)
	if err != nil {
		glog.Errorf("Json parse ERROR, topic:%s ,err %q.", string(msg), err)
	}
	updateDevice(deviceID, deviceEventData)
}

//DealDeleteDeviceMsg ...
func (m *Manager)DealDeleteDeviceMsg(msg []byte, deviceID string) {
	glog.Infof("Deleting device (%q) , msg detail :(%q)\n", deviceID, string(msg))
	deviceEventData := &model.DeviceEvent{}
	err := json.Unmarshal(msg, deviceEventData)
	if err != nil {
		glog.Errorf("Json parse ERROR, topic: %s ,err : %s.", string(msg), err.Error())
	}
	deleteDevice(deviceID)
}

func (m *Manager) subscribeNodeUpdate(NodeID string) {
	glog.Infof("Subscribing topic (%q) , update all devices in node (%q)\n", config.TopicUpdatedDevices, NodeID)
	m.MQTTClient.Subscribe(config.TopicUpdatedDevices, func(mqtt MQTT.Client, msg MQTT.Message){
		m.DealUpdateDevices(msg.Payload())
	})
}

//DealUpdateDevices ...
func (m *Manager)DealUpdateDevices(msg []byte) {
	glog.Infof("Subscribed TopicUpdatedDevices with msg: (%q) successfully ,updating all devices info in node\n", string(msg))
	groupEventData := model.GroupMembershipEvent{}
	err := json.Unmarshal(msg, &groupEventData)
	if err != nil {
		glog.Errorf("Json parse ERROR, topic:%s ,err %s.", string(msg), err.Error())
	}
	for i := range groupEventData.MemberShip.AddedDevices {
		device := groupEventData.MemberShip.AddedDevices[i]
		glog.Infof("subscribe add device id:%s", device.ID)
		addDevice(device.ID, device)
	}
	for i := range groupEventData.MemberShip.RemovedDevices {
		device := groupEventData.MemberShip.RemovedDevices[i]
		glog.Infof("Begining deleting device (%q)", device.ID)
		deleteDevice(device.ID)
	}
	syncMap.Range(func(key, value interface{}) bool {
		devID, _ := key.(string), value.(model.Device)
		go m.subscribeDeviceUpdate(devID)
		return true
	})
}