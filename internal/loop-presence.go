package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	routerosmqtt "github.com/claes/routeros-mqtt/lib"
	"github.com/qmuntal/stateless"
)

const (
	stateLampOn  = "LampOn"
	stateLampOff = "LampOff"
)

type PresenceLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
}

type StateMachineMQTTBridge struct {
	stateMachine    *stateless.StateMachine
	eventsToPublish []MQTTPublish
	stateValueMap   StateValueMap
}

func CreateStateMachineMQTTBridge() StateMachineMQTTBridge {
	return StateMachineMQTTBridge{eventsToPublish: []MQTTPublish{}, stateValueMap: NewStateValueMap()}
}

func livingroomLampMQTTPublish(on bool) MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return MQTTPublish{
		Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
		Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
		Qos:      2,
		Retained: true,
	}
}

func (l *PresenceLoop) Init(m *mqttMessageHandler, config Config) {
	slog.Info("Initializing FSM")
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge()

	s := l.stateMachineMQTTBridge.stateValueMap.getState("livingroomFloorlamp")
	if s.isDefined {
		slog.Info("Floorlamp state determined")
		var sm *stateless.StateMachine
		if s.value {
			sm = stateless.NewStateMachine(stateLampOn)
		} else {
			sm = stateless.NewStateMachine(stateLampOff)
		}
		//livingroomLampFSM.OnUnhandledTrigger(func(_ context.Context, state stateless.State, _ stateless.Trigger, _ []string) {})
		sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

		sm.Configure(stateLampOn).
			OnEntry(l.stateMachineMQTTBridge.turnOnLamp).
			Permit("mqttEvent", stateLampOff, l.stateMachineMQTTBridge.guardTurnOffLamp)

		sm.Configure(stateLampOff).
			OnEntry(l.stateMachineMQTTBridge.turnOffLamp).
			Permit("mqttEvent", stateLampOn, l.stateMachineMQTTBridge.guardTurnOnLamp)

		l.stateMachineMQTTBridge.stateMachine = sm
		l.isInitialized = true
		slog.Info("FSM initialized")
	}
}

func (l *PresenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	l.stateMachineMQTTBridge.detectLivingroomFloorlampState(ev)
	if l.isInitialized {
		slog.Info("Process event")
		l.stateMachineMQTTBridge.detectPhonePresent(ev)
		l.stateMachineMQTTBridge.detectLivingroomPresence(ev)
		l.stateMachineMQTTBridge.stateValueMap.LogState()
		slog.Info("Fire event")
		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.stateMachineMQTTBridge.eventsToPublish
		slog.Info("Event fired", "state", l.stateMachineMQTTBridge.stateMachine.MustState())
		l.stateMachineMQTTBridge.eventsToPublish = []MQTTPublish{}
		return eventsToPublish
	} else {
		slog.Info("Cannot process event is not initialized", "event", ev)
		return []MQTTPublish{}
	}
}

func (l *StateMachineMQTTBridge) guardTurnOnLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("phonePresent") && l.stateValueMap.requireRecently("livingroomPresence", 10*time.Minute)
	slog.Info("guardTurnOnLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) turnOnLamp(_ context.Context, _ ...any) error {
	slog.Info("turnOnLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomLampMQTTPublish(true)}...)
	return nil
}

func (l *StateMachineMQTTBridge) guardTurnOffLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("phonePresent") || l.stateValueMap.requireNotRecently("livingroomPresence", 10*time.Minute)
	slog.Info("guardTurnOffLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) turnOffLamp(_ context.Context, _ ...any) error {
	slog.Info("turnOffLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomLampMQTTPublish(false)}...)
	return nil
}

func (l *StateMachineMQTTBridge) detectPhonePresent(ev MQTTEvent) {
	if ev.Topic == "routeros/wificlients" {
		var wifiClients []routerosmqtt.WifiClient

		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
		if err != nil {
			slog.Debug("Could not parse payload", "topic", "routeros/wificlients")
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

func (l *StateMachineMQTTBridge) detectLivingroomPresence(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/livingroom-presence" {
		m := parseJSONPayload(ev)
		present := m["occupancy"].(bool)
		l.stateValueMap.setState("livingroomPresence", present)
	}
}

func (l *StateMachineMQTTBridge) detectLivingroomFloorlampState(ev MQTTEvent) {
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
