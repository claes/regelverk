package main

import (
	"fmt"
	"time"
)

type presenceLoop struct {
	statusLoop
	livingroomLastAbsence  time.Time
	livingroomLastPresence time.Time
	livingroomPresence     bool
}

func (l *presenceLoop) Init(m *mqttMessageHandler) {}

func (l *presenceLoop) processPresence(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {

	case "zigbee2mqtt/livingroom-presence":
		m := parseJSONPayload(ev)
		present := m["occupancy"].(bool)

		absentSeconds := 0
		presentSeconds := 0
		if present {
			l.livingroomLastPresence = time.Now()
			presentSeconds = int(time.Now().Sub(l.livingroomLastAbsence).Seconds())
		} else {
			l.livingroomLastAbsence = time.Now()
			absentSeconds = int(time.Now().Sub(l.livingroomLastPresence).Seconds())
		}

		return []MQTTPublish{
			{
				Topic: "regelverk/presence/livingroom",
				Payload: fmt.Sprintf("{\"present\": \"%t\", \"absentSeconds\": \"%d\", \"presentSeconds\": \"%d\"}",
					present, absentSeconds, presentSeconds),
				Qos:      2,
				Retained: true,
			},
		}

	default:
		return nil
	}
}

func (l *presenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.processPresence(ev)
}