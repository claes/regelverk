package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"time"

	pulsemqtt "github.com/claes/pulseaudio-mqtt/lib"
	snapcastmqtt "github.com/claes/snapcast-mqtt/lib"
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

type SnapcastLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
	snapcastServerState    snapcastmqtt.SnapcastServer
	pulseAudioState        pulsemqtt.PulseAudioState
}

func snapcastOnOutputTmp(sinkInputIndex uint32, sinkName string) []MQTTPublish {
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

func snapcastOffOutputTmp(sinkInputIndex uint32, sinkName string) []MQTTPublish {
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

func (l *SnapcastLoop) turnOnSnapcastTmp(_ context.Context, _ ...any) error {
	for _, sinkInput := range l.pulseAudioState.SinkInputs {
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			events := snapcastOnOutputTmp(sinkInput.SinkInputIndex, "Snapcast")
			l.stateMachineMQTTBridge.addEventsToPublish(events)
		}
	}
	return nil
}

func (l *SnapcastLoop) turnOffSnapcastTmp(_ context.Context, _ ...any) error {
	for _, sinkInput := range l.pulseAudioState.SinkInputs {
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			events := snapcastOffOutputTmp(sinkInput.SinkInputIndex, "alsa_output.pci-0000_00_0e.0.hdmi-stereo")
			l.stateMachineMQTTBridge.addEventsToPublish(events)
		}
	}
	return nil
}

func (l *SnapcastLoop) Init(m *MQTTMessageHandler, config Config) {
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("snapcast")
	sm := stateless.NewStateMachine(stateSnapcastOff) // can this be reliable determined early on? probably not
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(stateSnapcastOn).
		OnEntry(l.turnOnSnapcastTmp).
		Permit("mqttEvent", stateSnapcastOff, l.stateMachineMQTTBridge.guardStateSnapcastOff)

	sm.Configure(stateSnapcastOff).
		OnEntry(l.turnOffSnapcastTmp).
		Permit("mqttEvent", stateSnapcastOn, l.stateMachineMQTTBridge.guardStateSnapcastOn)
	l.stateMachineMQTTBridge.stateMachine = sm
	l.isInitialized = true
	slog.Debug("FSM initialized")

	slog.Debug("Initialized Snapcast SM")

}

func (l *SnapcastLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Debug("Process event", "topic", ev.Topic)

		// Not strictly needed
		l.parseSnapcastClient(ev)
		l.parseSnapcastGroup(ev)
		l.parseSnapcastStream(ev)

		l.parsePulseaudio(ev)
		l.foo(ev)
		l.remoteToggle(ev)

		l.stateMachineMQTTBridge.stateValueMap.LogState()
		slog.Debug("Fire event")
		beforeState := l.stateMachineMQTTBridge.stateMachine.MustState()
		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.stateMachineMQTTBridge.getAndResetEventsToPublish()
		slog.Info("Event fired", "topic", ev.Topic, "fsm", l.stateMachineMQTTBridge.name, "beforeState", beforeState,
			"afterState", l.stateMachineMQTTBridge.stateMachine.MustState())
		return eventsToPublish
	} else {
		slog.Debug("Cannot process event: is not initialized")
		return []MQTTPublish{}
	}
}

func (l *SnapcastLoop) parseSnapcastStream(ev MQTTEvent) {
	matches := topicStreamRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		streamID := matches[1]
		var snapcastStream snapcastmqtt.SnapcastStream
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastStream)
		if err != nil {
			slog.Error("Could not parse payload for snapcast stream", "error", err, "topic", ev.Topic)
		}
		if l.snapcastServerState.Streams == nil {
			l.snapcastServerState.Streams = make(map[string]snapcastmqtt.SnapcastStream)
		}
		l.snapcastServerState.Streams[streamID] = snapcastStream
	}
}

func (l *SnapcastLoop) parseSnapcastClient(ev MQTTEvent) {
	matches := topicClientRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		clientID := matches[1]
		var snapcastClient snapcastmqtt.SnapcastClient
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastClient)
		if err != nil {
			slog.Error("Could not parse payload for snapcast client", "error", err, "topic", ev.Topic)
		}
		if l.snapcastServerState.Clients == nil {
			l.snapcastServerState.Clients = make(map[string]snapcastmqtt.SnapcastClient)
		}
		l.snapcastServerState.Clients[clientID] = snapcastClient
	}
}

func (l *SnapcastLoop) parseSnapcastGroup(ev MQTTEvent) {
	matches := topicGroupRe.FindStringSubmatch(ev.Topic)
	if len(matches) == 2 {
		groupID := matches[1]
		var snapcastGroup snapcastmqtt.SnapcastGroup
		err := json.Unmarshal(ev.Payload.([]byte), &snapcastGroup)
		if err != nil {
			slog.Error("Could not parse payload for snapcast group", "error", err, "topic", ev.Topic)
		}
		if l.snapcastServerState.Groups == nil {
			l.snapcastServerState.Groups = make(map[string]snapcastmqtt.SnapcastGroup)
		}
		l.snapcastServerState.Groups[groupID] = snapcastGroup
	}
}

func (l *SnapcastLoop) parsePulseaudio(ev MQTTEvent) {
	if ev.Topic == "pulseaudio/state" {
		err := json.Unmarshal(ev.Payload.([]byte), &l.pulseAudioState)
		if err != nil {
			slog.Error("Could not parse payload for topic", "topic", ev.Topic, "error", err)
		}
	}
}

func (l *SnapcastLoop) foo(ev MQTTEvent) {
	if ev.Topic == "foo" {
		snapcast, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Info("Could not parse payload", "topic", "foo", "error", err)
		}
		l.stateMachineMQTTBridge.stateValueMap.setState("snapcast", snapcast)
	}
}

func (l *SnapcastLoop) remoteToggle(ev MQTTEvent) {
	if ev.Topic == "zigbee2mqtt/media_remote" {
		m := parseJSONPayload(ev)
		action := m["action"].(string)
		if action == "arrow_right_click" {
			l.stateMachineMQTTBridge.stateValueMap.setState("snapcast", true)
		} else if action == "arrow_left_click" {
			l.stateMachineMQTTBridge.stateValueMap.setState("snapcast", false)
		}
	}
}
