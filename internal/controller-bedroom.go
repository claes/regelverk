package regelverk

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

type blindsState int

const (
	bedroomBlindsStateClosed blindsState = iota
	bedroomBlindsStateOpen
)

func (t blindsState) ToInt() int {
	return int(t)
}

type BedroomController struct {
	BaseController
}

func (c *BedroomController) Initialize(masterController *MasterController) []MQTTPublish {
	c.name = "bedroom"
	c.masterController = masterController

	// var initialState tvState
	// if masterController.stateValueMap.requireTrue("tvpower") {
	// 	initialState = stateTvOn
	// } else if masterController.stateValueMap.requireFalse("tvpower") {
	// 	initialState = stateTvOff
	// } else {
	// 	return nil
	// }

	c.stateMachine = stateless.NewStateMachine(bedroomBlindsStateOpen)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(bedroomBlindsStateOpen).
		OnEntry(c.openBedroomBlinds).
		Permit("blindsdown", bedroomBlindsStateClosed).
		Ignore("blindsup").
		PermitReentry("timer").
		OnEntryFrom("timer", c.refreshBedroomBlinds)

	c.stateMachine.Configure(bedroomBlindsStateClosed).
		OnEntry(c.closeBedroomBlinds).
		Permit("blindsup", bedroomBlindsStateOpen).
		Ignore("blindsdown").
		PermitReentry("timer").
		OnEntryFrom("timer", c.refreshBedroomBlinds)

	// TODO - how to detect state from manual actions?
	// Any use of detectBedroomBlindsOpen /  guardStateBedroomBlindsOpen / Closed?

	go func() {
		for {
			now := time.Now()
			if now.Hour() == 9 && now.Minute() == 0 {
				c.stateMachine.Fire("blindsup")
			} else if now.Hour() == 21 && now.Minute() == 0 {
				c.stateMachine.Fire("blindsdown")
			}

			if now.Hour() == 8 && now.Minute() == 0 {
				c.stateMachine.Fire("timer")
			} else if now.Hour() == 20 && now.Minute() == 0 {
				c.stateMachine.Fire("timer")
			}

			time.Sleep(1 * time.Minute)
		}
	}()

	c.isInitialized = true
	return nil
}

func (c *BedroomController) openBedroomBlinds(_ context.Context, _ ...any) error {
	c.addEventsToPublish(bedroomBlindsOutput(true))
	return nil
}

func (c *BedroomController) closeBedroomBlinds(_ context.Context, _ ...any) error {
	c.addEventsToPublish(bedroomBlindsOutput(false))
	return nil
}

func (c *BedroomController) refreshBedroomBlinds(_ context.Context, _ ...any) error {
	c.addEventsToPublish(bedroomBlindsRefreshOutput())
	return nil
}

func bedroomBlindsRefreshOutput() []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/blinds-bedroom/get",
			Payload:  "{\"state\": \"\"}",
			Qos:      2,
			Retained: false,
		},
	}
}

func bedroomBlindsOutput(open bool) []MQTTPublish {
	state := "CLOSE"
	if open {
		state = "OPEN"
	}
	return []MQTTPublish{
		{
			Topic:    "zigbee2mqtt/blinds-bedroom/set",
			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
			Qos:      2,
			Retained: true,
		},
		{
			Topic:    "zigbee2mqtt/blinds-bedroom/get",
			Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
			Qos:      2,
			Wait:     60 * time.Second,
			Retained: true,
		},
	}
}
