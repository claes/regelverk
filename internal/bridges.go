package regelverk

import (
	"context"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BridgeWrapper interface {
	InitializeBridge(mqttClient mqtt.Client, config Config) error
	Run(context context.Context) error
}

func initBridges(context context.Context, mqttClient mqtt.Client, config Config, bridgeWrappers *[]BridgeWrapper) {

	for _, bridgeWrapper := range *bridgeWrappers {
		slog.Debug("Initializing bridge", "bridgeWrapper", bridgeWrapper)
		err := bridgeWrapper.InitializeBridge(mqttClient, config)
		if err != nil {
			slog.Error("Could not initialize bridge", "error", err, "bridgeWrapper", bridgeWrapper)
		} else {
			slog.Debug("Starting bridge", "bridgeWrapper", bridgeWrapper)
			err = bridgeWrapper.Run(context)
			if err != nil {
				slog.Error("Error when starting bridge", "error", err, "bridgeWrapper", bridgeWrapper)
			} else {
				slog.Debug("Started bridge", "bridgeWrapper", bridgeWrapper)
			}
		}
	}
}
