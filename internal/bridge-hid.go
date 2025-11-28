package regelverk

import (
	"context"
	"log/slog"
	"strconv"

	hidmqtt "github.com/claes/mqtt-bridges/hid-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type HidBridgeWrapper struct {
	bridge *hidmqtt.HIDMQTTBridge
}

func (l *HidBridgeWrapper) String() string {
	return "HIDBridgeWrapper"
}

func (l *HidBridgeWrapper) InitializeBridge(mqttClient mqtt.Client, config Config) error {
	var err error

	vendorID, err := strconv.ParseUint(config.HIDVendorID, 16, 16)
	if err != nil {
		slog.Error("Error converting Vendor ID. Cannot initialize HID bridge", "error", err, "vendorID", config.HIDVendorID)
		return err
	}

	productID, err := strconv.ParseUint(config.HIDProductID, 16, 16)
	if err != nil {
		slog.Error("Error converting Product ID. Cannot initialize HID bridge", "error", err, "productID", config.HIDProductID)
		return err
	}

	hidConfig := hidmqtt.HIDBridgeConfig{VendorID: uint16(vendorID), ProductID: uint16(productID),
		PublishBytes: true, PublishNative: false, PublishReadable: true}
	l.bridge, err = hidmqtt.NewHIDMQTTBridge(hidConfig, mqttClient, config.MQTTTopicPrefix)
	return err
}

func (l *HidBridgeWrapper) Run(ctx context.Context) error {
	slog.Debug("Starting HID  bridge", "bridge", l.bridge)
	l.bridge.EventLoop(ctx)
	return nil
}
