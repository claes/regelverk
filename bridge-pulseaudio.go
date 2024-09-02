package main

import (
	"log/slog"

	pulsemqtt "github.com/claes/pulseaudio-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreatePulseaudioBridge(pulseserver string, mqttClient mqtt.Client) (*pulsemqtt.PulseaudioMQTTBridge, error) {
	pulseclient, err := pulsemqtt.CreatePulseClient(pulseserver)
	if err != nil {
		slog.Error("Could not create pulse client", "error", err)
		return nil, err
	}
	bridge := pulsemqtt.NewPulseaudioMQTTBridge(pulseclient, mqttClient)
	return bridge, nil
}

func initPulseaudioBridge(bridge *pulsemqtt.PulseaudioMQTTBridge) {
	go bridge.MainLoop()
	slog.Info("Pulseaudio bridge started")
}
