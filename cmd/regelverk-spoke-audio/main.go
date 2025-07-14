package main

import (
	internal "github.com/claes/regelverk/internal"
)

func main() {

	config := internal.ParseConfig()

	bridgeWrappers := []internal.BridgeWrapper{
		&internal.AudioBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
		&internal.BluezBridgeWrapper{},
	}

	controllers := &[]internal.Controller{}

	internal.StartRegelverk(config, &bridgeWrappers, controllers)
}
