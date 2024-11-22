package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	routerosmqtt "github.com/claes/routeros-mqtt/lib"
	"github.com/qmuntal/stateless"
)

type StateMachineMQTTBridge struct {
	stateMachine    *stateless.StateMachine
	eventsToPublish []MQTTPublish
	stateValueMap   StateValueMap
}

func CreateStateMachineMQTTBridge() StateMachineMQTTBridge {
	return StateMachineMQTTBridge{eventsToPublish: []MQTTPublish{}, stateValueMap: NewStateValueMap()}
}

func livingroomFloorlampMQTTPublish(on bool) MQTTPublish {
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

func (l *StateMachineMQTTBridge) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("phonePresent") && l.stateValueMap.requireRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOnLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) turnOnLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Debug("turnOnLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomFloorlampMQTTPublish(true)}...)
	return nil
}

func (l *StateMachineMQTTBridge) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("phonePresent") || l.stateValueMap.requireNotRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOffLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) turnOffLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Debug("turnOffLamp")
	l.eventsToPublish = append(l.eventsToPublish, []MQTTPublish{livingroomFloorlampMQTTPublish(false)}...)
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
		slog.Debug("detectPhonePresent", "phonePresent", found)
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

func (l *StateMachineMQTTBridge) detectTimeOfDay(ev MQTTEvent) {
	if ev.Topic == "regelverk/ticker/timeofday" {
		var timeOfDay = ev.Payload.(TimeOfDay)
		if timeOfDay == Nighttime {

		}
		//l.stateValueMap.setState("livingroomFloorlamp", on)
	}
}
