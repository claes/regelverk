package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initBridges(mqttClient mqtt.Client) {
	initCECBridge(CreateCECBridge(mqttClient))
}
