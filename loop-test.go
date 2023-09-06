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

func (l *testLoop) Init(m *mqttMessageHandler) {}

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

// https://github.com/void-spark/kodi2mqtt/
// https://github.com/mqtt-smarthome/mqtt-smarthome/blob/master/Software.md

// cec
// kodi active source
// tx 1f:82:40:00

//chromecast active source
// tx 4f:84:30:00:04
// tx 4f:82:30:00

// https://www.cec-o-matic.com/

// tv activre source
// 0f:82:00:00
// 0f:80:40:00:00:00
// tx 4f:82:00:00
// tx 0f:80:30:00:00:00

// https://github.com/chbmuc/cec
// https://github.com/tobiash/hdmi-cec-mqtt
// https://www.mankier.com/1/cec-ctl
// https://forum.libreelec.tv/thread/22882-cec-client-does-not-work-on-nightly-9-8-rpi-4b/
// https://git.linuxtv.org/v4l-utils.git
// https://search.nixos.org/packages?channel=23.05&show=v4l-utils&from=0&size=50&sort=relevance&type=packages&query=cec-ctl
