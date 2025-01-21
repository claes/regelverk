package main

import (
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
		&internal.KitchenFreezerDoorController{},
		&internal.LivingroomController{},
		&internal.BedroomController{},
		&internal.SnapcastController{},
		&internal.WebController{},
	}

	internal.StartRegelverk(config, bridgeWrappers, controllers)
}
