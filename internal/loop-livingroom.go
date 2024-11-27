package regelverk

import (
	"log/slog"
	"reflect"

	"github.com/qmuntal/stateless"
)

const (
	stateLivingroomFloorlampOn  = "LampOn"
	stateLivingroomFloorlampOff = "LampOff"
)

type LivingroomLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
}

func (l *LivingroomLoop) Init(m *mqttMessageHandler, config Config) {
	slog.Debug("Initializing FSM")
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("livingroomLamp")

	sm := stateless.NewStateMachine(stateLivingroomFloorlampOff) // can this be reliable determined early on? probably not
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(stateLivingroomFloorlampOn).
		OnEntry(l.stateMachineMQTTBridge.turnOnLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOff, l.stateMachineMQTTBridge.guardTurnOffLivingroomLamp)

	sm.Configure(stateLivingroomFloorlampOff).
		OnEntry(l.stateMachineMQTTBridge.turnOffLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOn, l.stateMachineMQTTBridge.guardTurnOnLivingroomLamp)

	l.stateMachineMQTTBridge.stateMachine = sm
	l.isInitialized = true
	slog.Debug("FSM initialized")
}

func (l *LivingroomLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Debug("Process event")
		l.stateMachineMQTTBridge.detectLivingroomFloorlampState(ev)
		l.stateMachineMQTTBridge.detectPhonePresent(ev)
		l.stateMachineMQTTBridge.detectLivingroomPresence(ev)
		l.stateMachineMQTTBridge.detectNighttime(ev)

		l.stateMachineMQTTBridge.stateValueMap.LogState()
		slog.Debug("Fire event")
		beforeState := l.stateMachineMQTTBridge.stateMachine.MustState()
		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.stateMachineMQTTBridge.getAndResetEventsToPublish()
		slog.Debug("Event fired", "fsm", l.stateMachineMQTTBridge.name, "beforeState", beforeState,
			"afterState", l.stateMachineMQTTBridge.stateMachine.MustState())
		return eventsToPublish
	} else {
		slog.Debug("Cannot process event: is not initialized")
		return []MQTTPublish{}
	}
}
