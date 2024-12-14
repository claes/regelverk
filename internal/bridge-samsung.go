package regelverk

import (
	"context"

	samsungmqtt "github.com/claes/samsung-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SamsungBridgeWrapper struct {
	bridge samsungmqtt.SamsungRemoteMQTTBridge
}

func (l *SamsungBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	l.bridge = *samsungmqtt.NewSamsungRemoteMQTTBridge(&config.SamsungTvAddress, mqttClient, config.MQTTTopicPrefix)
	return nil
}

func (l *SamsungBridgeWrapper) Run(context context.Context) error {
	go l.bridge.MainLoop()
	return nil
}
