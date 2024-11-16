package main

import (
	"strconv"
)

type mpdLoop struct {
	statusLoop
	hasMuted bool
}

func (l *mpdLoop) Init(m *mqttMessageHandler, config Config) {}

func (l *mpdLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "mpd/status":
		m := parseJSONPayload(ev)
		state := m["state"].(string)
		if state == "play" {
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/mpdplay",
					Payload:  strconv.FormatBool(true),
					Qos:      2,
					Retained: true,
				},
			}
		} else {
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/mpdplay",
					Payload:  strconv.FormatBool(false),
					Qos:      2,
					Retained: true,
				},
			}
		}
	}
	return nil
}
