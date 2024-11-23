package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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

// Output

func livingroomFloorlampOutput(on bool) []MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
			Qos:      2,
			Retained: true,
		},
	}
}

func kitchenAmpPowerOutput(on bool) []MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
			Qos:      2,
			Retained: true,
		},
	}
}

func tvPowerOffOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/ikea_uttag/set",
			Payload:  "{\"state\": \"OFF\", \"power_on_behavior\": \"ON\"}",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}

func tvPowerOnOutput() []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/ikea_uttag/set",
			Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "power_on!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "volume_38!",
			Qos:      2,
			Retained: false,
			Wait:     2 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt1!",
			Qos:      2,
			Retained: false,
			Wait:     3 * time.Second,
		},
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:hdmi-stereo",
			Qos:      2,
			Retained: false,
			Wait:     3 * time.Second,
		},
	}

	// Need to wait here since a newly started TV is not receptive first 20 or so seconds
	for i := int64(15); i < 40; i++ {
		p := MQTTPublish{
			Topic:    "samsungremote/key/reconnectsend",
			Payload:  "KEY_VOLDOWN",
			Qos:      2,
			Retained: false,
			Wait:     time.Duration(i) * time.Second / 2,
		}
		result = append(result, p)
	}
	return result

}

func mpdPlayOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "rotel/command/send",
			Payload:  "power_on!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt2!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:iec958-stereo+input:analog-stereo",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}

// Guards

func (l *StateMachineMQTTBridge) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("phonePresent") &&
		l.stateValueMap.require("nighttime") &&
		l.stateValueMap.requireRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOnLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("phonePresent") ||
		l.stateValueMap.requireNot("nighttime") ||
		l.stateValueMap.requireNotRecently("livingroomPresence", 10*time.Minute)
	slog.Debug("guardTurnOffLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("tvpower")
	slog.Debug("uardStateTvOn", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("tvpower")
	slog.Debug("guardStateTvOff", "check", check)
	return check
}

// Actions

func (l *StateMachineMQTTBridge) turnOnLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Debug("turnOnLamp")
	l.eventsToPublish = append(l.eventsToPublish, livingroomFloorlampOutput(true)...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Debug("turnOffLamp")
	l.eventsToPublish = append(l.eventsToPublish, livingroomFloorlampOutput(false)...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOnTvAppliances(_ context.Context, _ ...any) error {
	slog.Debug("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, tvPowerOnOutput()...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffTvAppliances(_ context.Context, _ ...any) error {
	slog.Debug("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, tvPowerOffOutput()...)
	return nil
}

// Detections

func (l *StateMachineMQTTBridge) detectPhonePresent(ev MQTTEvent) {
	if ev.Topic == "routeros/wificlients" {
		var wifiClients []routerosmqtt.WifiClient

		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
		if err != nil {
			slog.Debug("Could not parse payload", "topic", "routeros/wificlients", "error", err)
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
		l.stateValueMap.setState("livingroomPresence", m["occupancy"].(bool))
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

func (l *StateMachineMQTTBridge) detectNighttime(ev MQTTEvent) {
	if ev.Topic == "regelverk/ticker/timeofday" {
		l.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
	}
}

func (l *StateMachineMQTTBridge) detectTVPower(ev MQTTEvent) {
	if ev.Topic == "regelverk/state/tvpower" {
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Debug("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
		}
		l.stateValueMap.setState("tvpower", tvPower)
	}
}

func (l *StateMachineMQTTBridge) detectMPDPlay(ev MQTTEvent) {
	if ev.Topic == "mpd/status" {
		m := parseJSONPayload(ev)
		l.stateValueMap.setState("mpdPlay", m["state"].(string) == "play")
	}
}
