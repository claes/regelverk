package main

import (
	"log/slog"
	"strconv"
	"time"
)

type kitchenLoop struct {
	statusLoop
	kitchenAmpPower         bool
	livingRoomPresent       bool
	livingRoomAbsentSeconds int
}

func (l *kitchenLoop) Init(m *mqttMessageHandler) {}

func (l *kitchenLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	loopRules := []func(MQTTEvent) []MQTTPublish{
		l.updateRotelState,
		//		l.updateTvSourceState,
		l.turnOnAmpWhenTVOn,
		l.turnOffAmpWhenTVOff,
	}
	for _, loopRule := range loopRules {
		result := loopRule(ev)
		if result != nil {
			return result
		}
	}
	return nil
}

func (l *kitchenLoop) updatePresenceState(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "regelverk/presence/livingroom":
		m := parseJSONPayload(ev)
		l.livingRoomPresent = m["present"].(bool)
		l.livingRoomAbsentSeconds = m["absentSeconds"].(int)
	case "zigbee2mqtt/kitchen-amp":
		m := parseJSONPayload(ev)
		powerState := (m["state"] == "ON")
		if powerState != l.kitchenAmpPower {

		}
		l.kitchenAmpPower = powerState
	}
	return nil
}

func (l *kitchenLoop) toggleAmpByPresence(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "regelverk/ticker/1s":

		// if absent for more than one hour
		if !l.livingRoomPresent && l.livingRoomAbsentSeconds > 60*60 {
			returnList := []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/kitchen-amp/set",
					Payload:  "{\"state\": \"OFF\"}",
					Qos:      2,
					Retained: false,
					Wait:     0 * time.Second,
				},
			}
			return returnList
		} else {
			returnList := []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/kitchen-amp/set",
					Payload:  "{\"state\": \"ON\"}",
					Qos:      2,
					Retained: false,
					Wait:     0 * time.Second,
				},
			}
			return returnList
		}
	}
	return nil
}

func (l *kitchenLoop) turnOnAmpWhenPresent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "regelverk/state/tvpower":
		slog.Debug("regelverk/state/tvpower")
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Error("regelverk/state/tvpower error", "error", err)
			return nil
		}
		tvPowerStateChange := l.updateTvPower(tvPower)
		slog.Debug("regelverk/state/tvpower",
			"value", tvPower,
			"stateChange", tvPowerStateChange)
		if tvPowerStateChange {
			if tvPower {
				returnList := []MQTTPublish{
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
					returnList = append(returnList, p)
				}
				return returnList
			} else {
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
		}
	case "regelverk/state/mpdplay":
		slog.Debug("regelverk/state/mpdplay")

		mpdPlay, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Error("regelverk/state/mpdplay error", "error", err)
			return nil
		}
		if mpdPlay {
			returnList := []MQTTPublish{
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
			return returnList
		}

	default:
		return nil
	}
	return nil
}
