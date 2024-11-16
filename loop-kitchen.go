package main

import (
	"time"
)

type kitchenLoop struct {
	statusLoop
	kitchenAmpPower         bool
	livingRoomPresent       bool
	livingRoomAbsentSeconds int
}

func (l *kitchenLoop) Init(m *mqttMessageHandler, config Config) {}

func (l *kitchenLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	loopRules := []func(MQTTEvent) []MQTTPublish{
		// l.updateRotelState,
		// l.updateTvSourceState,
		// l.turnOnAmpWhenTVOn,
		// l.turnOffAmpWhenTVOff,
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
