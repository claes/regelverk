package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config, debug, dryRun := internal.ParseConfig()

	loops := []internal.ControlLoop{}

	bridgeWrappers := []internal.BridgeWrapper{
		&internal.PulseaudioBridgeWrapper{},
	}

	internal.StartRegelverk(config, loops, bridgeWrappers, dryRun, debug)
}