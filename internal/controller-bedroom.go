package regelverk

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

//go:generate stringer -type=blindsState
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
	scheduleUp   time.Time
	scheduleDown time.Time
}

func (c *BedroomController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "bedroom"
	c.masterController = masterController
	// Use controller-specific trigger logic instead of BaseController's default
	c.getTriggers = c.GetTriggers

	// var initialState tvState
	// if masterController.stateValueMap.requireTrue("tvPower") {
	// 	initialState = stateTvOn
	// } else if masterController.stateValueMap.requireFalse("tvPower") {
	// 	initialState = stateTvOff
	// } else {
	// 	return nil
	// }

	c.stateMachine = stateless.NewStateMachine(bedroomBlindsStateOpen)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(bedroomBlindsStateOpen).
		OnEntry(c.openBedroomBlinds).
		Permit("blindsdown", bedroomBlindsStateClosed).
		Permit("blindsdowntemporarily", bedroomBlindsStateClosed).
		Ignore("blindsup").
		Ignore("blindsuptemporarily").
		PermitReentry("timer").
		OnEntryFrom("timer", c.refreshBedroomBlinds).
		OnEntryFrom("blindsuptemporarily", c.scheduleBlindsDown)

	c.stateMachine.Configure(bedroomBlindsStateClosed).
		OnEntry(c.closeBedroomBlinds).
		Permit("blindsup", bedroomBlindsStateOpen).
		Permit("blindsuptemporarily", bedroomBlindsStateOpen).
		Ignore("blindsdown").
		Ignore("blindsdowntemporarily").
		PermitReentry("timer").
		OnEntryFrom("timer", c.refreshBedroomBlinds).
		OnEntryFrom("blindsdowntemporarily", c.scheduleBlindsUp)

	go func() {
		for {
			now := time.Now()
			if now.Hour() == 9 && now.Minute() == 0 {
				c.StateMachineFire("blindsup")
			} else if now.Hour() == 21 && now.Minute() == 0 {
				c.StateMachineFire("blindsdown")
			}

			if now.Hour() == 8 && now.Minute() == 0 {
				c.StateMachineFire("timer")
			} else if now.Hour() == 20 && now.Minute() == 0 {
				c.StateMachineFire("timer")
			}

			if !c.scheduleDown.IsZero() && now.After(c.scheduleDown) {
				c.StateMachineFire("blindsdown")
				c.scheduleDown = time.Time{} // unset
			} else if !c.scheduleUp.IsZero() && now.After(c.scheduleUp) {
				c.StateMachineFire("blindsup")
				c.scheduleUp = time.Time{} // unset
			}
			time.Sleep(1 * time.Minute)
		}
	}()

	c.SetInitialized()
	return nil
}

// Shadowing method
func (c *BedroomController) GetTriggers(ev MQTTEvent) []string {
	val, _ := processJSON(ev, "zigbee2mqtt/blinds-bedroom-remote", "action")
	slog.Info("Get triggers for bedroom", "controller", c.Name, "event", ev.Topic, "action", val)
	if val != nil {
		if val.(string) == "on" {
			slog.Info("Received blinds up temporarily", "controller", c.Name)
			return []string{"blindsuptemporarily"}
		} else if val.(string) == "off" {
			slog.Info("Received blinds down temporarily", "controller", c.Name)
			return []string{"blindsdowntemporarily"}
		}
	}
	return []string{"mqttEvent"}
}

func (c *BedroomController) scheduleBlindsDown(_ context.Context, _ ...any) error {
	c.scheduleDown = time.Now().Add(30 * time.Minute)
	return nil
}

func (c *BedroomController) scheduleBlindsUp(_ context.Context, _ ...any) error {
	c.scheduleUp = time.Now().Add(30 * time.Minute)
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
			Payload:  `{"state": ""}`,
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
			Payload:  fmt.Sprintf(`{"state": "%s"}`, state),
			Qos:      2,
			Retained: true,
		},
		{
			Topic:    "zigbee2mqtt/blinds-bedroom/get",
			Payload:  fmt.Sprintf(`{"state": "%s"}`, state),
			Qos:      2,
			Wait:     60 * time.Second,
			Retained: true,
		},
	}
}
