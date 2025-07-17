package regelverk

import (
	"context"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

type doorState int

const (
	doorClosed doorState = iota
	doorOpen
	doorOpenLong
)

func (t doorState) ToInt() int {
	return int(t)
}

type DoorReminderController struct {
	BaseController
	cancelFunc      context.CancelFunc
	Name            string
	SensorName      string
	StateOpenKey    string // "freezerDoorOpen"
	OpenLongLimit   time.Duration
	ReminderPeriod  time.Duration
	MaxReminders    int
	ReminderTopic   string
	ReminderPayload string
}

func (c *DoorReminderController) Initialize(masterController *MasterController) []MQTTPublish {
	c.masterController = masterController

	// var initialState doorState
	// if masterController.stateValueMap.requireTrue(c.StateOpenKey) {
	// 	initialState = doorOpen
	// } else if masterController.stateValueMap.requireFalse(c.StateOpenKey) {
	// 	initialState = doorClosed
	// } else {
	// 	const maxBackoff = 128 * time.Second
	// 	if c.checkBackoff() {
	// 		c.extendBackoff(maxBackoff)
	// 		return []MQTTPublish{requestIkeaTretaktPower("zigbee2mqtt/livingroom-floorlamp/get")}
	// 	} else {
	// 		return nil
	// 	}
	// }

	var initialState doorState = doorClosed

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(doorClosed).
		Permit("mqttEvent", doorOpen, c.masterController.requireTrueByKey(c.StateOpenKey))

	c.stateMachine.Configure(doorOpen).
		OnEntry(c.requestBatteryStatus).
		Permit("mqttEvent", doorClosed, c.masterController.requireFalseByKey(c.StateOpenKey)).
		Permit("mqttEvent", doorOpenLong, c.masterController.requireTrueSinceByKey(c.StateOpenKey, c.OpenLongLimit))

	c.stateMachine.Configure(doorOpenLong).
		OnEntry(c.startNotifyDoorOpen).
		OnExit(c.stopNotifyDoorOpen).
		Permit("mqttEvent", doorClosed, c.masterController.requireFalseByKey(c.StateOpenKey))

	c.SetInitialized()
	return nil
}

func (c *DoorReminderController) startNotifyDoorOpen(parentContext context.Context, _ ...any) error {
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

func (c *DoorReminderController) stopNotifyDoorOpen(_ context.Context, _ ...any) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
	return nil
}

func (c *DoorReminderController) requestBatteryStatus(_ context.Context, _ ...any) error {
	c.addEventsToPublish(c.requestBatteryStatusOutput())
	return nil
}

func (c *DoorReminderController) requestBatteryStatusOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    `zigbee2mqtt/` + c.SensorName + `/get`,
			Payload:  `{"battery":""}`,
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}
