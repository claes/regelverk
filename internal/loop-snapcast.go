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

// New approach
// Snapcast clietn with applicatio KodiDriver / process.binary "kodi.bin"
// Sink Snapcast (#1)
// Snapclient with -s = 1 argument (PulseAudio)

// SINK INPUTS / Pavucontrol "Playback"
// Sink input #62 : Sink 1  Client: 128    kodi audio stream KodiSink Binary kodi.bin
// Sink input #71 : Sink 25 Client: 150    ALSA playback / ALSA plug-in [snapclient] Binary "snapclient"

// SINKS / Pavucontrol "Output Devices"
// Sink #1  Snapcast

// Sink #25 alsooutput_pci-...
//        Active port: hdmi-output-0
//    or               iec958-stereo-output

// Changing something renumbers the sinks etc so need to be done in quick order
// So: general approach
// 1) Redirect the KODI sink input to user the Snapcast sink
// 2) ALSA Plugin for snapclient should go to appropriate out (via HDMI or soundcard optical)
// Make sure snapclient uses the default PA sink:  /bin/snapclient -s 1  -h 127.0.0.1 --hostID livingroom
// 3) Set stream for Snapcast group to Pulseaudio
// 4) Set appropriate amp source according to ALSO Plugin (HDMI OPT or soundcard OPT)
/*
	0) determine sink input index for Kodi etc through pulseaudiostate

	1) pulseaudio/sinkinput/req  '{ "Command": "movesink", "SinkInputIndex": 168, "SinkName": "Snapcast" }'


	3) 	Topic:    "snapcast/client/livingroom/stream/set",
		Payload:  "pulseaudio",
	4) 	Topic:    "rotel/command/send",
		Payload:  "opt2!",


	pulseaudio/sinkinput/req  '{ "Command": "movesink", "SinkInputIndex": 168, "SinkName": "alsa_output.pci-0000_00_1f.3.analog-stereo" }'
				Topic:    "zigbee2mqtt/ikea_uttag/set",
				Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
				Qos:      2,
				Retained: false,
					Wait:     0 * time.Second,
*/

func snapcastOnOutputTmp(sinkInputIndex uint32) []MQTTPublish {

	slog.Info("SETTING SNAPCAST OUTPUT", "sinkInputIndex", sinkInputIndex)
	result := []MQTTPublish{
		{
			Topic:    "pulseaudio/sinkinput/req",
			Payload:  fmt.Sprintf(`{ "Command": "movesink", "SinkInputIndex": %d, "SinkName": "Snapcast" }`, sinkInputIndex),
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

func snapcastOffOutputTmp(sinkInputIndex uint32, sinkName string) []MQTTPublish {

	slog.Info("SETTING SNAPCAST OUTPUT", "sinkInputIndex", sinkInputIndex)
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
			Payload:  "default",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt1!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
	return result
}

func (l *SnapcastLoop) turnOnSnapcastTmp(_ context.Context, _ ...any) error {

	// kodiBinary := "kodi.bin"
	// kodiApplicationName := "KodiSink"
	// kodiMediaName := "kodi audio stream"

	// ChromeBinary := "chrome"
	// ChromeApplicationName := "Google Chrome"
	// ChromeMediaName := "Playback"

	slog.Info("Attempting to move sink", "len", len(l.pulseAudioState.SinkInputs))
	found := false
	for _, sinkInput := range l.pulseAudioState.SinkInputs {
		slog.Info("Looping over sink input", "sinkIndex", sinkInput.SinkIndex, "props", sinkInput.Properties)
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			slog.Info("Sink input match", "sinkIndex", sinkInput.SinkIndex)
			l.stateMachineMQTTBridge.addEventsToPublish(snapcastOnOutputTmp(sinkInput.SinkInputIndex))
			found = true
		}
	}
	if found {
		return nil
	} else {
		return fmt.Errorf("Could not find sink input to turn on Snapcast for")
	}
}

func (l *SnapcastLoop) turnOffSnapcastTmp(_ context.Context, _ ...any) error {

	slog.Info("Attempting to move sink", "len", len(l.pulseAudioState.SinkInputs))
	found := false
	for _, sinkInput := range l.pulseAudioState.SinkInputs {
		slog.Info("Looping over sink input", "sinkIndex", sinkInput.SinkIndex, "props", sinkInput.Properties)
		if sinkInput.Properties["application.process.binary"] == "kodi.bin" {
			slog.Info("Sink input match", "sinkIndex", sinkInput.SinkIndex)
			events := snapcastOffOutputTmp(sinkInput.SinkInputIndex, "alsa_output.pci-0000_00_0e.0.hdmi-stereo")
			l.stateMachineMQTTBridge.addEventsToPublish(events)
			found = true
		}
	}
	if found {
		return nil
	} else {
		return fmt.Errorf("Could not find sink input to turn on Snapcast for")
	}
}

/*
type MoveSinkInput struct {
	SinkInputIndex uint32
	DeviceIndex    uint32 -1
	DeviceName     string //sink name
}

func (c *PulseClient) MoveSinkInput(sinkInputIndex uint32, deviceName string) error {
	err := c.protoClient.Request(&proto.MoveSinkInput{SinkInputIndex: sinkInputIndex, DeviceIndex: proto.Undefined, DeviceName: deviceName}, nil)
	if err != nil {
		return err
	}
	return nil
}


*/

// PA:
// GetClientInfo * not existing
// GetClientInfoReply
// GetClientInfoList
// GetClientInfoListReply

// GetSinkInfo * komplettera
// GetSinkInfoList *
// GetSinkInfoReply *
// GetSinkInfoListReply *

// GetSinkInputInfo * not existing
// GetSinkInputInfoList
// GetSinkInputInfoReply
// GetSinkInputInfoListReply
// Identify using client info

func (l *SnapcastLoop) Init(m *mqttMessageHandler, config Config) {
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("snapcast")

	// Variants:
	// stream mediaflix pulseaudio to snapcast
	// stream mediaflix mpd to snapcast
	// stream mediaflix gmediarender to snapcast
	// inspect currently playing source using for example
	// pactl list sink-inputs
	// inspect

	sm := stateless.NewStateMachine(stateSnapcastOff) // can this be reliable determined early on? probably not
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(stateSnapcastOn).
		OnEntry(l.turnOnSnapcastTmp).
		Permit("mqttEvent", stateSnapcastOff, l.stateMachineMQTTBridge.guardStateSnapcastOff)
		// Set mediaflix pulseaudio port/card/profile (?) to HDMI or diabled to avoid conflict with snapclient
		// Something like command
		// pactl set-card-profile alsa_card.pci-0000_00_0e.0  output:hdmi-stereo
		// or
		// pactl set-card-profile alsa_card.pci-0000_00_0e.0  off
		// pulseaudio/cardprofile/0/set  output:hdmi-stereo
		// or
		// pulseaudio/cardprofile/0/set  off

		// Set mediaflix pulseaudio sink = Snapcast
		// pulseaudio/sink/default/set alsa_output.pci-0000_00_0e.0.hdmi-stereo
		// or
		// pulseaudio/sink/default/set Snapcast

		// Set snapcast server stream=pulseaudio

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

		l.parseSnapcastClient(ev)
		l.parseSnapcastGroup(ev)
		l.parseSnapcastStream(ev)
		l.parsePulseaudio(ev)
		l.foo(ev)

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
