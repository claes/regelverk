package regelverk

import (
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BridgeWrapper interface {
	InitializeBridge(mqttClient mqtt.Client, config Config) error
	Run() error
}

func initBridges(mqttClient mqtt.Client, config Config, bridgeWrappers []BridgeWrapper) {

	for _, bridgeWrapper := range bridgeWrappers {
		err := bridgeWrapper.InitializeBridge(mqttClient, config)
		if err != nil {
			slog.Error("Could not initialize bridge", "error", err)
		} else {
			err = bridgeWrapper.Run()
			if err != nil {
				slog.Error("Error when running bridge", "error", err)
			}
		}
	}
}

// func initBridgesNew(mqttClient mqtt.Client, config Config) {
// 	bridgeWrappers := []BridgeWrapper{
// 		&cecBridgeWrapper{},
// 		&mpdBridgeWrapper{},
// 		&pulseaudioBridgeWrapper{},
// 		&rotelBridgeWrapper{},
// 		&samsungBridgeWrapper{},
// 	}
// 	initBridges(mqttClient, config, bridgeWrappers)
// }

// func initBridgesOld(mqttClient mqtt.Client, config Config) {

// 	// Unify bridges under a common interface
// 	// What do they need - just a main loop type of function?
// 	// Or should I have a common Regelverk type that abstracts the bridges?
// 	// Perhaps takes a config + mqtt client?

// 	rotelBridge, err := CreateRotelBridge(config.rotelSerialPort, mqttClient)
// 	if err != nil {
// 		slog.Error("Could not create rotel bridge", "error", err)
// 	} else {
// 		initRotelBridge(rotelBridge)
// 	}

// 	pulseBridge, err := CreatePulseaudioBridge(config.pulseserver, mqttClient)
// 	if err != nil {
// 		slog.Error("Could not create pulseaudio bridge", "error", err)
// 	} else {
// 		initPulseaudioBridge(pulseBridge)
// 	}

// 	initCECBridge(mqttClient)
// 	initSamsungBridge(CreateSamsungBridge(config.samsungTvAddress, mqttClient))
// 	initMPDBridge(CreateMPDBridge(config, mqttClient))
// }
