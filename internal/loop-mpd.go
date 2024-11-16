package regelverk

import (
	"strconv"
)

type MpdLoop struct {
	statusLoop
	hasMuted bool
}

func (l *MpdLoop) Init(m *mqttMessageHandler, config Config) {}

func (l *MpdLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
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
