package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config, debug, dryRun := internal.ParseConfig()

	loops := []internal.ControlLoop{
		//&tvLoop{},
		&internal.MpdLoop{},
		&internal.PresenceLoop{},
		&internal.KitchenLoop{},
		&internal.LivingroomLoop{},
		&internal.CecLoop{},
		&internal.WebLoop{},
	}

	bridgeWrappers := &[]internal.BridgeWrapper{
		&internal.CecBridgeWrapper{},
		&internal.MpdBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
		&internal.RotelBridgeWrapper{},
		&internal.SamsungBridgeWrapper{},
		&internal.RouterOSBridgeWrapper{},
	}

	internal.StartRegelverk(config, loops, bridgeWrappers, dryRun, debug)
}
