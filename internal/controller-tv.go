package regelverk

import (
	"context"
	"reflect"
	"time"

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
	c.name = "tv"
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

func tvPowerOffOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/ikea_uttag/set",
			Payload:  "{\"state\": \"OFF\", \"power_on_behavior\": \"ON\"}",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}

func tvPowerOffLongOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "rotel/command/send",
			Payload:  "power_off!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
	}
}

func tvPowerOnOutput() []MQTTPublish {
	result := []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/ikea_uttag/set",
			Payload:  "{\"state\": \"ON\", \"power_on_behavior\": \"ON\"}",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "power_on!",
			Qos:      2,
			Retained: false,
			Wait:     0 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "volume_38!",
			Qos:      2,
			Retained: false,
			Wait:     2 * time.Second,
		},
		{
			Topic:    "rotel/command/send",
			Payload:  "opt1!",
			Qos:      2,
			Retained: false,
			Wait:     3 * time.Second,
		},
		{
			Topic:    "pulseaudio/cardprofile/0/set",
			Payload:  "output:hdmi-stereo",
			Qos:      2,
			Retained: false,
			Wait:     3 * time.Second,
		},
	}

	// Need to wait here since a newly started TV is not receptive first 20 or so seconds
	for i := int64(15); i < 40; i++ {
		p := MQTTPublish{
			Topic:    "samsungremote/key/reconnectsend",
			Payload:  "KEY_VOLDOWN",
			Qos:      2,
			Retained: false,
			Wait:     time.Duration(i) * time.Second / 2,
		}
		result = append(result, p)
	}
	return result
}
