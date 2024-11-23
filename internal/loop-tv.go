package regelverk

import (
	"log/slog"
	"reflect"

	"github.com/qmuntal/stateless"
)

type tvState int

const (
	stateTvOn tvState = iota
	stateTvOff
)

type TVLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
}

func (l *TVLoop) Init(m *mqttMessageHandler, config Config) {
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("tv")

	sm := stateless.NewStateMachine(stateTvOff) // can this be reliable determined early on? probably not
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(stateTvOn).
		OnEntry(l.stateMachineMQTTBridge.turnOnTvAppliances).
		Permit("mqttEvent", stateTvOff, l.stateMachineMQTTBridge.guardStateTvOff)

	sm.Configure(stateTvOff).
		OnEntry(l.stateMachineMQTTBridge.turnOffTvAppliances).
		Permit("mqttEvent", stateTvOn, l.stateMachineMQTTBridge.guardStateTvOn)

	l.stateMachineMQTTBridge.stateMachine = sm
	l.isInitialized = true
	slog.Debug("FSM initialized")
}

func (l *TVLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Debug("Process event")
		l.stateMachineMQTTBridge.detectTVPower(ev)
		l.stateMachineMQTTBridge.detectMPDPlay(ev)

		l.stateMachineMQTTBridge.stateValueMap.LogState()
		slog.Debug("Fire event")
		beforeState := l.stateMachineMQTTBridge.stateMachine.MustState()
		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.stateMachineMQTTBridge.eventsToPublish
		slog.Debug("Event fired", "fsm", l.stateMachineMQTTBridge.name, "beforeState", beforeState,
			"afterState", l.stateMachineMQTTBridge.stateMachine.MustState())
		l.stateMachineMQTTBridge.eventsToPublish = []MQTTPublish{}
		return eventsToPublish
	} else {
		slog.Debug("Cannot process event: is not initialized")
		return []MQTTPublish{}
	}
}
