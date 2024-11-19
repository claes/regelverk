package regelverk

import (
	"log/slog"

	rotelmqtt "github.com/claes/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RotelBridgeWrapper struct {
	bridge *rotelmqtt.RotelMQTTBridge
}

func (l *RotelBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	slog.Debug("Preparing rotel bridge", "config", config, "mqttClient", mqttClient)
	port, err := rotelmqtt.CreateSerialPort(config.RotelSerialPort)
	if err != nil {
		slog.Error("Could not serial port connection for rotel bridge", "error", err)
		return err
	}
	slog.Debug("Creating rotel bridge", "port", port, "mqttClient", mqttClient)
	l.bridge = rotelmqtt.NewRotelMQTTBridge(port, mqttClient)
	slog.Debug("Initialized rotel bridge", "bridge", l.bridge, "mqttClient", mqttClient)
	return nil
}

func (l *RotelBridgeWrapper) Run() error {
	slog.Debug("Starting rotel bridge", "bridge", l.bridge)
	go l.bridge.SerialLoop()
	slog.Debug("Rotel bridge started")
	return nil
}
