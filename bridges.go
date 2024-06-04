package main

import (
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initBridges(mqttClient mqtt.Client, bridgeConfig BridgeConfig) {
	initCECBridge(CreateCECBridge(mqttClient))
	initRotelBridge(CreateRotelBridge(bridgeConfig.rotelSerialPort, mqttClient))

	pulseBridge, err := CreatePulseaudioBridge(bridgeConfig.pulseserver, mqttClient)
	if err != nil {
		slog.Error("Could not create pulseaudio bridge", "error", err)
	} else {
		initPulseaudioBridge(pulseBridge)
	}
	initSamsungBridge(CreateSamsungBridge(bridgeConfig.samsungTvAddress, mqttClient))
	initMPDBridge(CreateMPDBridge(bridgeConfig.mpdServer, bridgeConfig.mpdPassword, mqttClient))
}
