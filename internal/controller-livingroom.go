package regelverk

import (
	"context"
	"reflect"

	"github.com/qmuntal/stateless"
)

type livingroomLamp int

const (
	stateLivingroomFloorlampOff livingroomLamp = iota
	stateLivingroomFloorlampOn
)

func (t livingroomLamp) ToInt() int {
	return int(t)
}

type LivingroomController struct {
	BaseController
}

func (c *LivingroomController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "livingroom"
	c.masterController = masterController

	var initialState livingroomLamp
	if masterController.stateValueMap.requireTrue("livingroomFloorlamp") {
		initialState = stateLivingroomFloorlampOn
	} else if masterController.stateValueMap.requireFalse("livingroomFloorlamp") {
		initialState = stateLivingroomFloorlampOff
	} else {
		return []MQTTPublish{requestIkeaTretaktPower("zigbee2mqtt/livingroom-floorlamp/get")}
	}

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateLivingroomFloorlampOn).
		OnEntry(c.turnOnLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOff, c.masterController.guardTurnOffLivingroomLamp)

	c.stateMachine.Configure(stateLivingroomFloorlampOff).
		OnEntry(c.turnOffLivingroomFloorlamp).
		Permit("mqttEvent", stateLivingroomFloorlampOn, c.masterController.guardTurnOnLivingroomLamp)

	c.SetInitialized()
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
