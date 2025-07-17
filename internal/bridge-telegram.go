package regelverk

import (
	"context"
	"log/slog"

	telegrammqtt "github.com/claes/mqtt-bridges/telegram-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TelegramBridgeWrapper struct {
	bridge *telegrammqtt.TelegramMQTTBridge
}

func (l *TelegramBridgeWrapper) String() string {
	return "TelegramBridgeWrapper"
}

func (l *TelegramBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error

	telegramToken, err := fileToString(config.TelegramTokenFile)
	if err != nil {
		slog.Error("Error reading telegram token",
			"telegramTokenFile", config.TelegramTokenFile, "error", err)
	}

	telegramConfig := telegrammqtt.TelegramConfig{
		BotToken: telegramToken,
		ChatNamesToIds: map[string]int64{
			"regelverkgeneral": -4915582934,
		},
	}

	l.bridge, err = telegrammqtt.NewTelegramMQTTBridge(telegramConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *TelegramBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting Telegram bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
