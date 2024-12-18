package regelverk

import (
	"log/slog"

	pulsemqtt "github.com/claes/pulseaudio-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type PulseaudioBridgeWrapper struct {
	bridge *pulsemqtt.PulseaudioMQTTBridge
}

func (l *PulseaudioBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	pulseclient, err := pulsemqtt.CreatePulseClient(config.Pulseserver)
	if err != nil {
		slog.Error("Could not create pulse client", "error", err)
		return err
	}
	l.bridge = pulsemqtt.NewPulseaudioMQTTBridge(pulseclient, mqttClient, config.MQTTTopicPrefix)
	return nil
}

func (l *PulseaudioBridgeWrapper) Run() error {
	go l.bridge.MainLoop()
	slog.Info("Pulseaudio bridge started")
	return nil
}
