package main

import (
	"log/slog"

	mpdmqtt "github.com/claes/mpd-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateMPDBridge(mpdserver, mpdpassword string, mqttClient mqtt.Client) *mpdmqtt.MpdMQTTBridge {
	mpdClient, mpdWatcher, err := mpdmqtt.CreateMPDClient(mpdserver, mpdpassword)
	if err != nil {
		slog.Error("Could not create MPD client", "error", err, "mpdserver", mpdserver, "mpdpassword", mpdpassword)
	}

	bridge := mpdmqtt.NewMpdMQTTBridge(mpdClient, mpdWatcher, mqttClient)

	go func() {
		bridge.DetectReconnectMPDClient(mpdserver, mpdpassword)
	}()

	return bridge
}

func initMPDBridge(bridge *mpdmqtt.MpdMQTTBridge) {
	go bridge.MainLoop()
}
