package main

import (
	"strconv"
)

type cecLoop struct {
	statusLoop
}

func (l *cecLoop) Init(m *mqttMessageHandler) {}

func (l *cecLoop) turnOnAmpWhenTVOn(ev MQTTEvent) []MQTTPublish {
	// 	// "01:90:00:00:00" //power on
	// 	// "01:90:01:00:00" //standby
	// 	// "0F:36" //standby
	switch ev.Topic {

	case "cec/command":
		command := string(ev.Payload.([]byte))
		if command == "01:90:00:00:00" {
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(true),
					Qos:      2,
					Retained: true,
				},
			}
		} else if command == "01:90:01:00:00" {
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(false),
					Qos:      2,
					Retained: true,
				},
			}
		}
	default:
		return nil
	}
	return nil
}

func (l *cecLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)

}
