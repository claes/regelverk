package regelverk

import (
	"context"

	snapcastmqtt "github.com/claes/mqtt-bridges/snapcast-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SnapcastBridgeWrapper struct {
	bridge *snapcastmqtt.SnapcastMQTTBridge
}

func (l *SnapcastBridgeWrapper) String() string {
	return "SnapcastBridgeWrapper"
}

func (l *SnapcastBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	snapConfig := snapcastmqtt.SnapClientConfig{SnapServerAddress: config.SnapcastServer}
	l.bridge, err = snapcastmqtt.NewSnapcastMQTTBridge(snapConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *SnapcastBridgeWrapper) Run(ctx context.Context) error {
	l.bridge.EventLoop(ctx)
	return nil
}
