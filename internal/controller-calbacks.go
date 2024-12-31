package regelverk

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/VictoriaMetrics/metrics"
	pulseaudiomqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	routerosmqtt "github.com/claes/mqtt-bridges/routeros-mqtt/lib"
)

func processJSON(ev MQTTEvent, topic, eventProperty string) (any, bool) {
	if ev.Topic == topic {
		m := parseJSONPayload(ev)
		if m == nil {
			return nil, false
		}
		val, exists := m[eventProperty]
		if !exists || val == nil {
			return nil, false
		}
		return val, true
	} else {
		return nil, false
	}
}

func processString(ev MQTTEvent, topic string) (string, bool) {
	if ev.Topic == topic {
		s := string(ev.Payload.([]byte))
		return s, true
	} else {
		return "", false
	}
}

func (l *MasterController) createProcessEventFunc(extractValueFunc func(MQTTEvent) (any, bool),
	stateValueFunc func(any) (string, bool),
	metricsGaugeFunc func(any) (string, float64)) func(MQTTEvent) {

	return func(ev MQTTEvent) {
		val, _ := extractValueFunc(ev)
		if val != nil {

			if stateValueFunc != nil {
				key, b := stateValueFunc(val)
				l.stateValueMap.setState(key, b)
			}

			if metricsGaugeFunc != nil {
				key, v := metricsGaugeFunc(val)
				if l.metricsConfig.CollectMetrics {
					gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`eventvalue{name="%s",realm="%s"}`, key, l.metricsConfig.MetricsRealm), nil)
					gauge.Set(v)
				}
			}
		}
	}
}

func (masterController *MasterController) registerEventCallback(callback func(MQTTEvent)) {
	masterController.eventCallbacks = append(masterController.eventCallbacks, callback)
}

func (masterController *MasterController) executeEventCallbacks(ev MQTTEvent) {
	for _, callback := range masterController.eventCallbacks {
		callback(ev)
	}
}

func (masterController *MasterController) registerEventCallbacks() {

	masterController.registerEventCallback(masterController.detectCECState)

	//masterController.registerCallback(masterController.detectPhonePresent)
	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "routeros/wificlients" {
			var wifiClients []routerosmqtt.WifiClient

			err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
			if err != nil {
				slog.Error("Could not parse payload", "topic", "routeros/wificlients", "error", err)
				return
			}
			found := false
			for _, wifiClient := range wifiClients {
				if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
					found = true
					break
				}
			}
			masterController.stateValueMap.setState("phonePresent", found)
		}
	})
	// masterController.registerCallback(masterController.detectNighttime)
	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "regelverk/ticker/timeofday" {
			masterController.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
		}
	})

	// Livingroom
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "occupancy")
		},
		func(val any) (string, bool) { return "livingroomPresence", val.(bool) },
		nil,
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "battery")
		},
		func(val any) (string, bool) { return "livingroomPresenceBatteryLow", val.(float64) < 20 },
		func(val any) (string, float64) { return "livingroomPresenceBattery", val.(float64) },
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "illuminance_lux")
		},
		nil,
		func(val any) (string, float64) { return "livingroomPresenceIlluminanceLux", val.(float64) },
	))

	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-floorlamp", "state")
		},
		func(val any) (string, bool) { return "livingroomFloorlamp", val.(string) == "ON" },
		nil,
	))
	//masterController.registerEventCallback(masterController.detectTVPower)
	// masterController.registerEventCallback(func(ev MQTTEvent) {
	// 	if ev.Topic == "regelverk/state/tvpower" {
	// 		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
	// 		if err != nil {
	// 			slog.Error("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
	// 		}
	// 		masterController.stateValueMap.setState("tvPower", tvPower)
	// 	}
	// })

	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "rotel/state", "state")
		},
		func(val any) (string, bool) { return "rotelActive", val.(string) == "" },
		nil,
	))

	// Kitchen
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/kitchen-amp", "state")
		},
		func(val any) (string, bool) { return "kitchenAmpPower", val.(string) == "ON" },
		nil,
	))

	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/kitchen-computer", "state")
		},
		func(val any) (string, bool) { return "kitchenComputerPower", val.(string) == "ON" },
		nil,
	))

	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "kitchen/pulseaudio/state" {
			var pulseaudioState pulseaudiomqtt.PulseAudioState
			err := json.Unmarshal(ev.Payload.([]byte), &pulseaudioState)
			if err != nil {
				slog.Error("Could not parse payload", "topic", "kitchen/pulseaudio/state", "error", err)
				return
			}
			masterController.stateValueMap.setState("kitchenAudioPlaying", pulseaudioState.DefaultSink.State == 0)
		}
	})

	// Bedroom
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/blinds-bedroom", "position")
		},
		func(val any) (string, bool) { return "bedroomBlindsOpen", val.(float64) > 50 },
		func(val any) (string, float64) { return "bedroomBlindsPosition", val.(float64) },
	))

	// Balcony door
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/balcony-door", "contact")
		},
		func(val any) (string, bool) { return "balconyDoorOpen", !val.(bool) },
		nil,
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/balcony-door", "battery")
		},
		func(val any) (string, bool) { return "balconyDoorBatteryLow", val.(float64) < 20 },
		func(val any) (string, float64) { return "balconyDoorBattery", val.(float64) },
	))

	// MPD
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "mpd/status", "state")
		},
		func(val any) (string, bool) { return "mpdPlay", val.(string) == "play" },
		nil,
	))

}

// func (l *MasterController) detectTVPower(ev MQTTEvent) {
// 	if ev.Topic == "regelverk/state/tvpower" {
// 		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
// 		if err != nil {
// 			slog.Error("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
// 		}
// 		l.stateValueMap.setState("tvPower", tvPower)
// 	}
// }

func (l *MasterController) detectCECState(ev MQTTEvent) {
	switch ev.Topic {

	// case "cec/command":
	case "cec/message/hex/rx":
		fallthrough
	case "cec/message/hex/tx":
		command := strings.ToUpper(string(ev.Payload.([]byte)))
		slog.Debug("CEC command", "command", command)
		switch command {
		case "01:90:00":
			fallthrough
		case "01:90:00:00:00":
			slog.Debug("TV power")
			l.stateValueMap.setState("tvPower", true)
		case "01:90:01":
			fallthrough
		case "01:90:01:00:00":
			slog.Debug("TV standby")
			l.stateValueMap.setState("tvPower", false)
		case "0F:82:00:00:00:00":
			fallthrough
		case "0F:82:00:00":
			slog.Debug("TV active source")
			l.stateValueMap.setState("tvSourceTvActive", true)
			l.stateValueMap.setState("tvSourceMediaflixActive", false)
			l.stateValueMap.setState("tvSourceChromecastActive", false)
			l.stateValueMap.setState("tvSourceBlurayActive", false)
		case "1F:82:40:00:00:00":
			fallthrough
		case "1F:82:40:00":
			slog.Debug("Mediaflix active source")
			l.stateValueMap.setState("tvSourceTvActive", false)
			l.stateValueMap.setState("tvSourceMediaflixActive", true)
			l.stateValueMap.setState("tvSourceChromecastActive", false)
			l.stateValueMap.setState("tvSourceBlurayActive", false)
		case "8F:82:30:00:00:00":
			fallthrough
		case "8F:82:30:00":
			slog.Debug("Chromecast active source")
			l.stateValueMap.setState("tvSourceTvActive", false)
			l.stateValueMap.setState("tvSourceMediaflixActive", false)
			l.stateValueMap.setState("tvSourceChromecastActive", true)
			l.stateValueMap.setState("tvSourceBlurayActive", false)
		case "4F:82:20:00:00:00":
			fallthrough
		case "4F:82:20:00":
			slog.Debug("Bluray active source")
			l.stateValueMap.setState("tvSourceTvActive", false)
			l.stateValueMap.setState("tvSourceMediaflixActive", false)
			l.stateValueMap.setState("tvSourceChromecastActive", false)
			l.stateValueMap.setState("tvSourceBlurayActive", true)
		case "0F:36":
			slog.Debug("TV requests standby")
		default:
			slog.Debug("CEC command not recognized", "command", command)
		}
	}
}
