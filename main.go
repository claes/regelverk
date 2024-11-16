package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Mostly reused from https://github.com/stapelberg/regelwerk

var debug *bool
var dryRun *bool

type Config struct {
	broker           string
	webAddress       string
	rotelSerialPort  string
	samsungTvAddress string
	mpdServer        string
	mpdPasswordFile  string
	pulseserver      string
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

type controlLoop interface {
	sync.Locker

	Init(*mqttMessageHandler, Config)

	ProcessEvent(MQTTEvent) []MQTTPublish
}

type statusLoop struct {
	mu sync.Mutex
	m  *mqttMessageHandler
}

func (l *statusLoop) Lock() { l.mu.Lock() }

func (l *statusLoop) Unlock() { l.mu.Unlock() }

type mqttMessageHandler struct {
	dryRun bool
	client mqtt.Client
	loops  []controlLoop
}

func (h *mqttMessageHandler) handle(_ mqtt.Client, m mqtt.Message) {
	slog.Debug("MQTT handle", "topic", m.Topic(), "payload", m.Payload())
	ev := MQTTEvent{
		Timestamp: time.Now(), // consistent for all loops
		Topic:     m.Topic(),
		Payload:   m.Payload(),
	}
	h.handleEvent(ev)
}

var count int64 = 0

func (h *mqttMessageHandler) handleEvent(ev MQTTEvent) {
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

func createMqttMessageHandler(broker string, loops []controlLoop) (*mqttMessageHandler, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	mqttMessageHandler := &mqttMessageHandler{
		dryRun: *dryRun,
		loops:  loops,
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
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
	mqttMessageHandler.client = client
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// This can indeed fail, e.g. if the broker DNS is not resolvable.
		return nil, fmt.Errorf("MQTT connection failed: %v", token.Error())
	} else if *debug {
		slog.Info("Connected to MQTT broker", "broker", broker)
	}
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

func regelverk(config Config, loops []controlLoop, bridgeWrappers []BridgeWrapper) error {

	mqttMessageHandler, err := createMqttMessageHandler(config.broker, loops)
	if err != nil {
		return err
	}

	slog.Info("Initializing bridges")
	initBridges(mqttMessageHandler.client, config, bridgeWrappers)

	slog.Info("Initializing loops")
	for _, l := range loops {
		l.Init(mqttMessageHandler, config)
	}

	// Init web after handlers are established
	// createWebServer(config)

	for tick := range time.Tick(1 * time.Second) {
		ev := MQTTEvent{
			Timestamp: tick,
			Topic:     "regelverk/ticker/1s",
			Payload:   nil,
		}
		mqttMessageHandler.handleEvent(ev)
	}

	slog.Info("Started regelverk")
	select {} // loop forever
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

func printHelp() {
	fmt.Println("Usage: regelverk [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	listenAddr := flag.String("listenAddr", ":8080", "HTTP listen address")
	rotelSerialPort := flag.String("rotelSerialPort", "", "Rotel serial port")
	samsungTVAddress := flag.String("samsungTVAddress", "", "Samsung TV address")
	pulseServer := flag.String("pulseServer", "", "Pulse server")
	mpdServer := flag.String("mpdServer", "", "MPD server")
	mpdPasswordFile := flag.String("mpdPasswordFile", "", "MPD password")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	dryRun = flag.Bool("dry_run", false, "Dry run (do not publish)")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *debug {
		var programLevel = new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)
		handler := slog.NewTextHandler(os.Stderr,
			&slog.HandlerOptions{Level: programLevel})
		slog.SetDefault(slog.New(handler))
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	config := Config{
		broker:           *mqttBroker,
		webAddress:       *listenAddr,
		rotelSerialPort:  *rotelSerialPort,
		samsungTvAddress: *samsungTVAddress,
		mpdServer:        *mpdServer,
		mpdPasswordFile:  *mpdPasswordFile,
		pulseserver:      *pulseServer}

	loops := []controlLoop{
		//&tvLoop{},
		&mpdLoop{},
		&presenceLoop{},
		&kitchenLoop{},
		&cecLoop{},
		&webLoop{},
	}

	bridgeWrappers := []BridgeWrapper{
		&cecBridgeWrapper{},
		&mpdBridgeWrapper{},
		&pulseaudioBridgeWrapper{},
		&rotelBridgeWrapper{},
		&samsungBridgeWrapper{},
	}

	go func() {
		slog.Info("Initializing Regelverk", "config", config)
		err := regelverk(config, loops, bridgeWrappers)
		if err != nil {
			slog.Error("Error initializing regelverk", "error", err)
			os.Exit(1)
		} else {
			slog.Info("Initialized regelverk", "mqttBroker", mqttBroker)
		}
	}()

	slog.Info("Starting regelverk")
	<-c
	slog.Info("Shut down regelverk")
	os.Exit(0)
}
