package main

import (
	samsungmqtt "github.com/claes/samsung-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateSamsungBridge(tvIPAddress string, mqttClient mqtt.Client) *samsungmqtt.SamsungRemoteMQTTBridge {
	bridge := samsungmqtt.NewSamsungRemoteMQTTBridge(&tvIPAddress, mqttClient)
	return bridge
}

func initSamsungBridge(bridge *samsungmqtt.SamsungRemoteMQTTBridge) {
	go bridge.MainLoop()
}
