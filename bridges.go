package main

import (
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initBridges(mqttClient mqtt.Client, bridgeConfig BridgeConfig) {

	rotelBridge, err := CreateRotelBridge(bridgeConfig.rotelSerialPort, mqttClient)
	if err != nil {
		slog.Error("Could not create rotel bridge", "error", err)
	} else {
		initRotelBridge(rotelBridge)
	}

	pulseBridge, err := CreatePulseaudioBridge(bridgeConfig.pulseserver, mqttClient)
	if err != nil {
		slog.Error("Could not create pulseaudio bridge", "error", err)
	} else {
		initPulseaudioBridge(pulseBridge)
	}

	//initCECBridge(CreateCECBridge(mqttClient))
	initCECBridgeNew(mqttClient)
	initSamsungBridge(CreateSamsungBridge(bridgeConfig.samsungTvAddress, mqttClient))
	initMPDBridge(CreateMPDBridge(bridgeConfig.mpdServer, bridgeConfig.mpdPassword, mqttClient))
}
