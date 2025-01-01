package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config := internal.ParseConfig()

	bridgeWrappers := []internal.BridgeWrapper{
		&internal.SnapcastBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
	}

	controllers := &[]internal.Controller{
		&internal.SnapcastController{},
	}

	internal.StartRegelverk(config, &bridgeWrappers, controllers)
}
