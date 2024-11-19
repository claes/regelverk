package regelverk

type LivingroomLoop struct {
	statusLoop
	hasMuted bool
}

func (l *LivingroomLoop) Init(m *mqttMessageHandler, config Config) {}

func (l *LivingroomLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "regelverk/ticker/1min":
		var timeOfDay = ev.Payload.(TimeOfDay)
		if timeOfDay == Nighttime {
			return []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
					Payload:  "{\"state\": \"ON\"}",
					Qos:      2,
					Retained: true,
				},
			}
		} else {
			return []MQTTPublish{
				{
					Topic:    "zigbee2mqtt/livingroom-floorlamp/set",
					Payload:  "{\"state\": \"OFF\"}",
					Qos:      2,
					Retained: true,
				},
			}
		}
	}
	return nil
}
