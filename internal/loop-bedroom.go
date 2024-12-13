package regelverk

type blindsState int

const (
	bedroomBlindsStateOpen blindsState = iota
	bedroomBlindsStateClosed
)

// type BedroomLoop struct {
// 	statusLoop
// 	stateMachineMQTTBridge StateMachineMQTTBridge
// 	isInitialized          bool
// }

// func (l *BedroomLoop) Init(m *MQTTMessageHandler, config Config) {
// 	slog.Info("Init bedroom blinds")
// 	l.stateMachineMQTTBridge = CreateStateMachineMQTTBridge("bedroomblinds")

// 	sm := stateless.NewStateMachine(bedroomBlindsStateOpen)
// 	sm.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

// 	sm.Configure(bedroomBlindsStateOpen).
// 		OnEntry(l.stateMachineMQTTBridge.openBedroomBlinds).
// 		Permit("blindsdown", bedroomBlindsStateClosed).
// 		Ignore("blindsup").
// 		PermitReentry("timer").
// 		OnEntryFrom("timer", l.stateMachineMQTTBridge.refreshBedroomBlinds)

// 	sm.Configure(bedroomBlindsStateClosed).
// 		OnEntry(l.stateMachineMQTTBridge.closeBedroomBlinds).
// 		Permit("blindsup", bedroomBlindsStateOpen).
// 		Ignore("blindsdown").
// 		PermitReentry("timer").
// 		OnEntryFrom("timer", l.stateMachineMQTTBridge.refreshBedroomBlinds)

// 	// TODO - how to detect state from manual actions?
// 	// Any use of detectBedroomBlindsOpen /  guardStateBedroomBlindsOpen / Closed?

// 	go func() {
// 		for {
// 			now := time.Now()
// 			if now.Hour() == 9 && now.Minute() == 0 {
// 				l.stateMachineMQTTBridge.stateMachine.Fire("blindsup")
// 			} else if now.Hour() == 21 && now.Minute() == 0 {
// 				l.stateMachineMQTTBridge.stateMachine.Fire("blindsdown")
// 			}

// 			if now.Hour() == 8 && now.Minute() == 0 {
// 				l.stateMachineMQTTBridge.stateMachine.Fire("timer")
// 			} else if now.Hour() == 20 && now.Minute() == 0 {
// 				l.stateMachineMQTTBridge.stateMachine.Fire("timer")
// 			}

// 			time.Sleep(1 * time.Minute)
// 		}
// 	}()

// 	l.stateMachineMQTTBridge.stateMachine = sm
// 	l.isInitialized = true
// 	slog.Debug("FSM initialized")
// }

// var stateUpdaters []func(MQTTEvent)

// func (l *KitchenLoop) ProcessEventInternal(ev MQTTEvent) {
// 	l.stateMachineMQTTBridge.detectBedroomBlindsOpen(ev)
// }

// func (s *StateMachineMQTTBridge) foo(ev MQTTEvent) []MQTTPublish {
// 	slog.Debug("Process event")

// 	for _, stateUpdater := range stateUpdaters {
// 		stateUpdater(ev)
// 	}

// 	s.stateValueMap.LogState()
// 	slog.Debug("Fire event")
// 	beforeState := s.stateMachine.MustState()
// 	s.stateMachine.Fire("mqttEvent", ev)

// 	eventsToPublish := s.getAndResetEventsToPublish()
// 	slog.Debug("Event fired", "fsm", s.name, "beforeState", beforeState,
// 		"afterState", s.stateMachine.MustState())
// 	return eventsToPublish
// }

// func (l *BedroomLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
// 	if l.isInitialized {
// 		slog.Debug("Process event", "name", l.stateMachineMQTTBridge.name)
// 		l.stateMachineMQTTBridge.detectPhonePresent(ev)
// 		l.stateMachineMQTTBridge.detectBedroomBlindsOpen(ev)

// 		l.stateMachineMQTTBridge.stateValueMap.LogState()
// 		slog.Info("Fire event", "name", l.stateMachineMQTTBridge.name)
// 		beforeState := l.stateMachineMQTTBridge.stateMachine.MustState()
// 		l.stateMachineMQTTBridge.stateMachine.Fire("mqttEvent", ev)

// 		eventsToPublish := l.stateMachineMQTTBridge.getAndResetEventsToPublish()
// 		slog.Info("Event fired", "fsm", l.stateMachineMQTTBridge.name, "beforeState", beforeState,
// 			"afterState", l.stateMachineMQTTBridge.stateMachine.MustState())
// 		return eventsToPublish
// 	} else {
// 		slog.Debug("Cannot process event: is not initialized")
// 		return []MQTTPublish{}
// 	}
// }
