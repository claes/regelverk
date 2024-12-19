package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"time"

	pulsemqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	snapcastmqtt "github.com/claes/mqtt-bridges/snapcast-mqtt/lib"
	"github.com/qmuntal/stateless"
)

type snapcastState int

var topicStreamRe = regexp.MustCompile(`snapcast/stream/([^/]+)$`)
var topicClientRe = regexp.MustCompile(`snapcast/client/([^/]+)$`)
var topicGroupRe = regexp.MustCompile(`snapcast/group/([^/]+)$`)

const (
	stateSnapcastOn snapcastState = iota
	stateSnapcastOff
)

type SnapcastController struct {
	BaseController
	snapcastServerState snapcastmqtt.SnapcastServer
	pulseAudioState     pulsemqtt.PulseAudioState
}

func (c *SnapcastController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "snapcast-controller"
	c.masterController = masterController

	// var initialState snapcastState
	// if masterController.stateValueMap.requireTrue("tvpower") {
	// 	initialState = stateSnapcastOff
	// } else if masterController.stateValueMap.requireFalse("tvpower") {
	// 	initialState = stateSnapcastOn
	// } else {
	// 	return nil
	// }

	var initialState snapcastState = stateSnapcastOff
	c.stateMachine = stateless.NewStateMachine(initialState) // can this be reliable determined early on?
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateSnapcastOn).
		OnEntry(c.turnOnSnapcast).
		Permit("mqttEvent", stateSnapcastOff, masterController.guardStateSnapcastOff)

	c.stateMachine.Configure(stateSnapcastOff).
		OnEntry(c.turnOffSnapcast).
		Permit("mqttEvent", stateSnapcastOn, masterController.guardStateSnapcastOn)

	c.eventHandlers = append(c.eventHandlers, c.customProcessEvent)

	c.isInitialized = true
	return nil
}

func (c *SnapcastController) customProcessEvent(ev MQTTEvent) []MQTTPublish {
	c.parseSnapcastClient(ev)
	c.parseSnapcastGroup(ev)
	c.parseSnapcastStream(ev)
	c.parsePulseaudio(ev)
	c.detectRemoteToggle(ev)
	return nil
}

func (c *SnapcastController) turnOnSnapcast(_ context.Context, _ ...any) error {
	for _, sinkInput := range c.pulseAudioState.SinkInputs {
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			events := snapcastOnOutput(sinkInput.SinkInputIndex, "Snapcast")
			c.addEventsToPublish(events)
		}
	}
	return nil
}

func (c *SnapcastController) turnOffSnapcast(_ context.Context, _ ...any) error {
	for _, sinkInput := range c.pulseAudioState.SinkInputs {
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			events := snapcastOffOutput(sinkInput.SinkInputIndex, "alsa_output.pci-0000_00_0e.0.hdmi-stereo")
			c.addEventsToPublish(events)
		}
	}
	return nil
}

func (c *SnapcastController) detectRemoteToggle(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/media_remote" {
		m := parseJSONPayload(ev)
		if m == nil {
			return
		}
		val, exists := m["action"]
		if !exists || val == nil {
			return
		}
		action := val.(string)
		if action == "arrow_right_click" {
			c.masterController.stateValueMap.setState("snapcast", true)
		} else if action == "arrow_left_click" {
			c.masterController.stateValueMap.setState("snapcast", false)
		}
	}
}

func (c *SnapcastController) parseSnapcastStream(ev MQTTEvent) {
	matches := topicStreamRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		streamID := matches[1]
		var snapcastStream snapcastmqtt.SnapcastStream
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastStream)
		if err != nil {
			slog.Error("Could not parse payload for snapcast stream", "error", err, "topic", ev.Topic)
		}
		if c.snapcastServerState.Streams == nil {
			c.snapcastServerState.Streams = make(map[string]snapcastmqtt.SnapcastStream)
		}
		c.snapcastServerState.Streams[streamID] = snapcastStream
	}
}

func (c *SnapcastController) parseSnapcastClient(ev MQTTEvent) {
	matches := topicClientRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		clientID := matches[1]
		var snapcastClient snapcastmqtt.SnapcastClient
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastClient)
		if err != nil {
			slog.Error("Could not parse payload for snapcast client", "error", err, "topic", ev.Topic)
		}
		if c.snapcastServerState.Clients == nil {
			c.snapcastServerState.Clients = make(map[string]snapcastmqtt.SnapcastClient)
		}
		c.snapcastServerState.Clients[clientID] = snapcastClient
	}
}

func (c *SnapcastController) parseSnapcastGroup(ev MQTTEvent) {
	matches := topicGroupRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		groupID := matches[1]
		var snapcastGroup snapcastmqtt.SnapcastGroup
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastGroup)
		if err != nil {
			slog.Error("Could not parse payload for snapcast group", "error", err, "topic", ev.Topic)
		}
		if c.snapcastServerState.Groups == nil {
			c.snapcastServerState.Groups = make(map[string]snapcastmqtt.SnapcastGroup)
		}
		c.snapcastServerState.Groups[groupID] = snapcastGroup
	}
}

func (c *SnapcastController) parsePulseaudio(ev MQTTEvent) {
	if ev.Topic == "pulseaudio/state" {
		err := json.Unmarshal(ev.Payload.([]byte), &c.pulseAudioState)
		if err != nil {
			slog.Error("Could not parse payload for topic", "topic", ev.Topic, "error", err)
		}
	}
}

func snapcastOnOutput(sinkInputIndex uint32, sinkName string) []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "pulseaudio/sinkinput/req",
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
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:iec958-stereo+input:analog-stereo",
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

func snapcastOffOutput(sinkInputIndex uint32, sinkName string) []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:hdmi-stereo", //TODO: switch this in advance?
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "pulseaudio/sinkinput/req",
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
