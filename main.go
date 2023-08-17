package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

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
}

type controlLoop interface {
	sync.Locker

	StatusString() string

	ProcessEvent(MQTTEvent) []MQTTPublish
}

type statusLoop struct {
	mu sync.Mutex

	statusMu   sync.Mutex
	status     string
	statusPrev string
}

func (l *statusLoop) Lock()   { l.mu.Lock() }
func (l *statusLoop) Unlock() { l.mu.Unlock() }

func (l *statusLoop) StatusString() string {
	l.statusMu.Lock()
	defer l.statusMu.Unlock()
	return l.status
}

func (l *statusLoop) statusf(format string, v ...interface{}) {
	l.statusMu.Lock()
	defer l.statusMu.Unlock()
	l.status = fmt.Sprintf(format, v...)
	if l.status != l.statusPrev {
		log.Output(2, fmt.Sprintf("status: %s", l.status))
		l.statusPrev = l.status
	}
}

type invocationLog struct {
	Time    time.Time
	Loop    controlLoop
	Event   MQTTEvent
	Results []MQTTPublish
}

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

func (h *mqttMessageHandler) handleEvent(ev MQTTEvent) {
	for _, l := range h.loops {
		l := l // copy
		go func() {
			// For reliability, we call each loop in its own goroutine (yes, one
			// per message), so that one loop can be stuck while others still
			// make progress.
			l.Lock()
			results := l.ProcessEvent(ev)
			l.Unlock()
			if len(results) == 0 {
				return
			}
			for _, r := range results {
				//log.Printf("publishing: %+v", r)
				if !h.dryRun {
					h.client.Publish(r.Topic, r.Qos, r.Retained, r.Payload)
				}
			}
		}()
	}
}

func regelverk() error {
	dryRun := flag.Bool("dry_run",
		false,
		"dry run (do not publish)")
	flag.Parse()

	// Enable file names in logs:
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mux := http.NewServeMux()

	host, err := os.Hostname()
	if err != nil {
		return err
	}

	mqttMessageHandler := &mqttMessageHandler{
		dryRun: *dryRun,
		loops:  loops,
	}

	opts := mqtt.NewClientOptions().
		AddBroker("tcp://192.168.88.174:1883").
		SetClientID("regelverk-" + host).
		SetOnConnectHandler(func(client mqtt.Client) {
			// TODO: add MQTTTopics() []string to controlLoop interface and
			// subscribe to the union of topics, with the same handler that feeds to the source control loops
			const topic = "#"
			token := client.Subscribe(
				topic,
				0, /* minimal QoS level zero: at most once, best-effort delivery */
				mqttMessageHandler.handle)
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
			log.Printf("subscribed to %q", topic)
		}).
		SetConnectRetry(true)

	go func() {
		if err := http.ListenAndServe(":37731", mux); err != nil {
			log.Fatal(err)
		}
	}()

	client := mqtt.NewClient(opts)
	mqttMessageHandler.client = client
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// This can indeed fail, e.g. if the broker DNS is not resolvable.
		return fmt.Errorf("MQTT connection failed: %v", token.Error())
	}
	log.Printf("MQTT subscription established")
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

func main() {
	if err := regelverk(); err != nil {
		log.Fatal(err)
	}
}
