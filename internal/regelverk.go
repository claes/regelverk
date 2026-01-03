package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metrics"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Inspired by https://github.com/stapelberg/regelwerk

type Config struct {
	BluetoothAddress    string
	CollectMetrics      bool
	CollectDebugMetrics bool
	HIDVendorID         string
	HIDProductID        string
	MetricsAddress      string
	MetricsRealm        string
	MpdPasswordFile     string
	MpdServer           string
	MQTTBroker          string
	MQTTPasswordFile    string
	MQTTTopicPrefix     string
	MQTTUserName        string
	Pulseserver         string
	RotelSerialPort     string
	RouterAddress       string
	RouterPasswordFile  string
	RouterUsername      string
	SamsungTvAddress    string
	SnapcastServer      string
	TelegramTokenFile   string
	WebAddress          string
}

type MQTTEvent struct {
	Timestamp time.Time
	Topic     string
	Payload   interface{}
}

type MQTTPublish struct {
	Topic    string
	Qos      byte
	Retained bool
	Payload  interface{}
	Wait     time.Duration
}

func (masterController *MasterController) handle(_ mqtt.Client, m mqtt.Message) {
	slog.Debug("MQTT handle", "topic", m.Topic(), "payload", m.Payload())
	ev := MQTTEvent{
		Timestamp: time.Now(),
		Topic:     m.Topic(),
		Payload:   m.Payload(),
	}

	masterController.ProcessEvent(masterController.mqttClient, ev)

	if masterController.metricsConfig.CollectDebugMetrics {
		counter := metrics.GetOrCreateCounter(fmt.Sprintf(`regelverk_mqtt_handled{topic="%s",realm="%s"}`,
			m.Topic(), masterController.metricsConfig.MetricsRealm))
		counter.Inc()
	}
}

func setupMQTTClient(config Config, masterController *MasterController) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	mqttPassword := ""
	if len(config.MQTTPasswordFile) > 0 {
		mqttPassword, err := fileToString(config.MQTTPasswordFile)
		if err != nil {
			slog.Error("Error reading MQTT password",
				"mqttPasswordFile", config.MQTTPasswordFile, "error", err)
		}
		slog.Debug("MQTT password", "password", mqttPassword)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(config.MQTTBroker).
		SetUsername(config.MQTTUserName).
		SetPassword(mqttPassword).
		SetClientID("regelverk-" + host).
		SetOnConnectHandler(func(client mqtt.Client) {
			topic := "#"
			token := client.Subscribe(topic, 1, masterController.handle)
			if token.Wait() && token.Error() != nil {
				slog.Error("Error subscribing to MQTT topic", "error", token.Error(), "topic", topic)
			}
			topic = "zigbee2mqtt/bridge/devices"
			token = client.Subscribe(topic, 1, masterController.initZ2MDevices)
			if token.Wait() && token.Error() != nil {
				slog.Error("Error subscribing to MQTT topic", "error", token.Error(), "topic", topic)
			}
		}).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second)

	client := mqtt.NewClient(opts)
	slog.Info("Connecting to MQTT broker", "broker", config.MQTTBroker)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// This can indeed fail, e.g. if the broker DNS is not resolvable.
		return fmt.Errorf("MQTT connection failed: %v", token.Error())
	}
	slog.Info("Connected to MQTT broker", "broker", config.MQTTBroker)

	masterController.mqttClient = client
	return nil
}

func runRegelverk(ctx context.Context, config Config, bridgeWrappers *[]BridgeWrapper, controllers *[]Controller) error {

	metricsConfig := MetricsConfig{CollectMetrics: config.CollectMetrics, CollectDebugMetrics: config.CollectDebugMetrics,
		MetricsAddress: config.MetricsAddress, MetricsRealm: config.MetricsRealm}
	if metricsConfig.CollectMetrics || metricsConfig.CollectDebugMetrics {
		slog.Info("Initialzing metrics collection", "address", config.MetricsAddress)
		metrics.InitPush("http://"+config.MetricsAddress+"/api/v1/import/prometheus", 10*time.Second, "", true)
	} else {
		slog.Info("Metrics collection not initialized")
	}

	masterController := CreateMasterController()
	masterController.config = config
	masterController.metricsConfig = metricsConfig
	masterController.Init()
	masterController.controllers = controllers

	err := setupMQTTClient(config, &masterController)
	if err != nil {
		slog.Error("Error initializing MQTT Client", "error", err)
		return err
	}

	slog.Info("Initializing bridges")
	initBridges(ctx, masterController.mqttClient, config, bridgeWrappers)

	go func() {
		for tick := range time.Tick(1 * time.Minute) {
			timeOfDay := ComputeTimeOfDay(time.Now(), 59, 18)
			ev := MQTTEvent{
				Timestamp: tick,
				Topic:     "regelverk/ticker/timeofday",
				Payload:   timeOfDay,
			}
			masterController.ProcessEvent(masterController.mqttClient, ev)
		}
	}()

	slog.Info("Started regelverk")
	<-ctx.Done()
	slog.Info("Finishing regelverk")
	return nil
}

func fileToString(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func parseJSONPayload(ev MQTTEvent) map[string]interface{} {
	var payload interface{}
	payloadJson := string(ev.Payload.([]byte))
	err := json.Unmarshal([]byte(payloadJson), &payload)
	if err != nil {
		slog.Error("Error parsing JSON payload", "payload", ev.Payload, "topic", ev.Topic, "error", err)
		return nil
	}
	m := payload.(map[string]interface{})
	return m
}
