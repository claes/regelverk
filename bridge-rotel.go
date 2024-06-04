package main

import (
	"log/slog"

	rotelmqtt "github.com/claes/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateRotelBridge(serialPort string, mqttClient mqtt.Client) (*rotelmqtt.RotelMQTTBridge, error) {
	port, err := rotelmqtt.CreateSerialPort(serialPort)
	if err != nil {
		slog.Error("Could not serial port connection for rotel bridge", "error", err)
		return nil, err
	}

	bridge := rotelmqtt.NewRotelMQTTBridge(port, mqttClient)
	return bridge, err
}

func initRotelBridge(bridge *rotelmqtt.RotelMQTTBridge) {
	go bridge.SerialLoop()
}
