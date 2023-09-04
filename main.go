package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var debug *bool
var dryRun *bool

// Mostly reused from https://github.com/stapelberg/regelwerk

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

	Init()

	ProcessEvent(MQTTEvent) []MQTTPublish
}

type statusLoop struct {
	mu sync.Mutex
}

func (l *statusLoop) Lock() { l.mu.Lock() }

func (l *statusLoop) Unlock() { l.mu.Unlock() }

type mqttMessageHandler struct {
	dryRun bool
	client mqtt.Client
	loops  []controlLoop
}

func (h *mqttMessageHandler) handle(_ mqtt.Client, m mqtt.Message) {
	//log.Printf("MQTT: %s %s", m.Topic(), m.Payload())
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

func regelverk(broker string) error {

	// Enable file names in logs:
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	host, err := os.Hostname()
	if err != nil {
		return err
	}

	mqttMessageHandler := &mqttMessageHandler{
		dryRun: *dryRun,
		loops:  loops,
	}

	for _, l := range loops {
		l.Init()
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("regelverk-" + host).
		SetOnConnectHandler(func(client mqtt.Client) {
			// TODO: add MQTTTopics() []string to controlLoop interface and
			// subscribe to the union of topics, with the same handler that feeds to the source control loops
			const topic = "#"
			token := client.Subscribe(
				topic,
				1, /* minimal QoS level zero: at most once, best-effort delivery */
				mqttMessageHandler.handle)
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
			fmt.Printf("Subscribed to %q\n", topic)
		}).
		SetConnectRetry(true)

	client := mqtt.NewClient(opts)
	mqttMessageHandler.client = client
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// This can indeed fail, e.g. if the broker DNS is not resolvable.
		return fmt.Errorf("MQTT connection failed: %v", token.Error())
	} else if *debug {
		fmt.Printf("Connected to MQTT broker: %s\n", broker)
	}

	fmt.Printf("MQTT subscription established\n")
	for tick := range time.Tick(1 * time.Second) {
		ev := MQTTEvent{
			Timestamp: tick,
			Topic:     "regelverk/ticker/1s",
			Payload:   nil,
		}
		mqttMessageHandler.handleEvent(ev)
	}

	select {} // loop forever
}

func printHelp() {
	fmt.Println("Usage: regelverk [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {

	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	dryRun = flag.Bool("dry_run", false, "Dry run (do not publish)")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	fmt.Printf("Started\n")

	go func() {
		if err := regelverk(*mqttBroker); err != nil {
			log.Fatal(err)
		}
	}()

	//inithttp()

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-c
	fmt.Printf("Shut down\n")
	os.Exit(0)

}
