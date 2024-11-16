package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	internal "github.com/claes/regelverk/internal"
)

func printHelp() {
	fmt.Println("Usage: regelverk [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	listenAddr := flag.String("listenAddr", ":8080", "HTTP listen address")
	rotelSerialPort := flag.String("rotelSerialPort", "", "Rotel serial port")
	samsungTVAddress := flag.String("samsungTVAddress", "", "Samsung TV address")
	pulseServer := flag.String("pulseServer", "", "Pulse server")
	mpdServer := flag.String("mpdServer", "", "MPD server")
	mpdPasswordFile := flag.String("mpdPasswordFile", "", "MPD password")
	help := flag.Bool("help", false, "Print help")
	debug := flag.Bool("debug", false, "Debug logging")
	dryRun := flag.Bool("dry_run", false, "Dry run (do not publish)")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *debug {
		var programLevel = new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)
		handler := slog.NewTextHandler(os.Stderr,
			&slog.HandlerOptions{Level: programLevel})
		slog.SetDefault(slog.New(handler))
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	config := internal.Config{
		Broker:           *mqttBroker,
		WebAddress:       *listenAddr,
		RotelSerialPort:  *rotelSerialPort,
		SamsungTvAddress: *samsungTVAddress,
		MpdServer:        *mpdServer,
		MpdPasswordFile:  *mpdPasswordFile,
		Pulseserver:      *pulseServer}

	loops := []internal.ControlLoop{
		//&tvLoop{},
		&internal.MpdLoop{},
		&internal.PresenceLoop{},
		&internal.KitchenLoop{},
		&internal.CecLoop{},
		&internal.WebLoop{},
	}

	bridgeWrappers := []internal.BridgeWrapper{
		&internal.CecBridgeWrapper{},
		&internal.MpdBridgeWrapper{},
		&internal.PulseaudioBridgeWrapper{},
		&internal.RotelBridgeWrapper{},
		&internal.SamsungBridgeWrapper{},
	}

	go func() {
		slog.Info("Initializing Regelverk", "config", config)
		err := internal.Regelverk(config, loops, bridgeWrappers, dryRun, debug)
		if err != nil {
			slog.Error("Error initializing regelverk", "error", err)
			os.Exit(1)
		} else {
			slog.Info("Initialized regelverk", "mqttBroker", mqttBroker)
		}
	}()

	slog.Info("Starting regelverk")
	<-c
	slog.Info("Shut down regelverk")
	os.Exit(0)
}
