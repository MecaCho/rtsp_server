package edgehub

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	qosClient  = 0x02
	)

type Client interface {
	Connect() error
	Subscribe(topic string, f func(mqtt MQTT.Client, msg MQTT.Message)) MQTT.Token
	Publish(topic string, msg string) MQTT.Token
}

type DefaultClient struct {
	Mqtt MQTT.Client
}

func (c *DefaultClient)Connect() error{
	token := c.Mqtt.Connect()
	if token.Error() != nil{
		return token.Error()
	}
	return nil
}

func (c *DefaultClient)Subscribe(topic string, f func(mqtt MQTT.Client, msg MQTT.Message)) MQTT.Token {
	return c.Mqtt.Subscribe(topic, qosClient, f)
}

func (c *DefaultClient)Publish(topic string, msg string) MQTT.Token {
	return c.Mqtt.Publish(topic, qosClient, false, msg)

}
