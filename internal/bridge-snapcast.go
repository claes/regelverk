package regelverk

import (
	"log/slog"

	snapcastmqtt "github.com/claes/snapcast-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SnapcastBridgeWrapper struct {
	bridge *snapcastmqtt.SnapcastMQTTBridge
}

func (l *SnapcastBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	snapConfig := snapcastmqtt.SnapClientConfig{SnapServerAddress: config.SnapcastServer}
	l.bridge, err = snapcastmqtt.NewSnapcastMQTTBridge(snapConfig, mqttClient, config.MQTTTopicPrefix)
	if err != nil {
		slog.Error("Could not create snapcast bridge", "error", err)
		return err
	}
	return nil
}

func (l *SnapcastBridgeWrapper) Run() error {
	go l.bridge.MainLoop()
	slog.Info("Snapcast bridge started")
	return nil
}
