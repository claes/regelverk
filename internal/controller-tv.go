package regelverk

import (
	"context"
	"reflect"

	"github.com/qmuntal/stateless"
)

type tvState int

const (
	stateTvOn tvState = iota
	stateTvOff
	stateTvOffLong
)

type TVController struct {
	BaseController
}

func (c *TVController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "tv-controller"
	c.masterController = masterController

	var initialState tvState
	if masterController.stateValueMap.requireTrue("tvpower") {
		initialState = stateTvOn
	} else if masterController.stateValueMap.requireFalse("tvpower") {
		initialState = stateTvOff
	} else {
		return nil
	}

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateTvOn).
		OnEntry(c.turnOnTvAppliances).
		Permit("mqttEvent", stateTvOff, masterController.guardStateTvOff)

	c.stateMachine.Configure(stateTvOff).
		OnEntry(c.turnOffTvAppliances).
		Permit("mqttEvent", stateTvOn, masterController.guardStateTvOn).
		Permit("mqttEvent", stateTvOffLong, masterController.guardStateTvOffLong)

	c.stateMachine.Configure(stateTvOffLong).
		OnEntry(c.turnOffTvAppliancesLong).
		Permit("mqttEvent", stateTvOn, masterController.guardStateTvOn)

	c.isInitialized = true
	return nil
}

func (c *TVController) turnOnTvAppliances(_ context.Context, _ ...any) error {
	c.addEventsToPublish(tvPowerOnOutput())
	return nil
}

func (c *TVController) turnOffTvAppliances(_ context.Context, _ ...any) error {
	c.addEventsToPublish(tvPowerOffOutput())
	return nil
}

func (c *TVController) turnOffTvAppliancesLong(_ context.Context, _ ...any) error {
	return nil
}
