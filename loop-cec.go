package main

import (
	"log/slog"
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
		//command := ev.Payload.(string)
		//command := fmt.Sprintf("%v", ev.Payload)
		slog.Info("cec/command payload", "command", command)
		if command == "01:90:00:00:00" {
			slog.Info("tv power true")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(true),
					Qos:      2,
					Retained: true,
				},
			}
		} else if command == "01:90:01:00:00" {
			slog.Info("tv power false")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(false),
					Qos:      2,
					Retained: true,
				},
			}
		} else {
			slog.Info("no tv power match")
		}
	default:
		return nil
	}
	return nil
}

func (l *cecLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)

}
