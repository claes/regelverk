package main

import (
	samsungmqtt "github.com/claes/samsung-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type samsungBridgeWrapper struct {
	bridge samsungmqtt.SamsungRemoteMQTTBridge
}

func (l samsungBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	l.bridge = *samsungmqtt.NewSamsungRemoteMQTTBridge(&config.samsungTvAddress, mqttClient)
	return nil
}

func (l samsungBridgeWrapper) Run() error {
	go l.bridge.MainLoop()
	return nil
}

// // ---
// func CreateSamsungBridge(tvIPAddress string, mqttClient mqtt.Client) *samsungmqtt.SamsungRemoteMQTTBridge {
// 	bridge := samsungmqtt.NewSamsungRemoteMQTTBridge(&tvIPAddress, mqttClient)
// 	return bridge
// }

// func initSamsungBridge(bridge *samsungmqtt.SamsungRemoteMQTTBridge) {
// 	go bridge.MainLoop()
// }
