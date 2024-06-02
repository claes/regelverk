package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initBridges(mqttClient mqtt.Client, bridgeConfig BridgeConfig) {
	initCECBridge(CreateCECBridge(mqttClient))
	initRotelBridge(CreateRotelBridge(bridgeConfig.rotelSerialPort, mqttClient))
	initSamsungBridge(CreateSamsungBridge(bridgeConfig.samsungTvAddress, mqttClient))
	initMPDBridge(CreateMPDBridge(bridgeConfig.mpdServer, bridgeConfig.mpdPassword, mqttClient))
}
