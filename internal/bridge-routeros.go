package regelverk

import (
	"context"
	"log/slog"

	routerosmqtt "github.com/claes/routeros-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RouterOSBridgeWrapper struct {
	bridge *routerosmqtt.RouterOSMQTTBridge
}

func (l *RouterOSBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {

	slog.Debug("Creating RouterOS bridge")

	routerPassword, err := fileToString(config.RouterPasswordFile)
	if err != nil {
		slog.Error("Error reading router password",
			"routerPasswordFile", config.RouterPasswordFile, "error", err)
	}

	routerOSClientConfig :=
		routerosmqtt.RouterOSClientConfig{RouterAddress: config.RouterAddress,
			Username: config.RouterUsername, Password: routerPassword}

	bridge, err :=
		routerosmqtt.NewRouterOSMQTTBridge(routerOSClientConfig, mqttClient, config.MQTTTopicPrefix)
	if err != nil {
		slog.Error("Could not create RouterOS bridge", "error", err)
		return err
	}
	l.bridge = bridge
	slog.Debug("Initialized RouterOS bridge", "bridge", l.bridge, "mqttClient", mqttClient)
	return nil
}

func (l *RouterOSBridgeWrapper) Run(context context.Context) error {
	slog.Debug("Starting RouterOS bridge", "bridge", l.bridge)
	go l.bridge.MainLoop()
	slog.Debug("RouterOS bridge started")
	return nil
}
