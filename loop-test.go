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
						Payload:  "volume_30!",
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
					MQTTPublish{
						Topic:    "samsungremote/key/reconnectsend",
						Payload:  "KEY_VOLDOWN",
						Qos:      2,
						Retained: false,
						Wait:     10 * time.Second,
					}
				}
				// Need to wait here since a newly started TV is not receptive first 20 or so seconds
				for i := int64(25); i < 20; i++ {
					returnList = append(returnList, MQTTPublish{
						Topic:    "samsungremote/key/reconnectsend",
						Payload:  "KEY_VOLDOWN",
						Qos:      2,
						Retained: false,
						Wait:     time.Duration(i) * time.Second,
					})
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

	default:
		return nil
	}
	return nil
}

func (l *testLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)
}
