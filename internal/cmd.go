package regelverk

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
)

func ParseConfig() (Config, *bool, *bool) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mqttBroker := flag.String("mqttBroker", "tcp://localhost:1883", "MQTT broker URL")
	mqttTopicPrefix := flag.String("mqttTopicPrefix", "", "MQTT topic prefix")
	mqttUserName := flag.String("mqttUserName", "", "MQTT username")
	mqttPasswordFile := flag.String("mqttPasswordFile", "", "MQTT password file")
	httpListenAddress := flag.String("httpListenAddress", ":8080", "HTTP listen address")
	rotelSerialPort := flag.String("rotelSerialPort", "", "Rotel serial port")
	samsungTVAddress := flag.String("samsungTVAddress", "", "Samsung TV address")
	snapcastServer := flag.String("snapcastServer", "", "Snapcast server address")
	pulseServer := flag.String("pulseServer", "", "Pulse server")
	mpdServer := flag.String("mpdServer", "", "MPD server")
	mpdPasswordFile := flag.String("mpdPasswordFile", "", "MPD password file")
	routerAddress := flag.String("routerAddress", "", "Mikrotik router address:port")
	routerUsername := flag.String("routerUsername", "", "Mikrotik router username")
	routerPasswordFile := flag.String("routerPasswordFile", "", "Mikrotik router password file")
	bluetoothAddress := flag.String("bluetoothAddress", "", "Bluetooth MAC address")

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
		MQTTBroker:         *mqttBroker,
		MQTTTopicPrefix:    *mqttTopicPrefix,
		MQTTUserName:       *mqttUserName,
		MQTTPasswordFile:   *mqttPasswordFile,
		WebAddress:         *httpListenAddress,
		RotelSerialPort:    *rotelSerialPort,
		SamsungTvAddress:   *samsungTVAddress,
		SnapcastServer:     *snapcastServer,
		MpdServer:          *mpdServer,
		MpdPasswordFile:    *mpdPasswordFile,
		RouterAddress:      *routerAddress,
		RouterUsername:     *routerUsername,
		RouterPasswordFile: *routerPasswordFile,
		BluetoothAddress:   *bluetoothAddress,
		Pulseserver:        *pulseServer}
	return config, debug, dryRun
}

func printHelp() {
	fmt.Println("Usage: regelverk [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func StartRegelverk(config Config, loops []ControlLoop, bridgeWrappers *[]BridgeWrapper, controllers *[]Controller,
	dryRun *bool, debug *bool) {

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Initializing Regelverk", "config", config)
		err := runRegelverk(ctx, config, loops, bridgeWrappers, controllers, dryRun, debug)
		if err != nil {
			slog.Error("Error initializing regelverk", "error", err)
		}
	}()

	slog.Info("Starting regelverk")
	<-c
	cancel()
	wg.Wait()
	slog.Info("Shut down regelverk")
}
