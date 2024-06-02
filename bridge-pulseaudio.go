package main

import (
	pulsemqtt "github.com/claes/pulseaudio-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreatePulseaudioBridge(pulseserver string, mqttClient mqtt.Client) *pulsemqtt.PulseaudioMQTTBridge {
	bridge := pulsemqtt.NewPulseaudioMQTTBridge(pulsemqtt.CreatePulseClient(pulseserver),
		mqttClient)
	return bridge
}

func initPulseaudioBridge(bridge *pulsemqtt.PulseaudioMQTTBridge) {
	go bridge.MainLoop()
}
