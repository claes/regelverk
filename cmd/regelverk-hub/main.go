package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config, debug, dryRun := internal.ParseConfig()

	loops := []internal.ControlLoop{
		//&internal.MpdLoop{},
		&internal.BedroomLoop{},
		&internal.CecLoop{},
		&internal.KitchenLoop{},
		&internal.LivingroomLoop{},
		&internal.SnapcastLoop{},
		&internal.TVLoop{},
		&internal.WebLoop{},
	}

	bridgeWrappers := &[]internal.BridgeWrapper{
		&internal.CecBridgeWrapper{},
		&internal.MpdBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
		&internal.RotelBridgeWrapper{},
		&internal.RouterOSBridgeWrapper{},
		&internal.SamsungBridgeWrapper{},
		&internal.SnapcastBridgeWrapper{},
	}

	internal.StartRegelverk(config, loops, bridgeWrappers, dryRun, debug)
}
