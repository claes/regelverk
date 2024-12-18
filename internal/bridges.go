package regelverk

import (
	"context"
	"fmt"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type BridgeWrapper interface {
	fmt.Stringer
	InitializeBridge(mqttClient mqtt.Client, config Config) error
	Run(context context.Context) error
}

func initBridges(ctx context.Context, mqttClient mqtt.Client, config Config, bridgeWrappers *[]BridgeWrapper) {

	for _, bridgeWrapper := range *bridgeWrappers {
		go func(ctx context.Context, bridgeWrapper BridgeWrapper) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Recovered from panic", "recover", r, "bridgeWrapper", bridgeWrapper)
				}
			}()

			bridgeCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			slog.Info("Initializing bridge", "bridgeWrapper", bridgeWrapper)
			err := bridgeWrapper.InitializeBridge(mqttClient, config)
			if err != nil {
				slog.Error("Could not initialize bridge", "error", err, "bridgeWrapper", bridgeWrapper)
				return
			}

			slog.Info("Starting bridge", "bridgeWrapper", bridgeWrapper)
			err = bridgeWrapper.Run(bridgeCtx)
			if err != nil {
				slog.Error("Error when running bridge", "error", err, "bridgeWrapper", bridgeWrapper)
			} else {
				slog.Info("Bridge exited gracefully", "bridgeWrapper", bridgeWrapper)
			}
		}(ctx, bridgeWrapper)
	}
}
