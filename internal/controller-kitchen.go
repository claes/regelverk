package regelverk

import (
	"context"
	"reflect"

	"github.com/qmuntal/stateless"
)

type KitchenController struct {
	BaseController
}

func (c *KitchenController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "kitchen-controller"
	c.masterController = masterController

	// var initialState tvState
	// if masterController.stateValueMap.requireTrue("tvpower") {
	// 	initialState = stateTvOn
	// } else if masterController.stateValueMap.requireFalse("tvpower") {
	// 	initialState = stateTvOff
	// } else {
	// 	return nil
	// }

	c.stateMachine = stateless.NewStateMachine(ampStateOff)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(ampStateOn).
		OnEntry(c.turnOnKitchenAmp).
		Permit("mqttEvent", ampStateOff, c.masterController.guardStateKitchenAmpOff)

	c.stateMachine.Configure(ampStateOff).
		OnEntry(c.turnOffKitchenAmp).
		Permit("mqttEvent", ampStateOn, c.masterController.guardStateKitchenAmpOn)

	c.isInitialized = true
	return nil
}

func (c *KitchenController) turnOnKitchenAmp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(kitchenAmpPowerOutput(true))
	return nil
}

func (c *KitchenController) turnOffKitchenAmp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(kitchenAmpPowerOutput(false))
	return nil
}
