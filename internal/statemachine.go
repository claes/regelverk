package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	pulseaudiomqtt "github.com/claes/pulseaudio-mqtt/lib"
	routerosmqtt "github.com/claes/routeros-mqtt/lib"
	"github.com/qmuntal/stateless"
)

type StateMachineMQTTBridge struct {
	name            string
	stateMachine    *stateless.StateMachine
	eventsToPublish []MQTTPublish
	stateValueMap   StateValueMap
}

func CreateStateMachineMQTTBridge(name string) StateMachineMQTTBridge {
	return StateMachineMQTTBridge{name: name, eventsToPublish: []MQTTPublish{}, stateValueMap: NewStateValueMap()}
}

// Output

func setIkeaTretaktPower(topic string, on bool) []MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return []MQTTPublish{
		{
			Topic:    topic,
			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
			Qos:      2,
			Retained: true,
		},
	}
}

func livingroomFloorlampOutput(on bool) []MQTTPublish {
	return setIkeaTretaktPower("zigbee2mqtt/livingroom-floorlamp/set", on)
}

func kitchenAmpPowerOutput(on bool) []MQTTPublish {
	return setIkeaTretaktPower("zigbee2mqtt/kitchen-amp/set", on)
}

// func livingroomFloorlampOutput(on bool) []MQTTPublish {
// 	state := "OFF"
// 	if on {
// 		state = "ON"
// 	}
// 	return []MQTTPublish{
// 		{
// 			Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
// 			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
// 			Qos:      2,
// 			Retained: true,
// 		},
// 	}
// }

// func kitchenAmpPowerOutput(on bool) []MQTTPublish {
// 	state := "OFF"
// 	if on {
// 		state = "ON"
// 	}
// 	return []MQTTPublish{
// 		{
// 			Topic:    "zigbee2mqtt/kitechen-amp/set",
// 			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
// 			Qos:      2,
// 			Retained: true,
// 		},
// 	}
// }

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

func tvPowerOffLongOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "rotel/command/send",
			Payload:  "power_off!",
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
	slog.Info("guardTurnOnLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("phonePresent") ||
		l.stateValueMap.requireNot("nighttime") ||
		l.stateValueMap.requireNotRecently("livingroomPresence", 10*time.Minute)
	slog.Info("guardTurnOffLamp", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("tvpower")
	slog.Info("guardStateTvOn", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNot("tvpower")
	slog.Info("guardStateTvOff", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateTvOffLong(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNotRecently("tvpower", 30*time.Minute)
	slog.Info("guardStateTvOff", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateKitchenAmpOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.require("kitchenaudioplaying")
	slog.Info("guardStateKitchenAmpOn", "check", check)
	return check
}

func (l *StateMachineMQTTBridge) guardStateKitchenAmpOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireNotRecently("kitchenaudioplaying", 10*time.Minute)
	slog.Info("guardStateKitchenAmpOn", "check", check)
	return check
}

// Actions

func (l *StateMachineMQTTBridge) turnOnLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Info("turnOnLamp")
	l.eventsToPublish = append(l.eventsToPublish, livingroomFloorlampOutput(true)...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffLivingroomFloorlamp(_ context.Context, _ ...any) error {
	slog.Info("turnOffLamp")
	l.eventsToPublish = append(l.eventsToPublish, livingroomFloorlampOutput(false)...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOnTvAppliances(_ context.Context, _ ...any) error {
	slog.Info("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, tvPowerOnOutput()...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffTvAppliances(_ context.Context, _ ...any) error {
	slog.Info("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, tvPowerOffOutput()...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffTvAppliancesLong(_ context.Context, _ ...any) error {
	slog.Info("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, tvPowerOffLongOutput()...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOnKitchenAmp(_ context.Context, _ ...any) error {
	slog.Info("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, kitchenAmpPowerOutput(true)...)
	return nil
}

func (l *StateMachineMQTTBridge) turnOffKitchenAmp(_ context.Context, _ ...any) error {
	slog.Info("turnOnTvAppliances")
	l.eventsToPublish = append(l.eventsToPublish, kitchenAmpPowerOutput(false)...)
	return nil
}

// Detections

func (l *StateMachineMQTTBridge) detectPhonePresent(ev MQTTEvent) {
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
			slog.Info("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
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

func (l *StateMachineMQTTBridge) detectKitchenAmpPower(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/kitchen-amp" {
		m := parseJSONPayload(ev)
		l.stateValueMap.setState("kitchenamppower", m["state"].(string) == "ON")
	}
}

func (l *StateMachineMQTTBridge) detectKitchenAudioPlaying(ev MQTTEvent) {
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
