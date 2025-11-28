package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	pulsemqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	"github.com/qmuntal/stateless"
)

type kitchenAudioState int

const (
	stateKitchenAudioLocal kitchenAudioState = iota
	stateKitchenAudioRemote
)

const (
	localSink  string = "alsa_output.usb-GeneralPlus_USB_Audio_Device-00.analog-stereo"
	remoteSink string = "tunnel.mediaflix.local.Snapcast"
)

func (t kitchenAudioState) ToInt() int {
	return int(t)
}

type KitchenAudioController struct {
	BaseController
	pulseAudioState pulsemqtt.PulseAudioState
}

func (c *KitchenAudioController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "kitchenaudio"
	c.masterController = masterController

	var initialState kitchenAudioState
	currentSink := c.getCurrentSink()
	if currentSink == localSink {
		initialState = stateKitchenAudioLocal
	} else if currentSink == remoteSink {
		initialState = stateKitchenAudioRemote
	} else {
		const maxBackoff = 128 * time.Second
		if c.checkBackoff() {
			c.extendBackoff(maxBackoff)
			return []MQTTPublish{
				{
					Topic:    "kitchen/pulseaudio/initialize",
					Payload:  `init`,
					Qos:      2,
					Retained: false,
				},
			}
		} else {
			return nil
		}
	}

	c.stateMachine = stateless.NewStateMachine(initialState) // can this be reliable determined early on?
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateKitchenAudioLocal).
		OnEntry(c.turnOnLocal).
		Permit("mqttEvent", stateKitchenAudioRemote, masterController.guardKitchenAudioRemote)

	c.stateMachine.Configure(stateKitchenAudioRemote).
		OnEntry(c.turnOnRemote).
		Permit("mqttEvent", stateKitchenAudioLocal, masterController.guardKitchenAudioLocal)

	c.eventHandlers = append(c.eventHandlers, c.customProcessEvent)

	c.SetInitialized()
	return nil
}

func (c *KitchenAudioController) getCurrentSink() string {
	currentSink := ""
	if c.pulseAudioState.SinkInputs != nil && c.pulseAudioState.Sinks != nil {
		for _, sinkInput := range c.pulseAudioState.SinkInputs {
			if sinkInput.Properties != nil {
				if iconName, exists := sinkInput.Properties["media.icon_name"]; exists && iconName == "audio-card-bluetooth" {
					for _, sink := range c.pulseAudioState.Sinks {
						if sinkInput.SinkIndex == sink.SinkIndex {
							if sink.Id != "" {
								currentSink = sink.Id
								break
							}
						}
					}
				}
			}
		}
	}
	return currentSink
}

func (c *KitchenAudioController) customProcessEvent(ev MQTTEvent) []MQTTPublish {
	c.parsePulseaudio(ev)
	c.handleMediaRemoteEvents(ev)
	return nil
}

func (l *MasterController) guardKitchenAudioLocal(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("kitchenaudiolocal")
	return check
}

func (l *MasterController) guardKitchenAudioRemote(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("kitchenaudiolocal")
	return check
}

func (c *KitchenAudioController) turnOnRemote(_ context.Context, _ ...any) error {
	for _, sinkInput := range c.pulseAudioState.SinkInputs {
		if sinkInput.Properties["media.icon_name"] == "audio-card-bluetooth" {
			events := pulseaudioRemoteOutput(sinkInput.SinkInputIndex, remoteSink)
			c.addEventsToPublish(events)
		}
	}
	return nil
}

func (c *KitchenAudioController) turnOnLocal(_ context.Context, _ ...any) error {
	for _, sinkInput := range c.pulseAudioState.SinkInputs {
		if sinkInput.Properties["media.icon_name"] == "audio-card-bluetooth" {
			events := pulseaudioLocalOutput(sinkInput.SinkInputIndex, localSink)
			c.addEventsToPublish(events)
		}
	}
	return nil
}

func (c *KitchenAudioController) handleMediaRemoteEvents(ev MQTTEvent) []MQTTPublish {
	if ev.Topic == "zigbee2mqtt/media_remote_kitchen" {
		m := parseJSONPayload(ev)
		if m == nil {
			return nil
		}
		val, exists := m["action"]
		if !exists || val == nil {
			return nil
		}
		switch val {
		case "dots_2_double_press":
			c.masterController.stateValueMap.setState("kitchenaudiolocal", false)
		case "dots_2_long_press":
			c.masterController.stateValueMap.setState("kitchenaudiolocal", true)
		}
	}
	return nil
}

func (c *KitchenAudioController) parsePulseaudio(ev MQTTEvent) {
	if ev.Topic == "kitchen/pulseaudio/state" {
		err := json.Unmarshal(ev.Payload.([]byte), &c.pulseAudioState)
		if err != nil {
			slog.Error("Could not parse payload for topic", "topic", ev.Topic, "error", err)
		}
	}
}

func pulseaudioLocalOutput(sinkInputIndex uint32, sinkName string) []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "kitchen/pulseaudio/sinkinput/req",
			Payload:  fmt.Sprintf(`{ "Command": "movesink", "SinkInputIndex": %d, "SinkName": "%s" }`, sinkInputIndex, sinkName),
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "snapcast/client/livingroom/stream/set",
			Payload:  "pulseaudio",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt2!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
	return result
}

func pulseaudioRemoteOutput(sinkInputIndex uint32, sinkName string) []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:hdmi-stereo", //TODO: switch this in advance?
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "kitchen/pulseaudio/sinkinput/req",
			Payload:  fmt.Sprintf(`{ "Command": "movesink", "SinkInputIndex": %d, "SinkName": "%s" }`, sinkInputIndex, sinkName),
			Qos:      2,
			Retained: false,
			Wait:     2 * time.Second,
		},
		{
			Topic:    "snapcast/client/livingroom/stream/set",
			Payload:  "default",
			Qos:      2,
			Retained: false,
			Wait:     2 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt1!",
			Qos:      2,
			Retained: false,
			Wait:     2 * time.Second,
		},
	}
	return result
}
