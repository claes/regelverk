package regelverk

import (
	"context"
	"log/slog"
	"time"

	"github.com/bendahl/uinput"
	"github.com/claes/cec"
	cecmqtt "github.com/claes/cec-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type CecBridgeWrapper struct {
	mqttClient  mqtt.Client
	topicPrefix string
}

func (l *CecBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	l.mqttClient = mqttClient
	l.topicPrefix = config.MQTTTopicPrefix
	return nil
}

func (l *CecBridgeWrapper) Run() error {
	go cecBridgeMainLoop(l.mqttClient, l.topicPrefix)
	return nil
}

// func bridgeMessages(bridge *cecmqtt.CecMQTTBridge) {
// 	pattern := `^(>>|<<)\s([0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2})*)`
// 	regex, err := regexp.Compile(pattern)
// 	if err != nil {
// 		slog.Info("Error compiling regex", "error", err)
// 		return
// 	}

// 	bridge.CECConnection.Messages = make(chan string, 20) // Buffered channel
// 	for message := range bridge.CECConnection.Messages {
// 		slog.Info("CEC Message", "message", message)
// 		matches := regex.FindStringSubmatch(message)
// 		if matches != nil {
// 			prefix := matches[1]
// 			hexPart := matches[2]
// 			slog.Info("CEC Message payload match", "prefix", prefix, "hex", hexPart)
// 			if prefix == "<<" {
// 				bridge.PublishMQTT("cec/msg/rx", hexPart, true)
// 			} else if prefix == ">>" {
// 				bridge.PublishMQTT("cec/msg/tx", hexPart, true)
// 			}
// 		}
// 	}
// }

func bridgeKeyPresses(ctx context.Context, bridge *cecmqtt.CecMQTTBridge, keyboard uinput.Keyboard) {

	bridge.CECConnection.KeyPresses = make(chan *cec.KeyPress, 20) // Buffered channel

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Bridge keypresses function is being cancelled")
			return
		case keyPress := <-bridge.CECConnection.KeyPresses:
			translatePerformKeypress(keyPress, keyboard)
		}
	}
}

func translatePerformKeypress(keyPress *cec.KeyPress, keyboard uinput.Keyboard) {
	slog.Debug("Key press", "keyCode", keyPress.KeyCode, "duration", keyPress.Duration)
	if keyPress.Duration == 0 ||
		(keyPress.Duration == 500 && keyPress.KeyCode == 145) { //strange workaround
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
			keycode = uinput.KeyCompose
			//keycode = uinput.KeyC // contextual menu / playlist
			// Menu key,
			// see https://stackoverflow.com/questions/32815986/which-key-macro-in-linux-input-h-matches-the-menu-key
			//keycode = uinput.KeyProps
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
			slog.Debug("Send keypress", "keycode", keycode)
			keyboard.KeyPress(keycode)
		}
	}
}

// func initCECBridge(mqttClient mqtt.Client) {
// 	go cecBridgeMainLoop(mqttClient)
// }

func cecBridgeMainLoop(mqttClient mqtt.Client, topicPrefix string) {

	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("regelverk"))
	if err != nil {
		slog.Error("Could not initialize uinput", "error", err)
		return
	}
	defer keyboard.Close()

	i := 0

	for {
		time.Sleep(4 * time.Second)
		slog.Info("Creating new CEC connection", "count", i)
		cecConnection := cecmqtt.CreateCECConnection("/dev/ttyACM0", "Regelverk")
		bridge := cecmqtt.NewCecMQTTBridge(cecConnection, mqttClient, topicPrefix)

		ctx, cancel := context.WithCancel(context.Background())

		go bridge.PublishCommands(ctx)
		go bridge.PublishMessages(ctx, true)
		go bridge.PublishSourceActivations(ctx)
		go bridgeKeyPresses(ctx, bridge, keyboard)

		if i == 0 {
			slog.Info("CEC bridge started")
		}

		for {
			bridge.CECConnection.Transmit("10:8F") //"Recording 1" asks TV for power status
			//bridge.CECConnection.Transmit("1F:85") //"Recording 1" asks TV for active source
			time.Sleep(10 * time.Second)

			ping := bridge.CECConnection.Ping()
			slog.Debug("Ping CEC", "result", ping, "count", i)
			if ping == 0 {
				slog.Error("CEC ping not succcessful, resetting connection", "result", ping, "count", i)
				cancel()
				slog.Error("Destroying CEC connection")
				//cecConnection.Destroy()
				cecConnection.Close() //in case destroy does not work
				slog.Error("Will attempt to recreate connection")
				i++
				break
			}
		}
	}
}
