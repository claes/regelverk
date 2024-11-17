package regelverk

import (
	"log/slog"

	rotelmqtt "github.com/claes/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RotelBridgeWrapper struct {
	bridge *rotelmqtt.RotelMQTTBridge
}

// 	rotelBridge, err := CreateRotelBridge(config.rotelSerialPort, mqttClient)
// 	if err != nil {
// 		slog.Error("Could not create rotel bridge", "error", err)
// 	} else {
// 		initRotelBridge(rotelBridge)
// 	}

func (l *RotelBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	slog.Info("Preparing rotel bridge", "config", config, "mqttClient", mqttClient)
	port, err := rotelmqtt.CreateSerialPort(config.RotelSerialPort)
	if err != nil {
		slog.Error("Could not serial port connection for rotel bridge", "error", err)
		return err
	}
	slog.Info("Creating rotel bridge", "port", port, "mqttClient", mqttClient)
	l.bridge = rotelmqtt.NewRotelMQTTBridge(port, mqttClient)
	slog.Info("Initialized rotel bridge", "bridge", l.bridge, "mqttClient", mqttClient)
	return nil
}

func (l *RotelBridgeWrapper) Run() error {
	slog.Info("Starting rotel bridge", "bridge", l.bridge)
	go l.bridge.SerialLoop()
	slog.Info("Rotel bridge started")
	return nil
}

////----

// func CreateRotelBridge(serialPort string, mqttClient mqtt.Client) (*rotelmqtt.RotelMQTTBridge, error) {
// 	port, err := rotelmqtt.CreateSerialPort(serialPort)
// 	if err != nil {
// 		slog.Error("Could not serial port connection for rotel bridge", "error", err)
// 		return nil, err
// 	}

// 	bridge := rotelmqtt.NewRotelMQTTBridge(port, mqttClient)
// 	return bridge, err
// }

// func initRotelBridge(bridge *rotelmqtt.RotelMQTTBridge) {
// 	go bridge.SerialLoop()
// 	slog.Info("Rotel bridge started")
// }
