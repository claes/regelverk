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
		CreateCECConnection("tcp://localhost:1883", "Regelverk"),
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
			keycode := -1
			switch keyPress.KeyCode {
			case 0x01:
				keycode = uinput.KeyUp
			case 0x02:
				keycode = uinput.KeyDown
			case 0x03:
				keycode = uinput.KeyLeft
			case 0x04:
				keycode = uinput.KeyRight
			case 0x2B:
				keycode = uinput.KeyEnter
			case 0x41:
				keycode = uinput.KeyVolumeup
			case 0x42:
				keycode = uinput.KeyVolumedown
			case 0x43:
				keycode = uinput.KeyMute
			case 0x44:
				keycode = uinput.KeyPlay
			case 0x45:
				keycode = uinput.KeyStop
			case 0x46:
				keycode = uinput.KeyPause
			case 0x47:
				keycode = uinput.KeyRecord
			case 0x48:
				keycode = uinput.KeyRewind
			case 0x49:
				keycode = uinput.KeyFastforward
			case 0x71: //Blue
				keycode = uinput.KeyFastforward
			case 0x72: //Red
				keycode = uinput.KeyFastforward
			case 0x73: //Green
				keycode = uinput.KeyFastforward
			case 0x74: //Yellow
				keycode = uinput.KeyFastforward
			case 0x20: //0
				keycode = uinput.KeyFastforward
			case 0x21:
				keycode = uinput.KeyFastforward
			case 0x22:
				keycode = uinput.KeyFastforward
			case 0x23:
				keycode = uinput.KeyFastforward
			case 0x24:
				keycode = uinput.KeyFastforward
			case 0x25:
				keycode = uinput.KeyFastforward
			case 0x26:
				keycode = uinput.KeyFastforward
			case 0x27:
				keycode = uinput.KeyFastforward
			case 0x28:
				keycode = uinput.KeyFastforward
			case 0x29:
				keycode = uinput.KeyFastforward

			}

			/*
				0x00: "Select", 0x01: "Up", 0x02: "Down", 0x03: "Left",
					0x04: "Right", 0x05: "RightUp", 0x06: "RightDown", 0x07: "LeftUp",
					0x08: "LeftDown", 0x09: "RootMenu", 0x0A: "SetupMenu", 0x0B: "ContentsMenu",
					0x0C: "FavoriteMenu", 0x0D: "Exit", 0x20: "0", 0x21: "1", 0x22: "2", 0x23: "3",
					0x24: "4", 0x25: "5", 0x26: "6", 0x27: "7", 0x28: "8", 0x29: "9", 0x2A: "Dot",
					0x2B: "Enter", 0x2C: "Clear", 0x2F: "NextFavorite", 0x30: "ChannelUp",
					0x31: "ChannelDown", 0x32: "PreviousChannel", 0x33: "SoundSelect",
					0x34: "InputSelect", 0x35: "DisplayInformation", 0x36: "Help",
					0x37: "PageUp", 0x38: "PageDown", 0x40: "Power", 0x41: "VolumeUp",
					0x42: "VolumeDown", 0x43: "Mute", 0x44: "Play", 0x45: "Stop", 0x46: "Pause",
					0x47: "Record", 0x48: "Rewind", 0x49: "FastForward", 0x4A: "Eject",
					0x4B: "Forward", 0x4C: "Backward", 0x4D: "StopRecord", 0x4E: "PauseRecord",
					0x50: "Angle", 0x51: "SubPicture", 0x52: "VideoOnDemand",
					0x53: "ElectronicProgramGuide", 0x54: "TimerProgramming",
					0x55: "InitialConfiguration", 0x60: "PlayFunction", 0x61: "PausePlay",
					0x62: "RecordFunction", 0x63: "PauseRecordFunction",
					0x64: "StopFunction", 0x65: "Mute",
					0x66: "RestoreVolume", 0x67: "Tune", 0x68: "SelectMedia",
					0x69: "SelectAvInput", 0x6A: "SelectAudioInput", 0x6B: "PowerToggle",
					0x6C: "PowerOff", 0x6D: "PowerOn", 0x71: "Blue", 0x72: "Red", 0x73: "Green",
					0x74: "Yellow", 0x75: "F5", 0x76: "Data", 0x91: "AnReturn",
					0x96: "Max"
			*/
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
