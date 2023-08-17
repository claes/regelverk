package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type tvLoop struct {
	statusLoop
	tvLastActive time.Time
	tvOn         bool
}

func parseJSONPayload(ev MQTTEvent) map[string]interface{} {
	var payload interface{}
	payloadJson := string(ev.Payload.([]byte))
	err := json.Unmarshal([]byte(payloadJson), &payload)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	m := payload.(map[string]interface{})
	return m
}

func (l *tvLoop) turnOnAmpWhenTVOn(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "zigbee2mqtt/tv":
		m := parseJSONPayload(ev)
		power := m["power"].(float64)
		if power > 30.0 {
			l.tvLastActive = time.Now()
			l.tvOn = true
			return []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/ikea_uttag/set",
					Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
					Qos:      1,
					Retained: false,
				},
			}
		} else if power < 5.0 {
			l.tvOn = false
		}
	case "regelverk/ticker/1s":
		fmt.Printf("Tick %v %v\n", l.tvOn, l.tvLastActive)
		if !l.tvOn && l.tvLastActive.Add(1*time.Minute).Before(time.Now()) {
			return []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/ikea_uttag/set",
					Payload:  "{\"state\": \"OFF\", \"power_on_behavior\": \"ON\"}",
					Qos:      1,
					Retained: false,
				},
			}
		}

	default:
		return nil
	}
	return nil
}

func (l *tvLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)
}

// func (l *tvLoop) ProcessEventOld(ev MQTTEvent) []MQTTPublish {
// 	switch ev.Topic {
// 	case "zigbee2mqtt/tv":
// 		m := parseJSONPayload(ev)
// 		power := m["power"].(float64)

// 		if power > 70.0 {
// 			l.tvTurnedOn = true
// 		} else if power <= 70.0 {
// 			l.tvTurnedOn = false
// 		}
// 	default:
// 		return nil // did not influence state
// 	}

// 	if l.tvTurnedOn {
// 		return []MQTTPublish{
// 			{
// 				Topic:    "zigbee2mqtt/ikea_uttag/set",
// 				Payload:  "{\"state\": \"OFF\", \"power_on_behavior\": \"ON\"}",
// 				Retained: false,
// 			},
// 		}
// 	} else if !l.tvTurnedOn {
// 		return []MQTTPublish{
// 			{
// 				Topic:    "zigbee2mqtt/ikea_uttag/set",
// 				Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
// 				Retained: false,
// 			},
// 		}
// 	}
// 	return nil
// }
