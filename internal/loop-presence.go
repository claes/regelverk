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

type PresenceLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
}

func (l *PresenceLoop) Init(m *mqttMessageHandler, config Config) {
	slog.Debug("Initializing FSM")
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge()

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

func (l *PresenceLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Debug("Process event")
		l.stateMachineMQTTBridge.detectLivingroomFloorlampState(ev)
		l.stateMachineMQTTBridge.detectPhonePresent(ev)
		l.stateMachineMQTTBridge.detectLivingroomPresence(ev)
		l.stateMachineMQTTBridge.stateValueMap.LogState()
		slog.Debug("Fire event")
		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

		eventsToPublish := l.stateMachineMQTTBridge.eventsToPublish
		slog.Debug("Event fired", "state", l.stateMachineMQTTBridge.stateMachine.MustState())
		l.stateMachineMQTTBridge.eventsToPublish = []MQTTPublish{}
		return eventsToPublish
	} else {
		slog.Debug("Cannot process event: is not initialized")
		return []MQTTPublish{}
	}
}
