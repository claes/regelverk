package regelverk

import (
	"context"

	samsungmqtt "github.com/claes/mqtt-bridges/samsungtv-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SamsungBridgeWrapper struct {
	bridge *samsungmqtt.SamsungTVRemoteMQTTBridge
}

func (l *SamsungBridgeWrapper) String() string {
	return "SamsungBridgeWrapper"
}

func (l *SamsungBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	samsungConfig := samsungmqtt.SamsungTVClientConfig{TVIPAddress: config.SamsungTvAddress}
	var err error
	l.bridge, err = samsungmqtt.NewSamsungTVRemoteMQTTBridge(samsungConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *SamsungBridgeWrapper) Run(ctx context.Context) error {
	go l.bridge.EventLoop(ctx)
	return nil
}
