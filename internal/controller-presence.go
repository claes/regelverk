package regelverk

import (
	"reflect"
	"time"

	"github.com/qmuntal/stateless"
)

type homePresenceState int

const (
	HomePresenceStateKey = StateKey("atHome")

	presenceInitial homePresenceState = iota
	presenceHome
	presenceAway
)

func (t homePresenceState) ToInt() int {
	return int(t)
}

type PresenceController struct {
	BaseController
}

func (c *PresenceController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "homepresence"
	c.masterController = masterController
	atHomeModel := BayesianModel{
		Prior:     0.6,
		Threshold: 0.9,
		Likelihoods: map[StateKey][]LikelihoodModel{
			"freezerDoorOpen": {
				{
					ProbGivenTrue:  0.9,              // If home, phone detected 90% of the time
					ProbGivenFalse: 0.01,             // If not home, phone still shows up 20% of the time
					HalfLife:       60 * time.Minute, // Evidence fades slowly
					Weight:         1.0,              // Highly trusted
				},
			},
			"fridgeDoorOpen": {
				{
					ProbGivenTrue:  0.8,  // If home, motion detected 80% of the time
					ProbGivenFalse: 0.01, // If not home, motion falsely triggered 30% of the time
					HalfLife:       15 * time.Minute,
					Weight:         1.0, // Less trusted
				},
				{
					ProbGivenTrue:  0.8,
					ProbGivenFalse: 0.01,
					HalfLife:       0,
					Weight:         1.0,
					StateValueEvaluator: func(value StateValue) (bool, time.Duration) {
						return value.recentlyTrue(10 * time.Minute), 10 * time.Minute
					},
				},
			},
		},
	}

	masterController.registerBayesianModel(HomePresenceStateKey, atHomeModel)

	c.stateMachine = stateless.NewStateMachine(presenceInitial)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(presenceInitial).
		Permit("mqttEvent", presenceHome, c.masterController.requireTrueByKey(HomePresenceStateKey)).
		Permit("mqttEvent", presenceAway, c.masterController.requireFalseByKey(HomePresenceStateKey))

	c.stateMachine.Configure(presenceHome).
		//OnEntry(c.turnOnLivingroomFloorlamp).
		Permit("mqttEvent", presenceAway, masterController.requireFalseByKey(HomePresenceStateKey))

	c.stateMachine.Configure(presenceAway).
		//OnEntry(c.turnOffLivingroomFloorlamp).
		Permit("mqttEvent", presenceHome, masterController.requireTrueByKey(HomePresenceStateKey))

	c.SetInitialized()
	return nil
}
