package regelverk

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/VictoriaMetrics/metrics"
	"github.com/qmuntal/stateless"
)

type BaseController struct {
	name             string
	masterController *MasterController
	stateMachine     *stateless.StateMachine
	eventsToPublish  []MQTTPublish
	isInitialized    bool
	eventHandlers    []func(ev MQTTEvent) []MQTTPublish
	mu               sync.Mutex
}

func (c *BaseController) String() string {
	return c.name
}

func (c *BaseController) Lock() {
	c.mu.Lock()
}

func (c *BaseController) Unlock() {
	c.mu.Unlock()
}

func (c *BaseController) SetInitialized() {
	c.isInitialized = true

	if c.masterController.metricsConfig.CollectMetrics {
		stateInt, ok := c.stateMachine.MustState().(int)
		if ok {
			gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			gauge.Set(float64(stateInt))
			c.masterController.pushMetrics = true
		}
	}
}

func (c *BaseController) IsInitialized() bool {
	return c.isInitialized
}

func (c *BaseController) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	slog.Debug("Process event", "name", c.name)

	// In case special handling is needed that is not part of base processing
	// Under normal circumstances, state machine should be able to handle most
	for _, eventHandler := range c.eventHandlers {
		c.addEventsToPublish(eventHandler(ev))
	}

	beforeState := c.stateMachine.MustState()
	c.StateMachineFire("mqttEvent", ev)

	eventsToPublish := c.getAndResetEventsToPublish()
	afterState := c.stateMachine.MustState()
	slog.Debug("Event fired", "fsm", c.name, "topic", ev.Topic,
		"beforeState", beforeState,
		"afterState", afterState,
		"stateDiff", (beforeState != afterState),
		"eventsToPublish", (len(eventsToPublish) > 0),
		"noOfEventsToPublish", len(eventsToPublish))

	if c.masterController.metricsConfig.CollectDebugMetrics {
		triggerStr := createTriggerString(ev)
		if afterState != beforeState {
			if intState, ok := beforeState.(interface{ ToInt() int }); ok {
				beforeStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_change{controller="%s",trigger="%s",realm="%s"}`,
					c.name, triggerStr, c.masterController.metricsConfig.MetricsRealm), nil)
				beforeStateGauge.Set(float64(intState.ToInt()))
			} else {
				slog.Error("State does not implement ToInt", "state", afterState)
			}
		}
		if len(eventsToPublish) > 0 {
			counter := metrics.GetOrCreateCounter(fmt.Sprintf(`fsm_state_events{controller="%s",trigger="%s",realm="%s"}`,
				c.name, triggerStr, c.masterController.metricsConfig.MetricsRealm))
			counter.Add(len(eventsToPublish))
		}
		if intState, ok := beforeState.(interface{ ToInt() int }); ok {
			beforeStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_before{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			beforeStateGauge.Set(float64(intState.ToInt()))
		} else {
			slog.Error("State does not implement ToInt", "state", afterState)
		}
		if intState, ok := afterState.(interface{ ToInt() int }); ok {
			afterStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_after{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			afterStateGauge.Set(float64(intState.ToInt()))
		} else {
			slog.Error("State does not implement ToInt", "state", afterState)
		}
	}
	return eventsToPublish
}

func createTriggerString(trigger stateless.Trigger) string {
	var triggerStr string
	switch trigger.(type) {
	case string:
		triggerStr = trigger.(string)
	case MQTTEvent:
		ev := trigger.(MQTTEvent)
		triggerStr = ev.Topic
	default:
		triggerStr = "trigger"
	}
	return triggerStr
}
func (c *BaseController) StateMachineFire(trigger stateless.Trigger, args ...any) error {

	if c.masterController.metricsConfig.CollectDebugMetrics {
		counter := metrics.GetOrCreateCounter(fmt.Sprintf(`fsm_fire{controller="%s",realm="%s"}`,
			c.name, c.masterController.metricsConfig.MetricsRealm))
		counter.Inc()
	}
	return c.stateMachine.Fire(trigger, args...)
}

func (c *BaseController) addEventsToPublish(events []MQTTPublish) {
	c.eventsToPublish = append(c.eventsToPublish, events...)
}

func (c *BaseController) getAndResetEventsToPublish() []MQTTPublish {
	events := c.eventsToPublish
	c.eventsToPublish = []MQTTPublish{}
	return events
}
