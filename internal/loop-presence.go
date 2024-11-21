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
	triggerNightime = "Nighttime"
	triggerDaytime  = "Daytime"
)

const (
	eventLampOn       = "LampOn"
	eventLampOff      = "LampOff"
	eventPhonePresent = "PhonePresent"
	eventPhoneAbsent  = "PhoneAbsent"
)

type PresenceLoop struct {
	statusLoop
	livingroomLastAbsence   time.Time
	livingroomLastPresence  time.Time
	livingroomPresence      bool
	phoneWifiLastPresence   time.Time
	phoneWifiPresence       bool
	livingroomLampFSMBridge LivingroomLampFsmMQTTBridge
	isInitialized           bool
}

type StateMachineMQTTBridge struct {
	stateMachine    *stateless.StateMachine
	eventsToPublish []MQTTPublish
}

type LivingroomLampFsmMQTTBridge struct {
	StateMachineMQTTBridge
	phonePresent bool
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

	baseBridge := StateMachineMQTTBridge{eventsToPublish: []MQTTPublish{}}
	l.livingroomLampFSMBridge = LivingroomLampFsmMQTTBridge{StateMachineMQTTBridge: baseBridge}
	livingroomLampFSM := stateless.NewStateMachine(eventLampOn)
	//livingroomLampFSM.OnUnhandledTrigger(func(_ context.Context, state stateless.State, _ stateless.Trigger, _ []string) {})
	livingroomLampFSM.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	livingroomLampFSM.Configure(eventLampOn).
		OnEntry(l.livingroomLampFSMBridge.turnOnLamp).
		Permit("mqttEvent", eventLampOff, l.livingroomLampFSMBridge.guardTurnOffLamp)

	livingroomLampFSM.Configure(eventLampOff).
		OnEntry(l.livingroomLampFSMBridge.turnOffLamp).
		Permit("mqttEvent", eventLampOn, l.livingroomLampFSMBridge.guardTurnOnLamp)

	l.livingroomLampFSMBridge.stateMachine = livingroomLampFSM
	l.isInitialized = true
	slog.Info("FSM initialized")
}

func (l *PresenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Info("Process event", "event", ev)
		l.livingroomLampFSMBridge.detectPhonePresent(ev)
		slog.Info("Fire event", "event", ev)
		l.livingroomLampFSMBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.livingroomLampFSMBridge.eventsToPublish
		slog.Info("Event fired", "event", ev, "eventsToPublish", eventsToPublish)
		l.livingroomLampFSMBridge.eventsToPublish = []MQTTPublish{}
		return eventsToPublish

	} else {
		slog.Info("Cannot process event is not initialized", "event", ev)
		return []MQTTPublish{}
	}
}

func (l *LivingroomLampFsmMQTTBridge) guardTurnOnLamp(_ context.Context, _ ...any) bool {
	slog.Info("guardTurnOnLamp", "phonePresent", l.phonePresent)
	return l.phonePresent
}

func (l *LivingroomLampFsmMQTTBridge) turnOnLamp(_ context.Context, _ ...any) error {
	slog.Info("turnOnLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomLampMQTTPublish(true)}...)
	return nil
}

func (l *LivingroomLampFsmMQTTBridge) guardTurnOffLamp(_ context.Context, _ ...any) bool {
	slog.Info("guardTurnOffLamp", "phonePresent", l.phonePresent)
	return !l.phonePresent
}

func (l *LivingroomLampFsmMQTTBridge) turnOffLamp(_ context.Context, _ ...any) error {
	slog.Info("turnOffLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomLampMQTTPublish(false)}...)
	return nil
}

func (l *LivingroomLampFsmMQTTBridge) detectPhonePresent(ev MQTTEvent) {
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
		l.phonePresent = found
	}
}

// func (l *PresenceLoop) processPresence(ev MQTTEvent) []MQTTPublish {
// 	switch ev.Topic {

// 	case "routeros/wificlients":
// 		var wifiClients []routerosmqtt.WifiClient

// 		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
// 		if err != nil {
// 			slog.Debug("Could not parse payload", "topic", "routeros/wificlients")
// 		}
// 		found := false
// 		for _, wifiClient := range wifiClients {
// 			if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
// 				found = true
// 				l.phoneWifiLastPresence = time.Now()
// 				break
// 			}
// 		}
// 		l.phoneWifiPresence = found

// 		return []MQTTPublish{
// 			{
// 				Topic: "regelverk/presence/phone",
// 				Payload: fmt.Sprintf("{\"present\": \"%t\"}",
// 					l.phoneWifiPresence),
// 				Qos:      2,
// 				Retained: true,
// 			},
// 		}

// 	case "zigbee2mqtt/livingroom-presence":
// 		m := parseJSONPayload(ev)
// 		present := m["occupancy"].(bool)

// 		absentSeconds := 0
// 		presentSeconds := 0
// 		if present {
// 			l.livingroomLastPresence = time.Now()
// 			presentSeconds = int(time.Now().Sub(l.livingroomLastAbsence).Seconds())
// 		} else {
// 			l.livingroomLastAbsence = time.Now()
// 			absentSeconds = int(time.Now().Sub(l.livingroomLastPresence).Seconds())
// 		}

// 		return []MQTTPublish{
// 			{
// 				Topic: "regelverk/presence/livingroom",
// 				Payload: fmt.Sprintf("{\"present\": \"%t\", \"absentSeconds\": \"%d\", \"presentSeconds\": \"%d\"}",
// 					present, absentSeconds, presentSeconds),
// 				Qos:      2,
// 				Retained: true,
// 			},
// 		}

// 	default:
// 		return nil
// 	}
// }

// func (l *PresenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
// 	return l.processPresence(ev)
// }
