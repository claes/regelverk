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

	bluetoothAddress := flag.String("bluetoothAddress", "", "Bluetooth MAC address")
	hidProductID := flag.String("hidProductId", "", "HID product id")
	hidVendorID := flag.String("hidVendorId", "", "HID vendor id")
	httpListenAddress := flag.String("httpListenAddress", ":8080", "HTTP listen address")
	collectMetrics := flag.Bool("collectMetrics", false, "true/false whether to collect metrics")
	metricsAddress := flag.String("metricsAddress", "", "Metrics address")
	metricsRealm := flag.String("metricsRealm", "", "Metrics realm")
	mpdPasswordFile := flag.String("mpdPasswordFile", "", "MPD password file")
	mpdServer := flag.String("mpdServer", "", "MPD server")
	mqttBroker := flag.String("mqttBroker", "tcp://localhost:1883", "MQTT broker URL")
	mqttPasswordFile := flag.String("mqttPasswordFile", "", "MQTT password file")
	mqttTopicPrefix := flag.String("mqttTopicPrefix", "", "MQTT topic prefix")
	mqttUserName := flag.String("mqttUserName", "", "MQTT username")
	pulseServer := flag.String("pulseServer", "", "Pulse server")
	rotelSerialPort := flag.String("rotelSerialPort", "", "Rotel serial port")
	routerAddress := flag.String("routerAddress", "", "Mikrotik router address:port")
	routerPasswordFile := flag.String("routerPasswordFile", "", "Mikrotik router password file")
	routerUsername := flag.String("routerUsername", "", "Mikrotik router username")
	samsungTVAddress := flag.String("samsungTVAddress", "", "Samsung TV address")
	snapcastServer := flag.String("snapcastServer", "", "Snapcast server address")

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
		BluetoothAddress:   *bluetoothAddress,
		HIDProductID:       *hidProductID,
		HIDVendorID:        *hidVendorID,
		CollectMetrics:     *collectMetrics,
		MetricsAddress:     *metricsAddress,
		MetricsRealm:       *metricsRealm,
		MpdPasswordFile:    *mpdPasswordFile,
		MpdServer:          *mpdServer,
		MQTTBroker:         *mqttBroker,
		MQTTPasswordFile:   *mqttPasswordFile,
		MQTTTopicPrefix:    *mqttTopicPrefix,
		MQTTUserName:       *mqttUserName,
		Pulseserver:        *pulseServer,
		RotelSerialPort:    *rotelSerialPort,
		RouterAddress:      *routerAddress,
		RouterPasswordFile: *routerPasswordFile,
		RouterUsername:     *routerUsername,
		SamsungTvAddress:   *samsungTVAddress,
		SnapcastServer:     *snapcastServer,
		WebAddress:         *httpListenAddress,
	}
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
