package main

import (
	"log/slog"

	mpdmqtt "github.com/claes/mpd-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mpdBridgeWrapper struct {
	bridge      *mpdmqtt.MpdMQTTBridge
	config      Config
	mpdPassword string
}

func (l *mpdBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	mpdPassword, err := fileToString(config.mpdPasswordFile)
	if err != nil {
		slog.Error("Error reading mpd password",
			"mpdPasswordFile", config.mpdPasswordFile, "error", err)
	}
	slog.Info("MPD password", "password", mpdPassword)

	mpdClient, mpdWatcher, err := mpdmqtt.CreateMPDClient(config.mpdServer, mpdPassword)
	if err != nil {
		slog.Error("Could not create MPD client", "error", err, "mpdserver", config.mpdServer, "mpdpassword", mpdPassword)
	}

	l.mpdPassword = mpdPassword
	l.bridge = mpdmqtt.NewMpdMQTTBridge(mpdClient, mpdWatcher, mqttClient)

	return nil
}

func (l *mpdBridgeWrapper) Run() error {
	go func() {
		l.bridge.DetectReconnectMPDClient(l.config.mpdServer, l.mpdPassword)
	}()

	go l.bridge.MainLoop()
	return nil
}

//-----------------------------------------

// func CreateMPDBridge(config Config, mqttClient mqtt.Client) *mpdmqtt.MpdMQTTBridge {

// 	mpdPassword, err := fileToString(config.mpdPasswordFile)
// 	if err != nil {
// 		slog.Error("Error reading mpd password",
// 			"mpdPasswordFile", config.mpdPasswordFile, "error", err)
// 	}
// 	slog.Info("MPD password", "password", mpdPassword)

// 	mpdClient, mpdWatcher, err := mpdmqtt.CreateMPDClient(config.mpdServer, mpdPassword)
// 	if err != nil {
// 		slog.Error("Could not create MPD client", "error", err, "mpdserver", config.mpdServer, "mpdpassword", mpdPassword)
// 	}

// 	bridge := mpdmqtt.NewMpdMQTTBridge(mpdClient, mpdWatcher, mqttClient)

// 	go func() {
// 		bridge.DetectReconnectMPDClient(config.mpdServer, mpdPassword)
// 	}()

// 	return bridge
// }

// func initMPDBridge(bridge *mpdmqtt.MpdMQTTBridge) {
// 	go bridge.MainLoop()
// }
