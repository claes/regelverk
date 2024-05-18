package main

import (
	"log/slog"
	"time"

	"github.com/bendahl/uinput"
	"github.com/claes/cec"
	cecmqtt "github.com/claes/cec-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func CreateCECBridge(mqttClient mqtt.Client) *cecmqtt.CecMQTTBridge {
	bridge := cecmqtt.NewCecMQTTBridge(cecmqtt.
		CreateCECConnection("/dev/ttyACM0", "Regelverk"),
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

	bridge.CECConnection.KeyPresses = make(chan *cec.KeyPress, 20) // Buffered channel
	for keyPress := range bridge.CECConnection.KeyPresses {
		slog.Debug("Key press", "keyCode", keyPress.KeyCode, "duration", keyPress.Duration)
		if keyPress.Duration == 0 {
			keycode := -1
			switch keyPress.KeyCode {
			case 0:
				keycode = uinput.KeyEnter
			case 1:
				keycode = uinput.KeyUp
			case 2:
				keycode = uinput.KeyDown
			case 3:
				keycode = uinput.KeyLeft
			case 4:
				keycode = uinput.KeyRight
			case 48:
				keycode = uinput.KeyPageup
			case 49:
				keycode = uinput.KeyPagedown
			case 145:
				// Menu key,
				// see https://stackoverflow.com/questions/32815986/which-key-macro-in-linux-input-h-matches-the-menu-key
				keycode = uinput.KeyCompose
				//keycode = uinput.KeyProps
				//keycode = uinput.KeyC // contextual menu / playlist
				//keycode = uinput.KeyMenu
			case 13:
				keycode = uinput.KeyBackspace
			case 32: //0
				keycode = uinput.Key0
			case 33:
				keycode = uinput.Key1
			case 34:
				keycode = uinput.Key2
			case 35:
				keycode = uinput.Key3
			case 36:
				keycode = uinput.Key4
			case 37:
				keycode = uinput.Key5
			case 38:
				keycode = uinput.Key6
			case 39:
				keycode = uinput.Key7
			case 40:
				keycode = uinput.Key8
			case 41:
				keycode = uinput.Key9
			case 113: //Blue
				keycode = uinput.KeyEnter
			case 114: //Red
				keycode = uinput.KeyL // Kodi next subtitle
			case 115: //Green
				keycode = uinput.KeyTab // Kodi fullscreen
			case 116: //Yellow
				keycode = uinput.KeyEnter
			case 83: //Guide
				keycode = uinput.KeyEnter
			case 68:
				keycode = uinput.KeyPlay
			case 69:
				keycode = uinput.KeyStop
			case 70:
				keycode = uinput.KeySpace
				//keycode = uinput.KeyPause
			case 72:
				keycode = uinput.KeyVideoPrev
				//keycode = uinput.KeyRewind
			case 73:
				keycode = uinput.KeyVideoNext
				//keycode = uinput.KeyFastforward
			}

			if keycode >= 0 {
				keyboard.KeyPress(keycode)
			}
		}
	}
}

func cecBridgeMainLoop(bridge *cecmqtt.CecMQTTBridge) {
	for {
		time.Sleep(10 * time.Second)
		bridge.CECConnection.Transmit("10:8F") //"Recording 1" asks TV for power status
	}
}

func initCECBridge(bridge *cecmqtt.CecMQTTBridge) {

	go bridge.PublishCommands()
	//go bridge.PublishKeyPresses()
	//go bridge.PublishSourceActivations()
	//go bridge.PublishMessages(true)
	go bridgeKeyPresses(bridge)
	go cecBridgeMainLoop(bridge)
}
