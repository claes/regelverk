package z2m

import (
	"encoding/json"
	"log/slog"
	"time"

	// "github.com/claes/regelverk/internal" // Commented out to avoid import cycle

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Created using command line like
// quicktype -l go -o parasoll.go parasoll.json
// where parasoll.json is from Zigbee2MQTT state tab of the device

// IKEA tretakt power plug

func UnmarshalIkeaTretakt(data []byte) (IkeaTretakt, error) {
	var r IkeaTretakt
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IkeaTretakt) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IkeaTretakt struct {
	LastSeen        time.Time         `json:"last_seen"`
	Linkquality     int64             `json:"linkquality"`
	State           string            `json:"state"`
	Update          IkeaTretaktUpdate `json:"update"`
	UpdateAvailable bool              `json:"update_available"`
	Identify        interface{}       `json:"identify"`
	PowerOnBehavior string            `json:"power_on_behavior"`
}

type IkeaTretaktUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

//IKEA Inspelning power plug

func UnmarshalIkeaInspelning(data []byte) (IkeaInspelning, error) {
	var r IkeaInspelning
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IkeaInspelning) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IkeaInspelning struct {
	Current         float64              `json:"current"`
	Energy          float64              `json:"energy"`
	LastSeen        time.Time            `json:"last_seen"`
	Linkquality     int64                `json:"linkquality"`
	Power           float64              `json:"power"`
	PowerOnBehavior string               `json:"power_on_behavior"`
	State           string               `json:"state"`
	Update          IkeaInspelningUpdate `json:"update"`
	UpdateAvailable bool                 `json:"update_available"`
	Voltage         float64              `json:"voltage"`
}

type IkeaInspelningUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

// TS011F Power plug

func UnmarshalTS011F(data []byte) (TS011F, error) {
	var r TS011F
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TS011F) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TS011F struct {
	Current           int64        `json:"current"`
	Energy            float64      `json:"energy"`
	LastSeen          time.Time    `json:"last_seen"`
	Linkquality       int64        `json:"linkquality"`
	Power             int64        `json:"power"`
	Update            TS011FUpdate `json:"update"`
	UpdateAvailable   bool         `json:"update_available"`
	Voltage           int64        `json:"voltage"`
	ChildLock         string       `json:"child_lock"`
	IndicatorMode     string       `json:"indicator_mode"`
	PowerOutageMemory string       `json:"power_outage_memory"`
	State             string       `json:"state"`
}

type TS011FUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

// IKEA Vindstyrka sensor

func UnmarshalIkeaVindstyrka(data []byte) (IkeaVindstyrka, error) {
	var r IkeaVindstyrka
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IkeaVindstyrka) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IkeaVindstyrka struct {
	Humidity    int64                `json:"humidity"`
	Linkquality int64                `json:"linkquality"`
	Pm25        int64                `json:"pm25"`
	Temperature int64                `json:"temperature"`
	Update      IkeaVindstyrkaUpdate `json:"update"`
	VocIndex    int64                `json:"voc_index"`
	Identify    interface{}          `json:"identify"`
}

type IkeaVindstyrkaUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

// IKEA Vallhorn presence sensor

func UnmarshalIkeaVallhorn(data []byte) (IkeaVallhorn, error) {
	var r IkeaVallhorn
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IkeaVallhorn) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IkeaVallhorn struct {
	Battery         int64              `json:"battery"`
	Illuminance     int64              `json:"illuminance"`
	LastSeen        time.Time          `json:"last_seen"`
	Linkquality     int64              `json:"linkquality"`
	Occupancy       bool               `json:"occupancy"`
	Update          IkeaVallhornUpdate `json:"update"`
	UpdateAvailable bool               `json:"update_available"`
}

type IkeaVallhornUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

// IKEA Parasoll door sensor

func UnmarshalIkeaParasoll(data []byte) (IkeaParasoll, error) {
	var r IkeaParasoll
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *IkeaParasoll) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type IkeaParasoll struct {
	Battery         int64              `json:"battery"`
	Contact         bool               `json:"contact"`
	LastSeen        time.Time          `json:"last_seen"`
	Linkquality     int64              `json:"linkquality"`
	Update          IkeaParasollUpdate `json:"update"`
	UpdateAvailable bool               `json:"update_available"`
	Voltage         int64              `json:"voltage"`
	Identify        interface{}        `json:"identify"`
}

type IkeaParasollUpdate struct {
	InstalledVersion int64  `json:"installed_version"`
	LatestVersion    int64  `json:"latest_version"`
	State            string `json:"state"`
}

// func getTypeUnmarshaller(topic string) func([]byte) (interface{}, error) {
// 	switch topic {
// 	case "zigbee2mqtt/bridge/devices":
// 		return func(data []byte) (interface{}, error) {
// 			return UnmarshalZ2MDevices(data)
// 		}
// 	}
// 	return nil
// }

func GetDeviceUnmarshaller(topic string) func([]byte) (interface{}, error) {
	switch topic {
	case "zigbee2mqtt/freezer-door", "zigbee2mqtt/fridge-door":
		return func(data []byte) (interface{}, error) {
			return UnmarshalIkeaParasoll(data)
		}
	case "zigbee2mqtt/kitchen-sink":
		return func(data []byte) (interface{}, error) {
			return UnmarshalIkeaInspelning(data)
		}
	case
		"zigbee2mqtt/livingroom-floorlamp",
		"zigbee2mqtt/kitchen-computer",
		"zigbee2mqtt/kitchen-amp":
		return func(data []byte) (interface{}, error) {
			return UnmarshalIkeaTretakt(data)
		}
	case "zigbee2mqtt/livingroom-presence":
		return func(data []byte) (interface{}, error) {
			return UnmarshalIkeaVallhorn(data)
		}
	case "zigbee2mqtt/tv-power":
		return func(data []byte) (interface{}, error) {
			return UnmarshalTS011F(data)
		}
	case "zigbee2mqtt/vindstyrka":
		return func(data []byte) (interface{}, error) {
			return UnmarshalIkeaVindstyrka(data)
		}
	}
	return nil
}

func InitZ2MDevices(_ mqtt.Client, m mqtt.Message) {
	switch m.Topic() {
	case "zigbee2mqtt/bridge/devices":
		z2mDevices, err := UnmarshalDevices(m.Payload())
		if err != nil {
			slog.Error("Could not unmarshal json state", "topic", m.Topic(), "payload", m.Payload(), "error", err)
		} else if z2mDevices != nil {
			slog.Info("Parsed Zigbee2MQTT devices", "noOfDevices", len(z2mDevices))
		}
	}
}
