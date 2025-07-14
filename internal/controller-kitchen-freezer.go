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
}

func (c *KitchenFreezerDoorController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "kitchenfreezerdoor"
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

var cancelFunc context.CancelFunc

func (c *KitchenFreezerDoorController) startNotifyDoorOpen(parentContext context.Context, _ ...any) error {
	var ctx context.Context
	ctx, cancelFunc = context.WithCancel(parentContext)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:

				events := []MQTTPublish{
					{
						Topic:    "/audio/play",
						Payload:  `embed://raven.mp3`,
						Qos:      2,
						Retained: false,
						Wait:     0 * time.Second,
					},
				}
				c.addEventsToPublish(events)

			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (c *KitchenFreezerDoorController) stopNotifyDoorOpen(_ context.Context, _ ...any) error {
	if cancelFunc != nil {
		cancelFunc()
		cancelFunc = nil
	}
	return nil
}
