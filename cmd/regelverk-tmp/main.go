package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config, debug, dryRun := internal.ParseConfig()

	loops := []internal.ControlLoop{}

	bridgeWrappers := []internal.BridgeWrapper{
		&internal.SnapcastBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
	}

	controllers := &[]internal.Controller{
		&internal.SnapcastController{},
	}

	internal.StartRegelverk(config, loops, &bridgeWrappers, controllers, dryRun, debug)
}
