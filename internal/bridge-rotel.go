package regelverk

import (
	"context"
	"log/slog"

	rotelmqtt "github.com/claes/mqtt-bridges/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RotelBridgeWrapper struct {
	bridge *rotelmqtt.RotelMQTTBridge
}

func (l *RotelBridgeWrapper) String() string {
	return "RotelBridgeWrapper"
}

func (l *RotelBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	rotelConfig := rotelmqtt.RotelClientConfig{SerialDevice: config.RotelSerialPort}
	l.bridge, err = rotelmqtt.NewRotelMQTTBridge(rotelConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *RotelBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting rotel bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
