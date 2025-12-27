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
		&internal.TelegramBridgeWrapper{},
	}

	controllers := &[]internal.Controller{
		&internal.TVController{},
		&internal.KitchenController{},
		//&internal.KitchenFreezerDoorController{},
		&internal.DoorReminderController{
			Name:            "kitchenfreezerdoor",
			SensorName:      "freezer-door",
			StateOpenKey:    "freezerDoorOpen",
			OpenLongLimit:   10 * time.Second,
			ReminderPeriod:  10 * time.Second,
			MaxReminders:    20,
			ReminderTopic:   "kitchen/audio/play",
			ReminderPayload: `embed://assets/ping.wav`,
		},
		&internal.DoorReminderController{
			Name:            "kitchenfridgedoor",
			SensorName:      "fridge-door",
			StateOpenKey:    "fridgeDoorOpen",
			OpenLongLimit:   10 * time.Second,
			ReminderPeriod:  10 * time.Second,
			MaxReminders:    20,
			ReminderTopic:   "kitchen/audio/play",
			ReminderPayload: `embed://assets/ping.wav`,
		},
		&internal.BatteryReminderController{
			Name:                "balconydoorbattery",
			StateBatteryPoorKey: "balconyDoorBatteryLow",
			ReminderPeriod:      24 * time.Hour,
			MaxReminders:        20,
			ReminderTopic:       "telegram/regelverkgeneral/send",
			ReminderPayload:     `Battery balcony door is low`,
		},
		&internal.BatteryReminderController{
			Name:                "kitchenfeezerdoorbattery",
			StateBatteryPoorKey: "freezerDoorBatteryLow",
			ReminderPeriod:      24 * time.Hour,
			MaxReminders:        20,
			ReminderTopic:       "telegram/regelverkgeneral/send",
			ReminderPayload:     `Battery freezer door is low`,
		},
		&internal.BatteryReminderController{
			Name:                "kitchenfridgedoorbattery",
			StateBatteryPoorKey: "fridgeDoorBatteryLow",
			ReminderPeriod:      24 * time.Hour,
			MaxReminders:        20,
			ReminderTopic:       "telegram/regelverkgeneral/send",
			ReminderPayload:     `Battery fridge door is low`,
		},
		&internal.LivingroomController{},
		&internal.BedroomController{},
		&internal.SnapcastController{},
		&internal.WebController{},
		&internal.DebugController{},
	}

	internal.StartRegelverk(config, bridgeWrappers, controllers)
}
