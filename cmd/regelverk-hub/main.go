package main

import (
	"time"

	internal "github.com/claes/regelverk/internal"
)

func main() {

	config := internal.ParseConfig()

	bridgeWrappers := &[]internal.BridgeWrapper{
		&internal.CecBridgeWrapper{},
		&internal.MpdBridgeWrapper{},
		&internal.HidBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
		&internal.RotelBridgeWrapper{},
		&internal.RouterOSBridgeWrapper{},
		&internal.SamsungBridgeWrapper{},
		&internal.SnapcastBridgeWrapper{},
	}

	controllers := &[]internal.Controller{
		&internal.TVController{},
		&internal.KitchenController{},
		//&internal.KitchenFreezerDoorController{},
		&internal.DoorController{
			Name:            "kitchenfreezerdoor",
			StateOpenKey:    "freezerDoorOpen",
			OpenLongLimit:   10 * time.Second,
			ReminderTopic:   "kitchen/audio/play",
			ReminderPayload: `embed://assets/ping.wav`,
			ReminderPeriod:  10 * time.Second,
			MaxReminders:    20},
		&internal.LivingroomController{},
		&internal.BedroomController{},
		&internal.SnapcastController{},
		&internal.WebController{},
	}

	internal.StartRegelverk(config, bridgeWrappers, controllers)
}
