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
	l.bridge = pulsemqtt.NewPulseaudioMQTTBridge(pulseclient, mqttClient)
	return nil
}

func (l *PulseaudioBridgeWrapper) Run() error {
	go l.bridge.MainLoop()
	slog.Info("Pulseaudio bridge started")
	return nil
}

// -----------------------------

// func CreatePulseaudioBridge(pulseserver string, mqttClient mqtt.Client) (*pulsemqtt.PulseaudioMQTTBridge, error) {
// 	pulseclient, err := pulsemqtt.CreatePulseClient(pulseserver)
// 	if err != nil {
// 		slog.Error("Could not create pulse client", "error", err)
// 		return nil, err
// 	}
// 	bridge := pulsemqtt.NewPulseaudioMQTTBridge(pulseclient, mqttClient)
// 	return bridge, nil
// }

// func initPulseaudioBridge(bridge *pulsemqtt.PulseaudioMQTTBridge) {
// 	go bridge.MainLoop()
// 	slog.Info("Pulseaudio bridge started")
// }
