package regelverk

import (
	"context"
	"reflect"

	"github.com/qmuntal/stateless"
)

type ampState int

const (
	ampStateOn ampState = iota
	ampStateOff
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

	c.eventHandlers = append(c.eventHandlers, c.handleMediaRemoteEvents)

	c.isInitialized = true
	return nil
}

func (c *KitchenController) handleMediaRemoteEvents(ev MQTTEvent) []MQTTPublish {
	if ev.Topic == "zigbee2mqtt/livingroom_remote_kitchen" {
		m := parseJSONPayload(ev)
		if m == nil {
			return nil
		}
		val, exists := m["action"]
		if !exists || val == nil {
			return nil
		}
		topicPrefix := "kitchen"
		mac := "4C:66:A6:A1:39:58"
		switch val {
		case "toggle":
			return getBluezMediaplayerCommand(topicPrefix, mac, "Play")
		case "dots_1_initial_press":
			return getBluezMediaplayerCommand(topicPrefix, mac, "Pause")
		case "track_next":
			return getBluezMediaplayerCommand(topicPrefix, mac, "Next")
		case "track_previous":
			return getBluezMediaplayerCommand(topicPrefix, mac, "Previous")
		}
	}
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

func kitchenAmpPowerOutput(on bool) []MQTTPublish {
	return []MQTTPublish{setIkeaTretaktPower("zigbee2mqtt/kitchen-amp/set", on)}
}

func getBluezMediaplayerCommand(topicPrefix string, mac string, command string) []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    topicPrefix + "/bluez/" + mac + "/mediaplayer/command/send",
			Payload:  command,
			Qos:      2,
			Retained: false,
		},
	}
}
