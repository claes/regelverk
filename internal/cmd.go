package regelverk

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
)

func ParseConfig() (Config, *bool, *bool) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mqttBroker := flag.String("mqttBroker", "tcp://localhost:1883", "MQTT broker URL")
	mqttUserName := flag.String("mqttUserName", "", "MQTT username")
	MQTTPasswordFile := flag.String("mqttPasswordFile", "", "MQTT password file")
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

	config := Config{
		MQTTBroker:       *mqttBroker,
		MQTTUserName:     *mqttUserName,
		MQTTPasswordFile: *MQTTPasswordFile,
		WebAddress:       *listenAddr,
		RotelSerialPort:  *rotelSerialPort,
		SamsungTvAddress: *samsungTVAddress,
		MpdServer:        *mpdServer,
		MpdPasswordFile:  *mpdPasswordFile,
		Pulseserver:      *pulseServer}
	return config, debug, dryRun
}

func printHelp() {
	fmt.Println("Usage: regelverk [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func StartRegelverk(config Config, loops []ControlLoop, bridgeWrappers []BridgeWrapper, dryRun *bool, debug *bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		slog.Info("Initializing Regelverk", "config", config)
		err := Regelverk(config, loops, bridgeWrappers, dryRun, debug)
		if err != nil {
			slog.Error("Error initializing regelverk", "error", err)
			os.Exit(1)
		} else {
			slog.Info("Initialized regelverk")
		}
	}()

	slog.Info("Starting regelverk")
	<-c
	slog.Info("Shut down regelverk")
	os.Exit(0)
}
