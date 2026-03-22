package regelverk

import (
	"context"
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

//go:generate stringer -type=livingroomWindowLamp
type livingroomWindowLamp int

const (
	stateLivingroomWindowlampOff livingroomWindowLamp = iota
	stateLivingroomWindowlampOn
)

func (t livingroomWindowLamp) ToInt() int {
	return int(t)
}

type LivingroomWindowlampController struct {
	BaseController
}

func (c *LivingroomWindowlampController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "livingroom"
	c.masterController = masterController
	c.triggerFactory = c.createTriggers

	var initialState livingroomWindowLamp
	if masterController.stateValueMap.currentlyTrue("livingroomWindowlamp") {
		initialState = stateLivingroomWindowlampOn
	} else if masterController.stateValueMap.currentlyFalse("livingroomWindowlamp") {
		initialState = stateLivingroomWindowlampOff
	} else {
		const maxBackoff = 128 * time.Second
		if c.checkBackoff() {
			c.extendBackoff(maxBackoff)
			return []MQTTPublish{requestIkeaKajplatsPower("zigbee2mqtt/livingroom-windowlamp/get")}
		} else {
			return nil
		}
	}

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(stateLivingroomWindowlampOn).
		OnEntry(c.turnOnLivingroomWindowlamp).
		Permit("non-evening", stateLivingroomWindowlampOff)

	c.stateMachine.Configure(stateLivingroomWindowlampOff).
		OnEntry(c.turnOffLivingroomWindowlamp).
		Permit("evening", stateLivingroomWindowlampOn)
	c.SetInitialized()
	return nil
}

func (c *LivingroomWindowlampController) createTriggers(ev MQTTEvent) []string {
	phaseOfDay, found := processType[PhaseOfDay](ev, "regelverk/ticker/phaseofday")
	if found {
		if phaseOfDay.Meridiem == PostMeridiem &&
			(phaseOfDay.SolarPhase == Nighttime ||
				phaseOfDay.SolarPhase == EveningAstronomcialTwilight ||
				phaseOfDay.SolarPhase == EveningNauticalTwilight) {
			return []string{"evening"}
		} else {
			return []string{"non-evening"}
		}
	}
	return []string{"mqttEvent"}
}

func (c *LivingroomWindowlampController) turnOnLivingroomWindowlamp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(livingroomWindowlampOutput(true))
	return nil
}

func (c *LivingroomWindowlampController) turnOffLivingroomWindowlamp(_ context.Context, _ ...any) error {
	c.addEventsToPublish(livingroomWindowlampOutput(false))
	return nil
}

func livingroomWindowlampOutput(on bool) []MQTTPublish {
	return []MQTTPublish{setIkeaKajplatsPower("zigbee2mqtt/livingroom-windowlamp/set", on)}
}
