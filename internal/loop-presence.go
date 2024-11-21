package regelverk

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	routerosmqtt "github.com/claes/routeros-mqtt/lib"
)

type PresenceLoop struct {
	statusLoop
	livingroomLastAbsence  time.Time
	livingroomLastPresence time.Time
	livingroomPresence     bool
	phoneWifiLastPresence  time.Time
	phoneWifiPresence      bool
}

func (l *PresenceLoop) Init(m *mqttMessageHandler, config Config) {}

func (l *PresenceLoop) processPresence(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {

	case "routeros/wificlients":
		var wifiClients []routerosmqtt.WifiClient

		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
		if err != nil {
			slog.Debug("Could not parse payload", "topic", "routeros/wificlients")
		}
		found := false
		for _, wifiClient := range wifiClients {
			if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
				found = true
				l.phoneWifiLastPresence = time.Now()
				break
			}
		}
		l.phoneWifiPresence = found

		return []MQTTPublish{
			{
				Topic: "regelverk/presence/phone",
				Payload: fmt.Sprintf("{\"present\": \"%t\"}",
					l.phoneWifiPresence),
				Qos:      2,
				Retained: true,
			},
		}

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

func (l *PresenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	return l.processPresence(ev)
}
