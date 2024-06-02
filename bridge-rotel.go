package main

import (
	rotelmqtt "github.com/claes/rotel-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateRotelBridge(serialPort string, mqttClient mqtt.Client) *rotelmqtt.RotelMQTTBridge {
	bridge := rotelmqtt.NewRotelMQTTBridge(rotelmqtt.CreateSerialPort(serialPort), mqttClient)
	return bridge
}

func initRotelBridge(bridge *rotelmqtt.RotelMQTTBridge) {
	go bridge.SerialLoop()
}
