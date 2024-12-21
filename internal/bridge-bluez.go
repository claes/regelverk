package regelverk

import (
	"context"
	"log/slog"

	bluezmqtt "github.com/claes/mqtt-bridges/bluez-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BluezBridgeWrapper struct {
	bridge *bluezmqtt.BluezMediaPlayerMQTTBridge
}

func (l *BluezBridgeWrapper) String() string {
	return "BluezBridgeWrapper"
}

func (l *BluezBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	bluezConfig := bluezmqtt.BluezMediaPlayerConfig{BluetoothMACAddress: config.BluetoothAddress}
	l.bridge, err = bluezmqtt.NewBluezMediaPlayerMQTTBridge(bluezConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *BluezBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting Bluez bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
