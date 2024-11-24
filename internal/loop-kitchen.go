package regelverk

import (
	"log/slog"
	"reflect"

	"github.com/qmuntal/stateless"
)

type ampState int

const (
	ampStateOn ampState = iota
	ampStateOff
)

type KitchenLoop struct {
	statusLoop
	stateMachineMQTTBridge StateMachineMQTTBridge
	isInitialized          bool
}

func (l *KitchenLoop) Init(m *mqttMessageHandler, config Config) {
	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("kitchenamp")

	sm := stateless.NewStateMachine(ampStateOff)
	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	sm.Configure(ampStateOn).
		OnEntry(l.stateMachineMQTTBridge.turnOnTvAppliances).
		Permit("mqttEvent", ampStateOff, l.stateMachineMQTTBridge.guardStateKitchenAmpOff)

	sm.Configure(ampStateOff).
		OnEntry(l.stateMachineMQTTBridge.turnOffTvAppliances).
		Permit("mqttEvent", ampStateOn, l.stateMachineMQTTBridge.guardStateKitchenAmpOn)

	l.stateMachineMQTTBridge.stateMachine = sm
	l.isInitialized = true
	slog.Debug("FSM initialized")
}

func (l *KitchenLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	if l.isInitialized {
		slog.Debug("Process event")
		l.stateMachineMQTTBridge.detectKitchenAmpPower(ev)
		l.stateMachineMQTTBridge.detectKitchenAudioPlaying(ev)

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
