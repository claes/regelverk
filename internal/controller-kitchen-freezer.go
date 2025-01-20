package regelverk

import (
	"reflect"

	"github.com/qmuntal/stateless"
)

type kitchenFreezerDoorState int

const (
	kitchenFreezerDoorClosed kitchenFreezerDoorState = iota
	kitchenFreezerDoorOpen
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
		//OnEntry(c.turnOnKitchenAmp).
		Permit("mqttEvent", kitchenFreezerDoorOpen, c.masterController.guardStateFreezerDoorOpen)

	c.stateMachine.Configure(kitchenFreezerDoorOpen).
		//OnEntry(c.turnOffKitchenAmp).
		Permit("mqttEvent", kitchenFreezerDoorClosed, c.masterController.guardStateFreezerDoorClosed)

	c.SetInitialized()
	return nil
}
