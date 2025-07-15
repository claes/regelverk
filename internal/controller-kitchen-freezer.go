package regelverk

import (
	"context"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

type kitchenFreezerDoorState int

const (
	kitchenFreezerDoorClosed kitchenFreezerDoorState = iota
	kitchenFreezerDoorOpen
	kitchenFreezerDoorOpenLong
)

func (t kitchenFreezerDoorState) ToInt() int {
	return int(t)
}

type KitchenFreezerDoorController struct {
	BaseController
	cancelFunc context.CancelFunc
}

func (c *KitchenFreezerDoorController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "kitchenfreezerdoor"
	c.masterController = masterController

	var initialState kitchenFreezerDoorState = kitchenFreezerDoorClosed

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(kitchenFreezerDoorClosed).
		Permit("mqttEvent", kitchenFreezerDoorOpen, c.masterController.guardStateFreezerDoorOpen)

	c.stateMachine.Configure(kitchenFreezerDoorOpen).
		Permit("mqttEvent", kitchenFreezerDoorClosed, c.masterController.guardStateFreezerDoorClosed).
		Permit("mqttEvent", kitchenFreezerDoorOpenLong, c.masterController.guardStateFreezerDoorOpenLong)

	c.stateMachine.Configure(kitchenFreezerDoorOpenLong).
		OnEntry(c.startNotifyDoorOpen).
		OnExit(c.stopNotifyDoorOpen).
		Permit("mqttEvent", kitchenFreezerDoorClosed, c.masterController.guardStateFreezerDoorClosed)

	c.SetInitialized()
	return nil
}

func (c *KitchenFreezerDoorController) startNotifyDoorOpen(parentContext context.Context, _ ...any) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	var ctx context.Context
	ctx, c.cancelFunc = context.WithCancel(parentContext)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for i := 0; i < 20; i++ {
			select {
			case <-ticker.C:

				events := []MQTTPublish{
					{
						Topic:    "kitchen/audio/play",
						Payload:  `embed://assets/ping.wav`,
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

func (c *KitchenFreezerDoorController) stopNotifyDoorOpen(_ context.Context, _ ...any) error {
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
	return nil
}
