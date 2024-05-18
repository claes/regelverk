package main

import (
	"log/slog"

	"github.com/bendahl/uinput"
	"github.com/claes/cec"
	cecmqtt "github.com/claes/cec-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateCECBridge(mqttClient mqtt.Client) *cecmqtt.CecMQTTBridge {
	bridge := cecmqtt.NewCecMQTTBridge(cecmqtt.
		CreateCECConnection("Regelverk", "tcp://localhost:1883"),
		mqttClient)
	return bridge
}

func bridgeKeyPresses(bridge *cecmqtt.CecMQTTBridge) {
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("regelverk"))
	if err != nil {
		slog.Error("Could not initialize uinput", "error", err)
		return
	}
	defer keyboard.Close()

	bridge.CECConnection.KeyPresses = make(chan *cec.KeyPress, 10) // Buffered channel
	for keyPress := range bridge.CECConnection.KeyPresses {
		slog.Debug("Key press", "keyCode", keyPress.KeyCode, "duration", keyPress.Duration)
		if keyPress.Duration == 0 {
			keyboard.KeyPress(uinput.KeyA)
		}
	}
}

func initCECBridge(bridge *cecmqtt.CecMQTTBridge) {

	go bridge.PublishCommands()
	//go bridge.PublishKeyPresses()
	//go bridge.PublishSourceActivations()
	//go bridge.PublishMessages(true)
	go bridgeKeyPresses(bridge)
}

func initBridges(mqttClient mqtt.Client) {
	initCECBridge(CreateCECBridge(mqttClient))
}
