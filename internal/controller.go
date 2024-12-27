package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	pulseaudiomqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	routerosmqtt "github.com/claes/mqtt-bridges/routeros-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/qmuntal/stateless"
)

type MetricsConfig struct {
	MetricsAddress string
	MetricsRealm   string
}
type MasterController struct {
	stateValueMap StateValueMap
	controllers   *[]Controller
	mu            sync.Mutex
	pushMetrics   bool
	metricsConfig MetricsConfig
}

func CreateMasterController() MasterController {
	return MasterController{stateValueMap: NewStateValueMap()}
}

func (l *MasterController) Init() {
	if len(l.metricsConfig.MetricsAddress) > 0 {
		l.stateValueMap.registerCallback(l.StateValueCallback)
	}
}

func (l *MasterController) StateValueCallback(key string, value, new, updated bool) {
	gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`statevalue{name="%s"}`, key), nil)
	if value {
		gauge.Set(1)
	} else {
		gauge.Set(0)
	}
	if new || updated {
		l.pushMetrics = true
	}
}

type Controller interface {
	sync.Locker

	IsInitialized() bool
	Initialize(sm *MasterController) []MQTTPublish
	ProcessEvent(ev MQTTEvent) []MQTTPublish
}

type BaseController struct {
	name             string
	masterController *MasterController
	stateMachine     *stateless.StateMachine
	eventsToPublish  []MQTTPublish
	isInitialized    bool
	eventHandlers    []func(ev MQTTEvent) []MQTTPublish
	mu               sync.Mutex
}

func (c *BaseController) Lock() {
	c.mu.Lock()
}

func (c *BaseController) Unlock() {
	c.mu.Unlock()
}

func (c *BaseController) SetInitialized() {
	c.isInitialized = true

	firstStateInt, ok := c.stateMachine.MustState().(int)
	if ok {
		gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state{controller="%s"}`, c.name), nil)
		gauge.Set(float64(firstStateInt))
		c.masterController.pushMetrics = true
	}
}

func (c *BaseController) IsInitialized() bool {
	return c.isInitialized
}

func (c *BaseController) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	slog.Debug("Process event", "name", c.name)

	// In case special handling is needed that is not part of base processing
	// Under normal circumstances, state machine should be able to handle most
	for _, eventHandler := range c.eventHandlers {
		c.addEventsToPublish(eventHandler(ev))
	}

	slog.Debug("Fire event", "name", c.name)
	beforeState := c.stateMachine.MustState()
	c.stateMachine.Fire("mqttEvent", ev)

	eventsToPublish := c.getAndResetEventsToPublish()
	afterState := c.stateMachine.MustState()
	slog.Debug("Event fired", "fsm", c.name, "beforeState", beforeState,
		"afterState", afterState)

	afterStateInt, ok := afterState.(int)
	if ok {
		gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state{controller="%s"}`, c.name), nil)
		gauge.Set(float64(afterStateInt))
	} else {
		slog.Info("Could not create int from state", "state", afterState)
	}

	// if beforeState != afterState {
	// 	afterStateInt, ok := afterState.(int)
	// 	if ok {
	// 		gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state{controller="%s"}`, c.name), nil)
	// 		gauge.Set(float64(afterStateInt))
	// 		c.masterController.pushMetrics = true
	// 	}
	// }
	return eventsToPublish
}

func (c *BaseController) addEventsToPublish(events []MQTTPublish) {
	c.eventsToPublish = append(c.eventsToPublish, events...)
}

func (c *BaseController) getAndResetEventsToPublish() []MQTTPublish {
	events := c.eventsToPublish
	c.eventsToPublish = []MQTTPublish{}
	return events
}

func (masterController *MasterController) ProcessEvent(client mqtt.Client, ev MQTTEvent) {

	masterController.mu.Lock()
	defer masterController.mu.Unlock()

	masterController.pushMetrics = false // Reset
	masterController.detectPhonePresent(ev)
	masterController.detectLivingroomPresence(ev)
	masterController.detectLivingroomFloorlampState(ev)
	masterController.detectNighttime(ev)
	masterController.detectTVPower(ev)
	masterController.detectMPDPlay(ev)
	masterController.detectKitchenAmpPower(ev)
	masterController.detectKitchenAudioPlaying(ev)
	masterController.detectBedroomBlindsOpen(ev)

	for _, c := range *masterController.controllers {
		controller := c
		go func() {
			// For reliability, we call each loop in its own goroutine (yes, one
			// per message), so that one loop can be stuck while others still
			// make progress.

			controller.Lock()
			defer controller.Unlock()

			var toPublish []MQTTPublish
			if !controller.IsInitialized() {
				// If initialize requires other processes to update some state to determine
				// correct init state it can be requested  by events returned here
				// But the Initialize method must make sure to not request unneccessarily often
				toPublish = append(toPublish, controller.Initialize(masterController)...)
			}
			if controller.IsInitialized() {
				toPublish = append(toPublish, controller.ProcessEvent(ev)...)
			}

			for _, result := range toPublish {
				count = count + 1
				go func(toPublish MQTTPublish) {
					if toPublish.Wait != 0 {
						time.Sleep(toPublish.Wait)
					}
					client.Publish(toPublish.Topic, toPublish.Qos, toPublish.Retained, toPublish.Payload)
				}(result)
			}
		}()
	}
	//masterController.checkPushMetrics()
}

func (masterController *MasterController) checkPushMetrics() {
	if masterController.pushMetrics {
		ctx := context.Background()
		metrics.PushMetrics(ctx, "http://"+masterController.metricsConfig.MetricsAddress+"/api/v1/import/prometheus", false, nil)
		// ctx := context.Background()
		// //ctx, err := context.WithTimeout(context.Background(), 100*time.Millisecond)
		// if err != nil {
		// 	slog.Error("Error creating context with timeout", "error", err)
		// } else {
		// 	metrics.PushMetrics(ctx, "http://"+masterController.metricsConfig.MetricsAddress+"/api/v1/import/prometheus", false, nil)
		// }
	}
}

// Guards

func (l *MasterController) guardStateMPDOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("mpdPlay")
	slog.Debug("guardStateMPDOn", "check", check)
	return check
}

func (l *MasterController) guardStateMPDOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("mpdPlay")
	slog.Debug("guardStateMPDOff", "check", check)
	return check
}

func (l *MasterController) guardStateSnapcastOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("snapcast")
	slog.Debug("guardStateSnapcastOn", "check", check)
	return check
}

func (l *MasterController) guardStateSnapcastOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("snapcast")
	slog.Debug("guardStateSnapcastOff", "check", check)
	return check
}

func (l *MasterController) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("phonePresent") &&
		l.stateValueMap.requireTrue("nighttime") &&
		l.stateValueMap.requireTrueRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOnLamp", "check", check)
	return check
}

func (l *MasterController) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("phonePresent") ||
		l.stateValueMap.requireFalse("nighttime") ||
		l.stateValueMap.requireTrueNotRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOffLamp", "check", check)
	return check
}

func (l *MasterController) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("tvpower")
	slog.Debug("guardStateTvOn", "check", check)
	return check
}

func (l *MasterController) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("tvpower")
	slog.Debug("guardStateTvOff", "check", check)
	return check
}

func (l *MasterController) guardStateTvOffLong(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("tvpower", 30*time.Minute)
	slog.Debug("guardStateTvOff", "check", check)
	return check
}

func (l *MasterController) guardStateKitchenAmpOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("kitchenaudioplaying")
	slog.Debug("guardStateKitchenAmpOn", "check", check)
	return check
}

func (l *MasterController) guardStateKitchenAmpOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("kitchenaudioplaying", 10*time.Minute)
	slog.Debug("guardStateKitchenAmpOn", "check", check)
	return check
}

func (l *MasterController) guardStateBedroomBlindsOpen(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("nighttime")
	slog.Debug("guardStateBedroomBlindsOpen", "check", check)
	return check
}

func (l *MasterController) guardStateBedroomBlindsClosed(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("nighttime")
	slog.Debug("guardStateBedroomBlindsClosed", "check", check)
	return check
}

// Detections

func (l *MasterController) detectPhonePresent(ev MQTTEvent) {
	if ev.Topic == "routeros/wificlients" {
		var wifiClients []routerosmqtt.WifiClient

		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
		if err != nil {
			slog.Error("Could not parse payload", "topic", "routeros/wificlients", "error", err)
			return
		}
		found := false
		for _, wifiClient := range wifiClients {
			if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
				found = true
				break
			}
		}
		slog.Debug("detectPhonePresent", "phonePresent", found)
		l.stateValueMap.setState("phonePresent", found)
	}
}

func (l *MasterController) detectLivingroomPresence(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/livingroom-presence" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["occupancy"]
		if !exists || val == nil {
			return
		}
		l.stateValueMap.setState("livingroomPresence", val.(bool))
	}
}

func (l *MasterController) detectLivingroomFloorlampState(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/livingroom-floorlamp" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["state"]
		if !exists || val == nil {
			return
		}
		state := val.(string)
		on := false
		if state == "ON" {
			on = true
		}
		l.stateValueMap.setState("livingroomFloorlamp", on)
	}
}

func (l *MasterController) detectNighttime(ev MQTTEvent) {
	if ev.Topic == "regelverk/ticker/timeofday" {
		l.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
	}
}

func (l *MasterController) detectTVPower(ev MQTTEvent) {
	if ev.Topic == "regelverk/state/tvpower" {
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Error("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
		}
		l.stateValueMap.setState("tvpower", tvPower)
	}
}

func (l *MasterController) detectMPDPlay(ev MQTTEvent) {
	if ev.Topic == "mpd/status" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["state"]
		if !exists || val == nil {
			return
		}
		l.stateValueMap.setState("mpdPlay", val.(string) == "play")
	}
}

func (l *MasterController) detectKitchenAmpPower(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/kitchen-amp" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["state"]
		if !exists || val == nil {
			return
		}
		l.stateValueMap.setState("kitchenamppower", val.(string) == "ON")
	}
}

func (l *MasterController) detectKitchenAudioPlaying(ev MQTTEvent) {
	if ev.Topic == "kitchen/pulseaudio/state" {
		var pulseaudioState pulseaudiomqtt.PulseAudioState
		err := json.Unmarshal(ev.Payload.([]byte), &pulseaudioState)
		if err != nil {
			slog.Error("Could not parse payload", "topic", "kitchen/pulseaudio/state", "error", err)
			return
		}
		l.stateValueMap.setState("kitchenaudioplaying", pulseaudioState.DefaultSink.State == 0)
	}
}

func (l *MasterController) detectBedroomBlindsOpen(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/blinds-bedroom" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["position"]
		if !exists || val == nil {
			return
		}
		l.stateValueMap.setState("bedroomblindsopen", val.(float64) > 50)
	}
}

func setIkeaTretaktPower(topic string, on bool) MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return MQTTPublish{
		Topic:    topic,
		Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
		Qos:      2,
		Retained: true,
	}
}
