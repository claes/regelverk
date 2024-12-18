package regelverk

import (
	"context"
	"log/slog"

	routerosmqtt "github.com/claes/mqtt-bridges/routeros-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RouterOSBridgeWrapper struct {
	bridge *routerosmqtt.RouterOSMQTTBridge
}

func (l *RouterOSBridgeWrapper) String() string {
	return "RouterOSBridgeWrapper"
}

func (l *RouterOSBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {

	routerPassword, err := fileToString(config.RouterPasswordFile)
	if err != nil {
		slog.Error("Error reading router password",
			"routerPasswordFile", config.RouterPasswordFile, "error", err)
		return err
	}
	routerOSClientConfig :=
		routerosmqtt.RouterOSClientConfig{RouterAddress: config.RouterAddress,
			Username: config.RouterUsername, Password: routerPassword}
	l.bridge, err =
		routerosmqtt.NewRouterOSMQTTBridge(routerOSClientConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *RouterOSBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting RouterOS bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
