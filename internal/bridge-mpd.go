package regelverk

import (
	"context"
	"log/slog"

	mpdmqtt "github.com/claes/mpd-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MpdBridgeWrapper struct {
	bridge      *mpdmqtt.MpdMQTTBridge
	config      Config
	mpdPassword string
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
	//slog.Debug("MPD password", "password", mpdPassword)

	mpdClient, mpdWatcher, err := mpdmqtt.CreateMPDClient(config.MpdServer, mpdPassword)
	if err != nil {
		slog.Error("Could not create MPD client", "error", err, "mpdserver", config.MpdServer, "mpdpassword", mpdPassword)
	}

	l.mpdPassword = mpdPassword
	l.bridge = mpdmqtt.NewMpdMQTTBridge(mpdClient, mpdWatcher, mqttClient, config.MQTTTopicPrefix)

	return nil
}

func (l *MpdBridgeWrapper) Run(context context.Context) error {
	go func() {
		l.bridge.DetectReconnectMPDClient(l.config.MpdServer, l.mpdPassword)
	}()

	go l.bridge.MainLoop()
	return nil
}
