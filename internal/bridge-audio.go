package regelverk

import (
	"context"
	"embed"
	"log/slog"

	audiomqtt "github.com/claes/mqtt-bridges/audio-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

//go:embed assets/ping.wav
var audioFiles embed.FS

type AudioBridgeWrapper struct {
	bridge *audiomqtt.AudioMQTTBridge
}

func (l *AudioBridgeWrapper) String() string {
	return "AudioBridgeWrapper"
}

func (l *AudioBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error
	audioConfig := audiomqtt.AudioConfig{EmbeddedFiles: audioFiles}
	l.bridge, err = audiomqtt.NewAudioMQTTBridge(audioConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *AudioBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting audio bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
