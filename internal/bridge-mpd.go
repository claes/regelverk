package regelverk

import (
	"context"
	"log/slog"

	mpdmqtt "github.com/claes/mqtt-bridges/mpd-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MpdBridgeWrapper struct {
	bridge          *mpdmqtt.MpdMQTTBridge
	config          Config
	mpdClientConfig mpdmqtt.MpdClientConfig
}

func (l *MpdBridgeWrapper) String() string {
	return "MpdBridgeWrapper"
}

func (l *MpdBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	mpdPassword, err := fileToString(config.MpdPasswordFile)
	if err != nil {
		slog.Error("Error reading mpd password",
			"mpdPasswordFile", config.MpdPasswordFile, "error", err)
	}

	l.mpdClientConfig = mpdmqtt.MpdClientConfig{MpdServer: config.MpdServer, MpdPassword: mpdPassword}

	l.bridge, err = mpdmqtt.NewMpdMQTTBridge(l.mpdClientConfig, mqttClient, config.MQTTTopicPrefix)
	// go func() {
	// 	l.bridge.DetectReconnectMPDClient(l.mpdClientConfig)
	// }()

	return nil
}

func (l *MpdBridgeWrapper) Run(ctx context.Context) error {

	l.bridge.EventLoop(ctx)
	return nil
}
