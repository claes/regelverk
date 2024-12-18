package regelverk

import (
	"context"

	pulsemqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type PulseaudioBridgeWrapper struct {
	bridge *pulsemqtt.PulseaudioMQTTBridge
}

func (l *PulseaudioBridgeWrapper) String() string {
	return "PulseaudioBridgeWrapper"
}

func (l *PulseaudioBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	pulseClientConfig := pulsemqtt.PulseClientConfig{PulseServerAddress: config.Pulseserver}
	l.bridge, err = pulsemqtt.NewPulseaudioMQTTBridge(pulseClientConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *PulseaudioBridgeWrapper) Run(ctx context.Context) error {
	l.bridge.EventLoop(ctx)
	return nil
}
