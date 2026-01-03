package regelverk

import (
	"encoding/json"
	"log/slog"
	"sync"

	pulsemqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	rotelmqtt "github.com/claes/mqtt-bridges/rotel-mqtt/lib"
	"github.com/claes/regelverk/internal/z2m"
)

// DeviceStateStore keeps cached device state that multiple controllers can read.
type DeviceStateStore struct {
	mu           sync.RWMutex
	rotelState   rotelmqtt.RotelState
	pulseState   pulsemqtt.PulseAudioState
	z2mDevices   z2m.Devices
	rotelUpdated chan struct{}
	pulseUpdated chan struct{}
	z2mUpdated   chan struct{}
}

func NewDeviceStateStore() *DeviceStateStore {
	return &DeviceStateStore{
		rotelUpdated: make(chan struct{}, 1),
		pulseUpdated: make(chan struct{}, 1),
		z2mUpdated:   make(chan struct{}, 1),
	}
}

func (s *DeviceStateStore) SetRotel(state rotelmqtt.RotelState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rotelState = state
	select {
	case s.rotelUpdated <- struct{}{}:
	default:
	}
}

func (s *DeviceStateStore) GetRotel() rotelmqtt.RotelState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rotelState
}

func (s *DeviceStateStore) SetPulse(state pulsemqtt.PulseAudioState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pulseState = state
	select {
	case s.pulseUpdated <- struct{}{}:
	default:
	}
}

func (s *DeviceStateStore) GetPulse() pulsemqtt.PulseAudioState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pulseState
}

func (s *DeviceStateStore) RotelUpdates() <-chan struct{} {
	return s.rotelUpdated
}

func (s *DeviceStateStore) PulseUpdates() <-chan struct{} {
	return s.pulseUpdated
}

func (s *DeviceStateStore) SetZ2MDevices(devices z2m.Devices) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.z2mDevices = devices
	select {
	case s.z2mUpdated <- struct{}{}:
	default:
	}
}

func (s *DeviceStateStore) GetZ2MDevices() z2m.Devices {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.z2mDevices
}

func (s *DeviceStateStore) Z2MUpdates() <-chan struct{} {
	return s.z2mUpdated
}

// UpdateFromEvent ingests known MQTT state topics into the shared cache.
// Returns true if the event was handled.
func (s *DeviceStateStore) UpdateFromEvent(ev MQTTEvent) bool {
	switch ev.Topic {
	case "rotel/state":
		var rotelState rotelmqtt.RotelState
		err := json.Unmarshal(ev.Payload.([]byte), &rotelState)
		if err != nil {
			slog.Error("Could not unmarshal rotel state", "rotelstate", ev.Payload)
			return false
		}
		s.SetRotel(rotelState)
		return true
	case "pulseaudio/state":
		var pulseState pulsemqtt.PulseAudioState
		err := json.Unmarshal(ev.Payload.([]byte), &pulseState)
		if err != nil {
			slog.Error("Could not unmarshal pulseaudio state", "pulseaudiostate", ev.Payload)
			return false
		}
		s.SetPulse(pulseState)
		return true
	case "zigbee2mqtt/bridge/devices":
		z2mDevices, err := z2m.UnmarshalDevices(ev.Payload.([]byte))
		if err != nil {
			slog.Error("Could not unmarshal zigbee devices", "topic", ev.Topic, "payload", ev.Payload, "error", err)
			return false
		}
		s.SetZ2MDevices(z2mDevices)
		return true
	default:
		return false
	}
}
