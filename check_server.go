package checker

import (
	"rtsp_server/pkg/config"
	"rtsp_server/pkg/edgehub"
	"rtsp_server/pkg/model"
	"rtsp_server/pkg/rtspclient"
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	//"github.com/golang/glog"
	"github.com/robfig/cron"
	"github.com/satori/go.uuid"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"glog"
)

// Manager rtsp Client and device list
type Manager struct {
	MQTTClient edgehub.Client
	Devices          map[string]model.Device //id -> device
	FinishedInitLoad bool
}

// NewRTSPClient ...
func NewRTSPClient(address string) (net.Conn, error) {
	con, err := net.Dial("tcp", address)
	if err != nil {
		glog.Errorf("connect to remote camera err: %s.", err.Error())
	}
	return con, err
}

func getHostName() string {
	ret, err := os.Hostname()
	if err != nil {
		return ""
	}
	return ret
}

//GetClientID get mqtt broker clientID
func GetClientID() string {
	timeNow := time.Now().String()
	tmpClient := []string{"camera_checker", getHostName(), timeNow}
	return strings.Join(tmpClient, "#")
}
//GetDeviceAttributeValue get attribute value according to key
func GetDeviceAttributeValue(key string, attributes []model.Attribute) (string, error) {
	for _,v := range(attributes){
		if v.Key == key{
			if v.IsEncrypt {
				decryptValue, err := rtspclient.DecKeytool(v.Value)
				if err == nil {
					return decryptValue, nil
				}
				return "", err
			}
			return v.Value,nil
		}
	}
	return "",fmt.Errorf("Get device attribute value error : %s",key)
}

func (m *Manager) deleteDevice(deviceID string) {
	delete(m.Devices, deviceID)
}

func (m *Manager) updateDevice(deviceID string, deviceEventData model.DeviceEvent) {
	dev := m.Devices[deviceID]
	if deviceEventData.DeviceName != "" {
		dev.Name = deviceEventData.DeviceName
	}
	dev.Attributes = deviceEventData.Attributes
	m.Devices[deviceID] = dev
}

//FormatLog gen format log output
func FormatLog(logStirng string) string {
	if len(logStirng) > 79 {
		lens := len(logStirng) / 80
		logLines := []string{}
		logLines = append(logLines, "||"+logStirng[:80]+"||")
		for i := 1; i < lens; i++ {
			logLines = append(logLines, "                                                     ||"+logStirng[i*80:i*80+80]+"||")
		}
		logLines = append(logLines, fmt.Sprintf("                                                     ||%-80s||", logStirng[lens*80:]))
		fmtLog := strings.Join(logLines, "\n")
		return fmtLog
	}
	fmtLog := fmt.Sprintf("||%-80s||", logStirng)
	return fmtLog
}

// DealCallbackMsg deal messages
func (m *Manager) DealCallbackMsg(topic string, payload []byte) error {
	baseEventData := &model.BaseEvent{}
	err := json.Unmarshal(payload, baseEventData)
	if err != nil {
		glog.Errorf("json parse topic:%s data failed.", topic)
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
			//Devices,设备初始化
			for i := range groupEventData.MemberShip.Devices {
				device := groupEventData.MemberShip.Devices[i]
				m.Devices[device.ID] = device
				glog.Infof("subscribe device id:%s", device.ID)
				go m.subscribeDeviceUpdate(device.ID)
			}
			m.FinishedInitLoad = true
		}
		//AddedDevices,增加设备
		for i := range groupEventData.MemberShip.AddedDevices {
			device := groupEventData.MemberShip.AddedDevices[i]
			glog.Infof("subscribe add device id:%s", device.ID)
			m.Devices[device.ID] = device
			glog.Infof("connect to remote c")
			go m.subscribeDeviceUpdate(device.ID)
		}
		//RemovedDevices , 删除设备
		for i := range groupEventData.MemberShip.RemovedDevices {
			device := groupEventData.MemberShip.RemovedDevices[i]
			glog.Infof("deleting device (%q)", device.ID)
			m.deleteDevice(device.ID)
		}
		return err
	default:
		return nil
	}
}

//CheckRealCamera ...
func CheckRealCamera(dev model.Device) (bool, string) {
	var checkingLog string
	address, err := GetDeviceAttributeValue("address", dev.Attributes)
	if err != nil {
		checkingLog = "Can not Get Device Attribute Value (CameraURL) " + err.Error()
		return false, checkingLog
	}
	CameraURL, err := GetDeviceAttributeValue("CameraURL", dev.Attributes)
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

// CheckCameraStatus check all camera status
func (m *Manager) CheckCameraStatus() {
	fmtLog := FormatLog("          --------------- Checking device status ... --------------")
	glog.Infof(fmtLog)
	devNum := len(m.Devices)
	var wg sync.WaitGroup
	wg.Add(devNum)
	devList := m.Devices
	for id, device := range devList {
		go func(deviceId string, dev model.Device) {
			copyDev := m.Devices[deviceId]
			defer wg.Done()
			var ret bool
			if config.CKconfig.Remote {
				var checkLog string
				copyDev.State, checkLog = CheckRealCamera(dev)
				fmtLog := FormatLog(fmt.Sprintf("#Device(%q) , check result : (%t) , detail : %q ", dev.ID, copyDev.State, checkLog))
				glog.Infof(fmtLog)
			} else {
				statusCode := rand.Intn(100)
				fmtLog := FormatLog(fmt.Sprintf("Device (%q) response : (%d)", dev.ID, statusCode))
				glog.Infof(fmtLog)
				if statusCode%2 == 0 {
					ret = true
				} else {
					ret = false
				}
				copyDev.State = ret
			}
			m.Devices[deviceId] = copyDev
		}(id, device)
	}
	wg.Wait()
}

//StartServer 启动服务
func StartServer() {
	m := &Manager{
		MQTTClient:       nil,
		Devices:          make(map[string]model.Device),
		FinishedInitLoad: false,
	}
	m.serverInit()
	//cronTask 定时发送请求检查摄像头状态
	cronTask := cron.New()
	cameraCheckInterval := config.CKconfig.CheckCameraSec
	glog.Infof("Begining to schedule every (%d) seconds \n", cameraCheckInterval)
	detectSpec := fmt.Sprintf("*/%d * * * * ?", cameraCheckInterval)
	cronTask.AddFunc(detectSpec, m.CheckWork)
	cronTask.Start()
	select {}
}
//CheckWork check worker
func (m *Manager)CheckWork() {
	glog.Infoln(" ============================ Begin Checking ======================================")
	fmtLog := FormatLog(fmt.Sprintf(" %s Checking camera status scheduler timestamp (%d)", time.Now().String(), time.Now().Unix()))
	glog.Infof(fmtLog)
	m.CheckCameraStatus()
	//tarval 更新状态
	fmtLog = FormatLog(fmt.Sprintf("        ------------ Updating Camera Devices Status : ------------"))
	glog.Infof(fmtLog)
	for k, v := range m.Devices {
		fmtLog := FormatLog(fmt.Sprintf("Device (%q) status : (%#v) ", k, v))
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
		updatedDeviceTopic := strings.Replace(config.TopicUpdateTwinDevice, "<deviceID>", deviceTwinData.DeviceID, -1)
		fmtLog = FormatLog(fmt.Sprintf("Publishing Devices Status , topic :(%q) msg: (%#v)", updatedDeviceTopic, string(deviceJSON)))
		glog.Infof(fmtLog)
		go m.MQTTClient.Publish(updatedDeviceTopic, string(deviceJSON))
	}
	glog.Infoln("============================== Finished Checking ===================================")
}

func (m *Manager) serverInit() {
	glog.Infoln("IEF-CameraChecker Server init...")
	//getMQTTClient 连接mqtt server
	m.getMQTTClient()
	edgeGroupID := config.CKconfig.EdgeGroupID
	if edgeGroupID == "" {
		glog.Fatalf("group id is nil (%q) , cameraChecker init failed", edgeGroupID)
	}
	getDevicesTopic := strings.Replace(config.TopicGetDevices, "<edgeGroupID>", edgeGroupID, -1)
	glog.Infof("Try to get membership of group (%q) Publishing topic: (%q) \n", edgeGroupID, getDevicesTopic)
	var detailGet model.EdgeGet
	uid, _ := uuid.NewV4()
	detailGet.EventID = uid.String()
	cont, _ := json.Marshal(detailGet)
	go func() {
		token := m.MQTTClient.Publish(getDevicesTopic, string(cont))
		if token.Wait() && token.Error() != nil {
			glog.Infof("Error in pubCloudMsgToEdge with topic: %s\n", getDevicesTopic)
		} else {
			glog.Infof("Published msg (%q) successfully", getDevicesTopic)
		}
	}()
	getDevicesResultTopic := strings.Replace(config.TopicGetDevicesResult, "<edgeGroupID>", config.CKconfig.EdgeGroupID, -1)
	glog.Infof("subscribing topic (%q) ,geting devices info result\n", getDevicesResultTopic)
	m.MQTTClient.Subscribe(getDevicesResultTopic, func(mqtt MQTT.Client, msg MQTT.Message) {
		glog.Infof("Subscribed topic: (%q) with msg: (%q) successfully ,got devices info result , \n", getDevicesResultTopic, msg.Payload())
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
			token := m.MQTTClient.Publish(getDevicesTopic, string(cont))
			if token.Wait() && token.Error() != nil {
				glog.Infof("Error in pubCloudMsgToEdge with topic: %s\n", getDevicesTopic)
			} else {
				fmt.Printf("Published msg (%q) successfully\n", getDevicesTopic)
			}
		}()
		RetryTime++
	}
	glog.Infoln("server init , finished load devices info")

	m.subscribeGroupUpdate(edgeGroupID)
	glog.Infoln("CameraChecker Finished init. \n")
}

func (m *Manager) getMQTTClient() {
	retriesNumber := config.CKconfig.MqttRetries
	for {
		if retriesNumber > 0 {
			mqqtclient, err := m.newMQTTClient()
			if err != nil {
				retriesNumber--
				glog.Errorln("Could not connect to MQTT, retry...", err)
				continue
			}
			m.MQTTClient = mqqtclient
			break
		} else {
			glog.Infoln("========================ERROR========================")
			glog.Infoln("Could not connect to MQTT, sleep 60s for continue.")
			glog.Infoln("=====================================================")
			time.Sleep(60 * 1e9)
			retriesNumber = config.CKconfig.MqttRetries
			continue
		}
	}
}

func (m *Manager) newMQTTClient() (edgehub.Client, error) {
	mqqtURL := config.CKconfig.MqttURL
	glog.Infof("Connecting to MQTT (%q)", mqqtURL)
	mqttOpts := MQTT.NewClientOptions()
	mqttOpts.AutoReconnect = false

	broker := fmt.Sprintf("tcp://%s", mqqtURL)
	mqttOpts.AddBroker(broker)
	clientID := GetClientID()
	glog.Infof("MQTT Client ID : (%q)", clientID)
	mqttOpts.SetClientID(clientID)
	mqttOpts.SetUsername(config.CKconfig.MqttUsername)
	mqttPwd := config.CKconfig.MqttPassword
	mqttOpts.SetPassword(mqttPwd)

	// TODO: Some tuning of these values probably won't hurt:
	mqttOpts.SetKeepAlive(30 * time.Second)
	mqttOpts.SetPingTimeout(10 * time.Second)

	// Usually this setting should not be used together with random ClientIDs, but
	// we configured The Things Network's MQTT servers to handle this correctly.
	mqttOpts.SetCleanSession(false)

	mqttOpts.SetConnectionLostHandler(func(client MQTT.Client, err error) {
		glog.Warning("Disconnected, reconnecting...")
		go m.getMQTTClient()
	})

	mqttOpts.SetOnConnectHandler(func(client MQTT.Client) {
		m.subscribeGroupUpdate(config.CKconfig.EdgeGroupID)
		glog.Info("MQTT Connected successfully")
	})

	mqttOpts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		glog.Infof("Recived TOPIC: %s\n", msg.Topic())
		glog.Infof("Received MSG: %s\n", msg.Payload())
		//topic，作为主题，添加发布消息后的信息回调处理。
		topic := msg.Topic()
		payload := msg.Payload()
		m.DealCallbackMsg(topic, payload)
	})

	client := &edgehub.DefaultClient{
		Mqtt: MQTT.NewClient(mqttOpts),
	}
	mqttClient := edgehub.Client(client)

	var err = mqttClient.Connect()
	if err != nil {
		glog.Infof("Could not connect to MQTT, error:%s.\n", err)
	} else {
		glog.Infof("Success connect to MQTT.\n")
	}
	return mqttClient, err
}

func (m *Manager) subscribeDeviceUpdate(deviceID string) {
	updatedDeviceTopic := strings.Replace(config.TopicUpdatedDevice, "<deviceID>", deviceID, -1)
	deletedDeviceTopic := strings.Replace(config.TopicDeletedDevice, "<deviceID>", deviceID, -1)
	glog.Infof("Subscribing device update , updating device (%q) info , topic:(%q) and (%q)\n", deviceID, updatedDeviceTopic, deletedDeviceTopic)
	//Subscribe，订阅设备更新
	m.MQTTClient.Subscribe(updatedDeviceTopic, func(mqtt MQTT.Client, msg MQTT.Message) {
		m.DealUpdateDeviceMsg(msg.Payload(), deviceID)
	})
	//Subscribe，订阅设备删除
	m.MQTTClient.Subscribe(deletedDeviceTopic, func(mqtt MQTT.Client, msg MQTT.Message) {
		m.DealDeleteDeviceMsg(msg.Payload(), deviceID)
	})
}

//DealUpdateDeviceMsg mqtt subscribe deal update device message
func (m *Manager)DealUpdateDeviceMsg(msg []byte, deviceID string){
	glog.Infof("Updating device (%q) , msg detail :(%q)\n", deviceID, string(msg))
	deviceEventData := model.DeviceEvent{}
	err := json.Unmarshal(msg, &deviceEventData)
	if err != nil {
		glog.Errorf("json parse updatedDeviceTopic data failed :%s , (%q)", string(msg), err)
	}
	m.updateDevice(deviceID, deviceEventData)
}

//DealDeleteDeviceMsg mqtt subscribe deal delete device message
func (m *Manager)DealDeleteDeviceMsg(msg []byte, deviceID string) {
	glog.Infof("Deleting device (%q) , msg detail :(%q)\n", deviceID, string(msg))
	deviceEventData := &model.DeviceEvent{}
	err := json.Unmarshal(msg, deviceEventData)
	if err != nil {
		glog.Errorf("json parse topic:%s data failed, %q.", string(msg), err)
	}
	m.deleteDevice(deviceID)
}

func (m *Manager) subscribeGroupUpdate(groupID string) {
	updateDevicesTopic := strings.Replace(config.TopicUpdatedDevices, "<edgeGroupID>", groupID, -1)
	glog.Infof("Subscribing topic (%q) , update all devices in group (%q)\n", updateDevicesTopic, groupID)
	m.MQTTClient.Subscribe(updateDevicesTopic, func(mqtt MQTT.Client, msg MQTT.Message){
		m.DealUpdateDevices(msg.Payload())
	})
}

//DealUpdateDevices mqtt subscribe deal update group message
func (m *Manager)DealUpdateDevices(msg []byte) {
	glog.Infof("Subscribed updateDevicesTopic with msg: (%q) successfully ,updating all devices info in group\n", string(msg))
	groupEventData := model.GroupMembershipEvent{}
	fmt.Printf("GroupMembershipEvent msg : %q \n", string(msg))
	err := json.Unmarshal(msg, &groupEventData)
	if err != nil {
		glog.Errorf("json parse msg: %s data failed, %q.", string(msg), err)
	}
	//AddedDevices,添加设备
	for i := range groupEventData.MemberShip.AddedDevices {
		device := groupEventData.MemberShip.AddedDevices[i]
		glog.Infof("subscribe add device id:%s", device.ID)
		m.Devices[device.ID] = device
	}
	//RemovedDevices,删除设备
	for i := range groupEventData.MemberShip.RemovedDevices {
		device := groupEventData.MemberShip.RemovedDevices[i]
		glog.Infof("Begining deleting device (%q)", device.ID)
		m.deleteDevice(device.ID)
	}
	//tarval,遍历设备 , 订阅设备更新
	for k := range m.Devices {
		go m.subscribeDeviceUpdate(m.Devices[k].ID)
	}
}