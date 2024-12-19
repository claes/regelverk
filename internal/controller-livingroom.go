package regelverk

import (
	"context"
	"reflect"

	"github.com/qmuntal/stateless"
)

const (
	stateLivingroomFloorlampOn  = "LampOn"
	stateLivingroomFloorlampOff = "LampOff"
)

type LivingroomController struct {
	BaseController
}

func (c *LivingroomController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "livingroom-controller"
	c.masterController = masterController

	// var initialState tvState
	// if masterController.stateValueMap.requireTrue("tvpower") {
	// 	initialState = stateTvOn
	// } else if masterController.stateValueMap.requireFalse("tvpower") {
	// 	initialState = stateTvOff
	// } else {
	// 	return nil
	// }

	c.stateMachine = stateless.NewStateMachine(stateLivingroomFloorlampOff) // can this be reliable determined early on? probably not
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateLivingroomFloorlampOn).
		OnEntry(c.turnOnLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOff, c.masterController.guardTurnOffLivingroomLamp)

	c.stateMachine.Configure(stateLivingroomFloorlampOff).
		OnEntry(c.turnOffLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOn, c.masterController.guardTurnOnLivingroomLamp)

	c.isInitialized = true
	return nil
}

func (c *LivingroomController) turnOnLivingroomFloorlamp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(livingroomFloorlampOutput(true))
	return nil
}

func (c *LivingroomController) turnOffLivingroomFloorlamp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(livingroomFloorlampOutput(false))
	return nil
}

func livingroomFloorlampOutput(on bool) []MQTTPublish {
	return []MQTTPublish{setIkeaTretaktPower("zigbee2mqtt/livingroom-floorlamp/set", on)}
}
