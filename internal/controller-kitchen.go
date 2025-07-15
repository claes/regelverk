package regelverk

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/qmuntal/stateless"
)

type kitchenAmpState int

const (
	kitchenAmpStateOff kitchenAmpState = iota
	kitchenAmpStateOn
)

func (t kitchenAmpState) ToInt() int {
	return int(t)
}

type KitchenController struct {
	BaseController
}

func (c *KitchenController) Initialize(masterController *MasterController) []MQTTPublish {
	c.Name = "kitchen"
	c.masterController = masterController

	var initialState kitchenAmpState
	if masterController.stateValueMap.requireTrue("kitchenAmpPower") {
		initialState = kitchenAmpStateOn
	} else if masterController.stateValueMap.requireFalse("kitchenAmpPower") {
		initialState = kitchenAmpStateOff
	} else {
		const maxBackoff = 128 * time.Second
		if c.checkBackoff() {
			c.extendBackoff(maxBackoff)
			return []MQTTPublish{requestIkeaTretaktPower("zigbee2mqtt/kitchen-amp/get")}
		} else {
			return nil
		}
	}

	c.stateMachine = stateless.NewStateMachine(initialState)
	c.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))

	c.stateMachine.Configure(kitchenAmpStateOn).
		OnEntry(c.turnOnKitchenAmp).
		Permit("mqttEvent", kitchenAmpStateOff, c.masterController.guardStateKitchenAmpOff)

	c.stateMachine.Configure(kitchenAmpStateOff).
		OnEntry(c.turnOffKitchenAmp).
		Permit("mqttEvent", kitchenAmpStateOn, c.masterController.guardStateKitchenAmpOn)

	c.eventHandlers = append(c.eventHandlers, c.handleMediaRemoteEvents)

	c.SetInitialized()
	return nil
}

func (c *KitchenController) handleMediaRemoteEvents(ev MQTTEvent) []MQTTPublish {
	if ev.Topic == "zigbee2mqtt/media_remote_kitchen" {
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
		case "volume_up":
			return getPulseaudioVolumeChangeCommand(topicPrefix, 0.1)
		case "volume_down":
			return getPulseaudioVolumeChangeCommand(topicPrefix, -0.1)
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

func getPulseaudioVolumeChangeCommand(topicPrefix string, change float64) []MQTTPublish {
	return []MQTTPublish{
		{
			Topic:    topicPrefix + "/pulseaudio/volume/change",
			Payload:  strconv.FormatFloat(change, 'f', 2, 64),
			Qos:      2,
			Retained: false,
		},
	}
}
