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
		fmt.Println("regelverk/state/mpdplay")
		mpdPlay, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			fmt.Println("Error:", err)
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

// TODO automation notes
//
// HDMI -> opt 1
// IEC958 ->  opt 2
// change of these to switch source
//
// snapcast / snapserver / snapclients

// mpd > pulseaudio profile > opt2

func (l *testLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)
}
