package mqtt_client

import (
	"glog"
	"rtsp_server/pkg/config"
	"rtsp_server/pkg/common"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"time"
	"fmt"
)



func ConnectMQTTClient() Client {
	retriesNumber := config.CKconfig.MqttRetries
	for {
		if retriesNumber > 0 {
			mqqtclient, err := newMQTTClient()
			if err != nil {
				retriesNumber--
				glog.Errorln("Could not connect to MQTT, retry...", err)
				continue
			}
			return mqqtclient
			break
		} else {
			glog.Infoln("Retry to connect MQTT...")
			time.Sleep(60 * 1e9)
			retriesNumber = config.CKconfig.MqttRetries
			continue
		}
	}
	return nil
}

func newMQTTClient() (Client, error) {
	mqqtURL := config.CKconfig.MqttURL
	glog.Infof("Connecting to MQTT (%q)", mqqtURL)
	mqttOpts := MQTT.NewClientOptions()
	mqttOpts.AutoReconnect = false

	broker := fmt.Sprintf("tcp://%s", mqqtURL)
	mqttOpts.AddBroker(broker)
	clientID := common.GetClientID()
	glog.Infof("MQTT Client ID : (%q)", clientID)
	mqttOpts.SetClientID(clientID)

	mqttOpts.SetKeepAlive(30 * time.Second)
	mqttOpts.SetPingTimeout(10 * time.Second)

	mqttOpts.SetCleanSession(false)


	client := &DefaultClient{
		Mqtt: MQTT.NewClient(mqttOpts),
	}
	mqttClient := Client(client)

	var err = mqttClient.Connect()
	if err != nil {
		glog.Errorf("Connect to MQTT error : %s.\n", err.Error())
	} else {
		glog.Infof("Connect to MQTT Success.\n")
	}
	return mqttClient, err
}
