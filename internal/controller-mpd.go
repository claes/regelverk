package regelverk

import (
	"context"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

type mpdState int

const (
	mpdStateOff mpdState = iota
	mpdStateOn
)

func (t mpdState) ToInt() int {
	return int(t)
}

type MPDController struct {
	BaseController
}

func (c *MPDController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "mpd"
	c.masterController = masterController

	// var initialState tvState
	// if masterController.stateValueMap.requireTrue("tvpower") {
	// 	initialState = stateTvOn
	// } else if masterController.stateValueMap.requireFalse("tvpower") {
	// 	initialState = stateTvOff
	// } else {
	// 	return nil
	// }

	c.stateMachine = stateless.NewStateMachine(mpdStateOff)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(mpdStateOn).
		OnEntry(c.turnOnMPD).
		Permit("mqttEvent", mpdStateOff, c.masterController.guardStateMPDOff)

	c.stateMachine.Configure(mpdStateOff).
		Permit("mqttEvent", mpdStateOn, c.masterController.guardStateMPDOn)

	c.isInitialized = true
	return nil
}

func (c *MPDController) turnOnMPD(_ context.Context, _ ...any) error {
	c.addEventsToPublish(mpdPlayOutput())
	return nil
}

func mpdPlayOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "rotel/command/send",
			Payload:  "power_on!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt2!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:iec958-stereo+input:analog-stereo",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}
