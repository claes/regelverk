package main

import (
	"log/slog"
	"strconv"
	"strings"
)

type cecLoop struct {
	statusLoop
}

func (l *cecLoop) Init(m *mqttMessageHandler) {}

func (l *cecLoop) turnOnAmpWhenTVOn(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {

	// case "cec/command":
	case "cec/message/hex/rx":
		fallthrough
	case "cec/message/hex/tx":
		command := strings.ToUpper(string(ev.Payload.([]byte)))
		slog.Debug("CEC command", "command", command)
		switch command {
		case "01:90:00:00:00":
			slog.Debug("TV power")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(true),
					Qos:      2,
					Retained: true,
				},
			}
		case "01:90:01:00:00":
			slog.Debug("TV standby")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvpower",
					Payload:  strconv.FormatBool(false),
					Qos:      2,
					Retained: true,
				},
			}
		case "0F:82:00:00:00:00":
			fallthrough
		case "0F:82:00:00":
			slog.Debug("TV active source")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvsource",
					Payload:  "tv",
					Qos:      2,
					Retained: true,
				},
			}
		case "1F:82:40:00:00:00":
			fallthrough
		case "1F:82:40:00":
			slog.Debug("Mediaflix active source")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvsource",
					Payload:  "mediaflix",
					Qos:      2,
					Retained: true,
				},
			}
		case "4F:82:30:00:00:00":
			fallthrough
		case "4F:82:30:00":
			slog.Debug("Chromecast active source")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvsource",
					Payload:  "chromecast",
					Qos:      2,
					Retained: true,
				},
			}
		case "4F:82:20:00:00:00":
			fallthrough
		case "4F:82:20:00":
			slog.Debug("Bluray active source")
			return []MQTTPublish{
				{
					Topic:    "regelverk/state/tvsource",
					Payload:  "bluray",
					Qos:      2,
					Retained: true,
				},
			}
		case "0F:36":
			slog.Debug("TV requests standby")
			return nil
		default:
			slog.Debug("CEC command not recognized", "command", command)
			return nil
		}
	default:
		return nil
	}
}

func (l *cecLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.turnOnAmpWhenTVOn(ev)
}
