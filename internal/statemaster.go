package regelverk

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	pulseaudiomqtt "github.com/claes/pulseaudio-mqtt/lib"
	routerosmqtt "github.com/claes/routeros-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/qmuntal/stateless"
)

type StateMaster struct {
	stateValueMap StateValueMap
	controllers   []Controller
}

// type Controller struct {
// 	name            string
// 	stateMachine    *stateless.StateMachine
// 	eventsToPublish []MQTTPublish
// 	isInitialized   bool
// }

// func CreateStateMachineMQTTBridgeNew(name string) Controller {
// 	return Controller{name: name, eventsToPublish: []MQTTPublish{}}
// }

// // Alternative to loops for statemachines
// // - Update common StateMaster
// // - StateMaster has one StateValueMap one many state machine bridges?
// // - Initialize non-initialized state machines
// // - Trigger initialized state machines

// func (l *Controller) getInitialState() *stateless.State {
// 	return nil
// }

// func createStateMachine(stateMaster *StateMaster) *stateless.StateMachine {
// 	// check if there is state corresponding to initial state, if so return a configured state machine
// 	// else return nil
// 	return nil
// }

// func (s *Controller) getAndResetEventsToPublish() []MQTTPublish {
// 	//Locking?
// 	events := s.eventsToPublish
// 	s.eventsToPublish = []MQTTPublish{}
// 	return events
// }

// func (s *Controller) addEventsToPublish(events []MQTTPublish) {
// 	//Locking?
// 	s.eventsToPublish = append(s.eventsToPublish, events...)
// }

// func (l *Controller) ProcessEvent(ev MQTTEvent) []MQTTPublish {
// 	slog.Debug("Process event", "name", l.name)

// 	slog.Info("Fire event", "name", l.name)
// 	beforeState := l.stateMachine.MustState()
// 	l.stateMachine.Fire("mqttEvent", ev)

// 	eventsToPublish := l.getAndResetEventsToPublish()
// 	slog.Info("Event fired", "fsm", l.name, "beforeState", beforeState,
// 		"afterState", l.stateMachine.MustState())
// 	return eventsToPublish
// }

func (l *StateMaster) Init() {
}

type Controller interface {
	IsInitialized() bool

	Initialize(sm *StateMaster)

	ProcessEvent(ev MQTTEvent) []MQTTPublish
}

type BaseController struct {
	name            string
	stateMachine    *stateless.StateMachine
	eventsToPublish []MQTTPublish
	isInitialized   bool
}

func (c *BaseController) IsInitialized() bool {
	return c.isInitialized
}

func (c *BaseController) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	slog.Debug("Process event", "name", c.name)

	slog.Info("Fire event", "name", c.name)
	beforeState := c.stateMachine.MustState()
	c.stateMachine.Fire("mqttEvent", ev)

	eventsToPublish := c.getAndResetEventsToPublish()
	slog.Info("Event fired", "fsm", c.name, "beforeState", beforeState,
		"afterState", c.stateMachine.MustState())
	return eventsToPublish
}

func (c *BaseController) addEventsToPublish(events []MQTTPublish) {
	//Locking?
	c.eventsToPublish = append(c.eventsToPublish, events...)
}

func (c *BaseController) getAndResetEventsToPublish() []MQTTPublish {
	//Locking?
	events := c.eventsToPublish
	c.eventsToPublish = []MQTTPublish{}
	return events
}

type TVController struct {
	BaseController
}

func (c *TVController) Initialize(stateMaster StateMaster) {
	var initialState tvState
	if stateMaster.stateValueMap.requireTrue("tvpower") {
		initialState = stateTvOn
	} else if stateMaster.stateValueMap.requireFalse("tvpower") {
		initialState = stateTvOff
	} else {
		return
	}

	sm := stateless.NewStateMachine(initialState) // can this be reliable determined early on? probably not
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(stateTvOn).
		OnEntry(c.turnOnTvAppliances).
		Permit("mqttEvent", stateTvOff, stateMaster.guardStateTvOff)

	sm.Configure(stateTvOff).
		OnEntry(c.turnOffTvAppliances).
		Permit("mqttEvent", stateTvOn, stateMaster.guardStateTvOn).
		Permit("mqttEvent", stateTvOffLong, stateMaster.guardStateTvOffLong)

	sm.Configure(stateTvOffLong).
		OnEntry(c.turnOffTvAppliancesLong).
		Permit("mqttEvent", stateTvOn, stateMaster.guardStateTvOn)

	c.stateMachine = sm
	c.isInitialized = true
}

func (c *TVController) turnOnTvAppliances(_ context.Context, _ ...any) error {
	c.addEventsToPublish(tvPowerOnOutput())
	return nil
}

func (c *TVController) turnOffTvAppliances(_ context.Context, _ ...any) error {
	c.addEventsToPublish(tvPowerOffOutput())
	return nil
}

func (c *TVController) turnOffTvAppliancesLong(_ context.Context, _ ...any) error {
	c.addEventsToPublish(tvPowerOffLongOutput())
	return nil
}

// Need:
// Way to assign controllers to StateMaster
// Interface for controller
//  ProcessEvent(ev MQTTEvent) []MQTTPublish
// Way to initialize controllers
// Types of controllers?
// Such as statemachine controller?
// Loop controller
// A controller base?

func (stateMaster *StateMaster) ProcessEvent(client mqtt.Client, ev MQTTEvent) {

	// Update state value map
	stateMaster.detectPhonePresent(ev)
	stateMaster.detectLivingroomPresence(ev)
	stateMaster.detectLivingroomFloorlampState(ev)
	stateMaster.detectNighttime(ev)
	stateMaster.detectTVPower(ev)
	stateMaster.detectMPDPlay(ev)
	stateMaster.detectKitchenAmpPower(ev)
	stateMaster.detectKitchenAudioPlaying(ev)
	stateMaster.detectBedroomBlindsOpen(ev)

	for _, c := range stateMaster.controllers {
		controller := c
		go func() {
			// For reliability, we call each loop in its own goroutine (yes, one
			// per message), so that one loop can be stuck while others still
			// make progress.

			// lock around this?
			var results []MQTTPublish
			if !controller.IsInitialized() {
				controller.Initialize(stateMaster)
			}
			if controller.IsInitialized() {
				results = controller.ProcessEvent(ev)
			}
			// end lock around this?

			for _, result := range results {
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
}

// Guards
func (l *StateMaster) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("phonePresent") &&
		l.stateValueMap.requireTrue("nighttime") &&
		l.stateValueMap.requireTrueRecently("livingroomPresence", 10*time.Minute)
	slog.Info("guardTurnOnLamp", "check", check)
	return check
}

func (l *StateMaster) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("phonePresent") ||
		l.stateValueMap.requireFalse("nighttime") ||
		l.stateValueMap.requireTrueNotRecently("livingroomPresence", 10*time.Minute)
	slog.Info("guardTurnOffLamp", "check", check)
	return check
}

func (l *StateMaster) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("tvpower")
	slog.Info("guardStateTvOn", "check", check)
	return check
}

func (l *StateMaster) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("tvpower")
	slog.Info("guardStateTvOff", "check", check)
	return check
}

func (l *StateMaster) guardStateTvOffLong(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("tvpower", 30*time.Minute)
	slog.Info("guardStateTvOff", "check", check)
	return check
}

func (l *StateMaster) guardStateKitchenAmpOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("kitchenaudioplaying")
	slog.Info("guardStateKitchenAmpOn", "check", check)
	return check
}

func (l *StateMaster) guardStateKitchenAmpOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("kitchenaudioplaying", 10*time.Minute)
	slog.Info("guardStateKitchenAmpOn", "check", check)
	return check
}

func (l *StateMaster) guardStateBedroomBlindsOpen(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("nighttime")
	slog.Info("guardStateBedroomBlindsOpen", "check", check)
	return check
}

func (l *StateMaster) guardStateBedroomBlindsClosed(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("nighttime")
	slog.Info("guardStateBedroomBlindsClosed", "check", check)
	return check
}

// Detections

func (l *StateMaster) detectPhonePresent(ev MQTTEvent) {
	if ev.Topic == "routeros/wificlients" {
		var wifiClients []routerosmqtt.WifiClient

		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
		if err != nil {
			slog.Info("Could not parse payload", "topic", "routeros/wificlients", "error", err)
			return
		}
		found := false
		for _, wifiClient := range wifiClients {
			if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
				found = true
				break
			}
		}
		slog.Info("detectPhonePresent", "phonePresent", found)
		l.stateValueMap.setState("phonePresent", found)
	}
}

func (l *StateMaster) detectLivingroomPresence(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/livingroom-presence" {
		m := parseJSONPayload(ev)
		l.stateValueMap.setState("livingroomPresence", m["occupancy"].(bool))
	}
}

func (l *StateMaster) detectLivingroomFloorlampState(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/livingroom-floorlamp" {
		m := parseJSONPayload(ev)
		state := m["state"].(string)
		on := false
		if state == "ON" {
			on = true
		}
		l.stateValueMap.setState("livingroomFloorlamp", on)
	}
}

func (l *StateMaster) detectNighttime(ev MQTTEvent) {
	if ev.Topic == "regelverk/ticker/timeofday" {
		l.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
	}
}

func (l *StateMaster) detectTVPower(ev MQTTEvent) {
	if ev.Topic == "regelverk/state/tvpower" {
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Info("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
		}
		l.stateValueMap.setState("tvpower", tvPower)
	}
}

func (l *StateMaster) detectMPDPlay(ev MQTTEvent) {
	if ev.Topic == "mpd/status" {
		m := parseJSONPayload(ev)
		l.stateValueMap.setState("mpdPlay", m["state"].(string) == "play")
	}
}

func (l *StateMaster) detectKitchenAmpPower(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/kitchen-amp" {
		m := parseJSONPayload(ev)
		l.stateValueMap.setState("kitchenamppower", m["state"].(string) == "ON")
	}
}

func (l *StateMaster) detectKitchenAudioPlaying(ev MQTTEvent) {
	if ev.Topic == "kitchen/pulseaudio/state" {
		var pulseaudioState pulseaudiomqtt.PulseAudioState
		err := json.Unmarshal(ev.Payload.([]byte), &pulseaudioState)
		if err != nil {
			slog.Info("Could not parse payload", "topic", "kitchen/pulseaudio/state", "error", err)
			return
		}
		l.stateValueMap.setState("kitchenaudioplaying", pulseaudioState.DefaultSink.State == 0)
	}
}

func (l *StateMaster) detectBedroomBlindsOpen(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/blinds-bedroom" {
		m := parseJSONPayload(ev)
		pos, exists := m["position"]
		if exists {
			l.stateValueMap.setState("bedroomblindsopen", pos.(float64) > 50)
		}
	}
}
