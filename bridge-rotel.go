package main

import (
	"log/slog"

	rotelmqtt "github.com/claes/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type rotelBridgeWrapper struct {
	bridge *rotelmqtt.RotelMQTTBridge
}

func (l rotelBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	port, err := rotelmqtt.CreateSerialPort(config.rotelSerialPort)
	if err != nil {
		slog.Error("Could not serial port connection for rotel bridge", "error", err)
		return err
	}

	l.bridge = rotelmqtt.NewRotelMQTTBridge(port, mqttClient)
	return nil
}

func (l rotelBridgeWrapper) Run() error {
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
