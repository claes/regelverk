package regelverk

import (
	"context"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

//go:generate stringer -type=batteryState
type batteryState int

const (
	batteryGood batteryState = iota
	batteryPoor
)

func (t batteryState) ToInt() int {
	return int(t)
}

type BatteryReminderController struct {
	BaseController
	cancelFunc          context.CancelFunc
	Name                string
	StateBatteryPoorKey StateKey
	ReminderPeriod      time.Duration
	MaxReminders        int
	ReminderTopic       string
	ReminderPayload     string
}

func (c *BatteryReminderController) Initialize(masterController *MasterController) []MQTTPublish {
	c.masterController = masterController

	var initialState batteryState = batteryGood

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(batteryGood).
		Permit("mqttEvent", batteryPoor, c.masterController.requireTrueByKey(c.StateBatteryPoorKey))

	c.stateMachine.Configure(batteryPoor).
		OnEntry(c.startNotifyBatteryPoor).
		OnExit(c.stopNotifyBatteryPoor).
		Permit("mqttEvent", batteryGood, c.masterController.requireFalseByKey(c.StateBatteryPoorKey))

	c.SetInitialized()
	return nil
}

func (c *BatteryReminderController) startNotifyBatteryPoor(parentContext context.Context, _ ...any) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	var ctx context.Context
	ctx, c.cancelFunc = context.WithCancel(parentContext)

	go func() {
		ticker := time.NewTicker(c.ReminderPeriod)
		defer ticker.Stop()
		for i := 0; i < c.MaxReminders; i++ {
			select {
			case <-ticker.C:

				events := []MQTTPublish{
					{
						Topic:    c.ReminderTopic,
						Payload:  c.ReminderPayload,
						Qos:      2,
						Retained: false,
						Wait:     0 * time.Second,
					},
				}
				c.addEventsToPublish(events)
				i = i + 1
			case <-ctx.Done():
				return
			}
		}
		c.cancelFunc()
	}()
	return nil
}

func (c *BatteryReminderController) stopNotifyBatteryPoor(_ context.Context, _ ...any) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
	return nil
}
