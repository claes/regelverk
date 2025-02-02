package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Inspired by https://github.com/stapelberg/regelwerk

type Config struct {
	BluetoothAddress   string
	MetricsAddress     string
	MetricsRealm       string
	MpdPasswordFile    string
	MpdServer          string
	MQTTBroker         string
	MQTTPasswordFile   string
	MQTTTopicPrefix    string
	MQTTUserName       string
	Pulseserver        string
	RotelSerialPort    string
	RouterAddress      string
	RouterPasswordFile string
	RouterUsername     string
	SamsungTvAddress   string
	SnapcastServer     string
	WebAddress         string
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

type ControlLoop interface {
	sync.Locker

	Init(*MQTTMessageHandler, Config)

	ProcessEvent(MQTTEvent) []MQTTPublish
}

type statusLoop struct {
	mu sync.Mutex
	m  *MQTTMessageHandler
}

func (l *statusLoop) Lock() { l.mu.Lock() }

func (l *statusLoop) Unlock() { l.mu.Unlock() }

type MQTTMessageHandler struct {
	dryRun           bool
	client           mqtt.Client
	loops            []ControlLoop
	masterController *MasterController
}

func (h *MQTTMessageHandler) handle(_ mqtt.Client, m mqtt.Message) {
	slog.Debug("MQTT handle", "topic", m.Topic(), "payload", m.Payload())
	ev := MQTTEvent{
		Timestamp: time.Now(), // consistent for all loops
		Topic:     m.Topic(),
		Payload:   m.Payload(),
	}
	h.handleEvent(ev)
}

var count int64 = 0

func (h *MQTTMessageHandler) handleEvent(ev MQTTEvent) {

	h.masterController.ProcessEvent(h.client, ev)

	for _, l := range h.loops {
		loop := l // copy
		go func() {
			// For reliability, we call each loop in its own goroutine (yes, one
			// per message), so that one loop can be stuck while others still
			// make progress.
			loop.Lock()
			results := loop.ProcessEvent(ev)
			loop.Unlock()
			if len(results) == 0 {
				return
			}
			for _, result := range results {
				count = count + 1
				if !h.dryRun {
					go func(toPublish MQTTPublish) {
						if toPublish.Wait != 0 {
							time.Sleep(toPublish.Wait)
						}
						h.client.Publish(toPublish.Topic, toPublish.Qos, toPublish.Retained, toPublish.Payload)
					}(result)
				}
			}
		}()
	}
}

func createMQTTMessageHandler(config Config, loops []ControlLoop, masterController *MasterController, dryRun, debug *bool) (*MQTTMessageHandler, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
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

	mqttMessageHandler := &MQTTMessageHandler{
		dryRun:           *dryRun,
		loops:            loops,
		masterController: masterController,
	}

	opts := mqtt.NewClientOptions().
		AddBroker(config.MQTTBroker).
		SetUsername(config.MQTTUserName).
		SetPassword(mqttPassword).
		SetClientID("regelverk-" + host).
		SetOnConnectHandler(func(client mqtt.Client) {
			// TODO: add MQTTTopics() []string to controlLoop interface and
			// subscribe to the union of topics, with the same handler that
			// feeds to the source control loops
			const topic = "#"
			token := client.Subscribe(
				topic,
				1, /* minimal QoS level zero: at most once, best-effort delivery */
				mqttMessageHandler.handle)
			if token.Wait() && token.Error() != nil {
				slog.Error("Error creating MQTT client", "error", token.Error())
				os.Exit(1)
			}
			slog.Info("Subscribed to topic", "topic", topic)
		}).
		SetConnectRetry(true)

	client := mqtt.NewClient(opts)
	slog.Info("Connecting to MQTT broker", "broker", config.MQTTBroker)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// This can indeed fail, e.g. if the broker DNS is not resolvable.
		return nil, fmt.Errorf("MQTT connection failed: %v", token.Error())
	}
	slog.Info("Connected to MQTT broker", "broker", config.MQTTBroker)

	mqttMessageHandler.client = client
	return mqttMessageHandler, nil
}

// func createWebServer(config Config) {
// 	go func() {
// 		slog.Info("Initializing HTTP server", "address", config.webAddress)

// 		err := http.ListenAndServe(config.webAddress, nil)

// 		if err != nil {
// 			slog.Error("Error initializing HTTP server",
// 				"listenAddr", config.webAddress, "error", err)
// 			os.Exit(1)
// 		}
// 	}()
// }

func runRegelverk(ctx context.Context, config Config,
	loops []ControlLoop, bridgeWrappers *[]BridgeWrapper, controllers *[]Controller,
	dryRun, debug *bool) error {

	masterController := CreateMasterController()

	if len(config.MetricsAddress) > 0 {
		slog.Info("Initialzing metrics collection", "address", config.MetricsAddress)
		masterController.metricsConfig = MetricsConfig{MetricsAddress: config.MetricsAddress, MetricsRealm: config.MetricsRealm}
		metrics.InitPush("http://"+config.MetricsAddress+"/api/v1/import/prometheus", 10*time.Second, "", true)
	} else {
		slog.Info("Metrics collection not initialized")
	}

	masterController.Init()
	masterController.controllers = controllers

	mqttMessageHandler, err := createMQTTMessageHandler(config, loops, &masterController, dryRun, debug)
	if err != nil {
		return err
	}

	slog.Info("Initializing bridges")
	initBridges(ctx, mqttMessageHandler.client, config, bridgeWrappers)

	slog.Info("Initializing loops")
	for _, l := range loops {
		l.Init(mqttMessageHandler, config)
	}

	// Init web after handlers are established
	// createWebServer(config)

	go func() {
		for tick := range time.Tick(1 * time.Minute) {
			timeOfDay := ComputeTimeOfDay(time.Now(), 59, 18)
			ev := MQTTEvent{
				Timestamp: tick,
				Topic:     "regelverk/ticker/timeofday",
				Payload:   timeOfDay,
			}
			mqttMessageHandler.handleEvent(ev)
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
		slog.Error("Error parsing JSON payload", "payload", ev.Payload)
		return nil
	}
	m := payload.(map[string]interface{})
	return m
}
