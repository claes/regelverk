package main

import (
	"fmt"
	"strconv"
	"time"
)

type testLoop struct {
	statusLoop
	tvPowerLastStateChange time.Time
	tvPowerLastState       bool
}

func (l *testLoop) updateTvPower(tvPower bool) bool {
	if l.tvPowerLastStateChange.Add(1 * time.Second).Before(time.Now()) {
		stateChanged := (tvPower != l.tvPowerLastState)
		if stateChanged {
			l.tvPowerLastStateChange = time.Now()
			l.tvPowerLastState = tvPower
		}
		return stateChanged
	}
	return false
}

func (l *testLoop) turnOnAmpWhenTVOn(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "regelverk/state/tvpower":
		fmt.Println("regelverk/state/tvpower")
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		tvPowerStateChange := l.updateTvPower(tvPower)
		fmt.Printf("regelverk/state/tvpower %t state change: %t \n", tvPower, tvPowerStateChange)
		if tvPowerStateChange {
			if tvPower {
				return []MQTTPublish{
					{
						Topic:    "zigbee2mqtt/ikea_uttag/set",
						Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
						Qos:      2,
						Retained: false,
					},
				}
			} else {
				return []MQTTPublish{
					{
						Topic:    "zigbee2mqtt/ikea_uttag/set",
						Payload:  "{\"state\": \"OFF\", \"power_on_behavior\": \"ON\"}",
						Qos:      2,
						Retained: false,
					},
				}
			}
		}

	default:
		return nil
	}
	return nil
}

func (l *testLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)
}
