package main

import (
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initBridges(mqttClient mqtt.Client, config Config) {

	rotelBridge, err := CreateRotelBridge(config.rotelSerialPort, mqttClient)
	if err != nil {
		slog.Error("Could not create rotel bridge", "error", err)
	} else {
		initRotelBridge(rotelBridge)
	}

	pulseBridge, err := CreatePulseaudioBridge(config.pulseserver, mqttClient)
	if err != nil {
		slog.Error("Could not create pulseaudio bridge", "error", err)
	} else {
		initPulseaudioBridge(pulseBridge)
	}

	initCECBridge(mqttClient)
	initSamsungBridge(CreateSamsungBridge(config.samsungTvAddress, mqttClient))
	initMPDBridge(CreateMPDBridge(config.mpdServer, config.mpdPassword, mqttClient))
}
