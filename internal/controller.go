package regelverk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	pulseaudiomqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	routerosmqtt "github.com/claes/mqtt-bridges/routeros-mqtt/lib"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/qmuntal/stateless"
)

type MetricsConfig struct {
	CollectMetrics      bool
	CollectDebugMetrics bool
	MetricsAddress      string
	MetricsRealm        string
}

type MasterController struct {
	stateValueMap  StateValueMap
	controllers    *[]Controller
	mu             sync.Mutex
	pushMetrics    bool
	metricsConfig  MetricsConfig
	eventCallbacks []func(MQTTEvent)
}

func CreateMasterController() MasterController {
	return MasterController{stateValueMap: NewStateValueMap()}
}

func (l *MasterController) Init() {
	l.registerEventCallbacks()
	if l.metricsConfig.CollectMetrics {
		slog.Info("Registering state value callback in master controller")
		l.stateValueMap.registerCallback(l.StateValueCallback)
	}
}

func (l *MasterController) StateValueCallback(key string, value, new, updated bool) {
	if l.metricsConfig.CollectMetrics {
		gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`statevalue{name="%s",realm="%s"}`, key, l.metricsConfig.MetricsRealm), nil)
		if value {
			gauge.Set(1)
		} else {
			gauge.Set(0)
		}
		if new || updated {
			l.pushMetrics = true
		}
	}
}

type Controller interface {
	sync.Locker

	IsInitialized() bool
	Initialize(sm *MasterController) []MQTTPublish
	ProcessEvent(ev MQTTEvent) []MQTTPublish
}

type BaseController struct {
	name             string
	masterController *MasterController
	stateMachine     *stateless.StateMachine
	eventsToPublish  []MQTTPublish
	isInitialized    bool
	eventHandlers    []func(ev MQTTEvent) []MQTTPublish
	mu               sync.Mutex
}

func (c *BaseController) String() string {
	return c.name
}

func (c *BaseController) Lock() {
	c.mu.Lock()
}

func (c *BaseController) Unlock() {
	c.mu.Unlock()
}

func (c *BaseController) SetInitialized() {
	c.isInitialized = true

	if c.masterController.metricsConfig.CollectMetrics {
		stateInt, ok := c.stateMachine.MustState().(int)
		if ok {
			gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			gauge.Set(float64(stateInt))
			c.masterController.pushMetrics = true
		}
	}
}

func (c *BaseController) IsInitialized() bool {
	return c.isInitialized
}

func (c *BaseController) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	slog.Debug("Process event", "name", c.name)

	// In case special handling is needed that is not part of base processing
	// Under normal circumstances, state machine should be able to handle most
	for _, eventHandler := range c.eventHandlers {
		c.addEventsToPublish(eventHandler(ev))
	}

	beforeState := c.stateMachine.MustState()
	c.StateMachineFire("mqttEvent", ev)

	eventsToPublish := c.getAndResetEventsToPublish()
	afterState := c.stateMachine.MustState()
	slog.Debug("Event fired", "fsm", c.name, "topic", ev.Topic,
		"beforeState", beforeState,
		"afterState", afterState,
		"stateDiff", (beforeState != afterState),
		"eventsToPublish", (len(eventsToPublish) > 0),
		"noOfEventsToPublish", len(eventsToPublish))

	if c.masterController.metricsConfig.CollectDebugMetrics {
		triggerStr := createTriggerString(ev)
		if afterState != beforeState {
			if intState, ok := beforeState.(interface{ ToInt() int }); ok {
				beforeStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_change{controller="%s",trigger="%s",realm="%s"}`,
					c.name, triggerStr, c.masterController.metricsConfig.MetricsRealm), nil)
				beforeStateGauge.Set(float64(intState.ToInt()))
			} else {
				slog.Error("State does not implement ToInt", "state", afterState)
			}
		}
		if len(eventsToPublish) > 0 {
			counter := metrics.GetOrCreateCounter(fmt.Sprintf(`fsm_state_events{controller="%s",trigger="%s",realm="%s"}`,
				c.name, triggerStr, c.masterController.metricsConfig.MetricsRealm))
			counter.Add(len(eventsToPublish))
		}
		if intState, ok := beforeState.(interface{ ToInt() int }); ok {
			beforeStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_before{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			beforeStateGauge.Set(float64(intState.ToInt()))
		} else {
			slog.Error("State does not implement ToInt", "state", afterState)
		}
		if intState, ok := afterState.(interface{ ToInt() int }); ok {
			afterStateGauge := metrics.GetOrCreateGauge(fmt.Sprintf(`fsm_state_after{controller="%s",realm="%s"}`,
				c.name, c.masterController.metricsConfig.MetricsRealm), nil)
			afterStateGauge.Set(float64(intState.ToInt()))
		} else {
			slog.Error("State does not implement ToInt", "state", afterState)
		}
	}
	return eventsToPublish
}

func createTriggerString(trigger stateless.Trigger) string {
	var triggerStr string
	switch trigger.(type) {
	case string:
		triggerStr = trigger.(string)
	case MQTTEvent:
		ev := trigger.(MQTTEvent)
		triggerStr = ev.Topic
	default:
		triggerStr = "trigger"
	}
	return triggerStr
}
func (c *BaseController) StateMachineFire(trigger stateless.Trigger, args ...any) error {

	if c.masterController.metricsConfig.CollectDebugMetrics {
		counter := metrics.GetOrCreateCounter(fmt.Sprintf(`fsm_fire{controller="%s",realm="%s"}`,
			c.name, c.masterController.metricsConfig.MetricsRealm))
		counter.Inc()
	}
	return c.stateMachine.Fire(trigger, args...)
}

func (c *BaseController) addEventsToPublish(events []MQTTPublish) {
	c.eventsToPublish = append(c.eventsToPublish, events...)
}

func (c *BaseController) getAndResetEventsToPublish() []MQTTPublish {
	events := c.eventsToPublish
	c.eventsToPublish = []MQTTPublish{}
	return events
}

func (masterController *MasterController) ProcessEvent(client mqtt.Client, ev MQTTEvent) {

	masterController.mu.Lock()
	defer masterController.mu.Unlock()

	masterController.pushMetrics = false // Reset
	masterController.executeEventCallbacks(ev)

	for _, c := range *masterController.controllers {
		controller := c
		go func() {
			// For reliability, we call each loop in its own goroutine (yes, one
			// per message), so that one loop can be stuck while others still
			// make progress.

			controller.Lock()
			defer controller.Unlock()

			var toPublish []MQTTPublish
			if !controller.IsInitialized() {
				// If initialize requires other processes to update some state to determine
				// correct init state it can be requested  by events returned here
				// But the Initialize method must make sure to not request unneccessarily often
				toPublish = append(toPublish, controller.Initialize(masterController)...)
			}
			if controller.IsInitialized() {
				toPublish = append(toPublish, controller.ProcessEvent(ev)...)
			}

			for _, result := range toPublish {
				count = count + 1
				go func(toPublish MQTTPublish) {
					if toPublish.Wait != 0 {
						time.Sleep(toPublish.Wait)
					}
					client.Publish(toPublish.Topic, toPublish.Qos, toPublish.Retained, toPublish.Payload)

					if masterController.metricsConfig.CollectDebugMetrics {
						counter := metrics.GetOrCreateCounter(fmt.Sprintf(`regelverk_mqtt_published{topic="%s",realm="%s"}`,
							toPublish.Topic, masterController.metricsConfig.MetricsRealm))
						counter.Inc()
					}
				}(result)
			}
		}()
	}
	masterController.checkPushMetrics()
}

func (masterController *MasterController) checkPushMetrics() {
	if masterController.metricsConfig.CollectMetrics && masterController.pushMetrics {
		ctx := context.Background()
		metrics.PushMetrics(ctx, "http://"+masterController.metricsConfig.MetricsAddress+"/api/v1/import/prometheus", false, nil)
		// ctx := context.Background()
		// //ctx, err := context.WithTimeout(context.Background(), 100*time.Millisecond)
		// if err != nil {
		// 	slog.Error("Error creating context with timeout", "error", err)
		// } else {
		// 	metrics.PushMetrics(ctx, "http://"+masterController.metricsConfig.MetricsAddress+"/api/v1/import/prometheus", false, nil)
		// }
	}
}

// Guards

func (l *MasterController) guardStateMPDOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("mpdPlay")
	return check
}

func (l *MasterController) guardStateMPDOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("mpdPlay")
	return check
}

func (l *MasterController) guardStateSnapcastOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("snapcast")
	return check
}

func (l *MasterController) guardStateSnapcastOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("snapcast")
	return check
}

func (l *MasterController) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("phonePresent") &&
		l.stateValueMap.requireTrue("nighttime") &&
		l.stateValueMap.requireTrueRecently("livingroomPresence", 10*time.Minute)
	return check
}

func (l *MasterController) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("phonePresent") ||
		l.stateValueMap.requireFalse("nighttime") ||
		l.stateValueMap.requireTrueNotRecently("livingroomPresence", 10*time.Minute)
	return check
}

func (l *MasterController) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("tvPower")
	return check
}

func (l *MasterController) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("tvPower")
	return check
}

func (l *MasterController) guardStateTvOffLong(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("tvPower", 30*time.Minute)
	return check
}

func (l *MasterController) guardStateKitchenAmpOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("kitchenAudioPlaying")
	return check
}

func (l *MasterController) guardStateKitchenAmpOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrueNotRecently("kitchenAudioPlaying", 10*time.Minute)
	return check
}

func (l *MasterController) guardStateBedroomBlindsOpen(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireFalse("nighttime")
	return check
}

func (l *MasterController) guardStateBedroomBlindsClosed(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.requireTrue("nighttime")
	return check
}

// Detections

// func (l *MasterController) detectPhonePresent(ev MQTTEvent) {
// 	if ev.Topic == "routeros/wificlients" {
// 		var wifiClients []routerosmqtt.WifiClient

// 		err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
// 		if err != nil {
// 			slog.Error("Could not parse payload", "topic", "routeros/wificlients", "error", err)
// 			return
// 		}
// 		found := false
// 		for _, wifiClient := range wifiClients {
// 			if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
// 				found = true
// 				break
// 			}
// 		}
// 		l.stateValueMap.setState("phonePresent", found)
// 	}
// }

// func (l *MasterController) detectNighttime(ev MQTTEvent) {
// 	if ev.Topic == "regelverk/ticker/timeofday" {
// 		l.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
// 	}
// }

func (l *MasterController) detectTVPower(ev MQTTEvent) {
	if ev.Topic == "regelverk/state/tvpower" {
		tvPower, err := strconv.ParseBool(string(ev.Payload.([]byte)))
		if err != nil {
			slog.Error("Could not parse payload", "topic", "regelverk/state/tvpower", "error", err)
		}
		l.stateValueMap.setState("tvPower", tvPower)
	}
}

// func (l *MasterController) detectKitchenAudioPlaying(ev MQTTEvent) {
// 	if ev.Topic == "kitchen/pulseaudio/state" {
// 		var pulseaudioState pulseaudiomqtt.PulseAudioState
// 		err := json.Unmarshal(ev.Payload.([]byte), &pulseaudioState)
// 		if err != nil {
// 			slog.Error("Could not parse payload", "topic", "kitchen/pulseaudio/state", "error", err)
// 			return
// 		}
// 		l.stateValueMap.setState("kitchenAudioPlaying", pulseaudioState.DefaultSink.State == 0)
// 	}
// }

func setIkeaTretaktPower(topic string, on bool) MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return MQTTPublish{
		Topic:    topic,
		Payload:  fmt.Sprintf("{\"state\": \"%s\"}", state),
		Qos:      2,
		Retained: true,
	}
}

func processJSON(ev MQTTEvent, topic, eventProperty string) (any, bool) {
	if ev.Topic == topic {
		m := parseJSONPayload(ev)
		if m == nil {
			return nil, false
		}
		val, exists := m[eventProperty]
		if !exists || val == nil {
			return nil, false
		}
		return val, true
	} else {
		return nil, false
	}
}

func processString(ev MQTTEvent, topic string) (string, bool) {
	if ev.Topic == topic {
		s := string(ev.Payload.([]byte))
		return s, true
	} else {
		return "", false
	}
}

func (l *MasterController) createProcessEventFunc(extractValueFunc func(MQTTEvent) (any, bool),
	stateValueFunc func(any) (string, bool),
	metricsGaugeFunc func(any) (string, float64)) func(MQTTEvent) {

	return func(ev MQTTEvent) {
		val, _ := extractValueFunc(ev)
		if val != nil {

			if stateValueFunc != nil {
				key, b := stateValueFunc(val)
				l.stateValueMap.setState(key, b)
			}

			if metricsGaugeFunc != nil {
				key, v := metricsGaugeFunc(val)
				if l.metricsConfig.CollectMetrics {
					gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`eventvalue{name="%s",realm="%s"}`, key, l.metricsConfig.MetricsRealm), nil)
					gauge.Set(v)
				}
			}
		}
	}
}

func (masterController *MasterController) registerEventCallback(callback func(MQTTEvent)) {
	masterController.eventCallbacks = append(masterController.eventCallbacks, callback)
}

func (masterController *MasterController) executeEventCallbacks(ev MQTTEvent) {
	for _, callback := range masterController.eventCallbacks {
		callback(ev)
	}
}

func (masterController *MasterController) registerEventCallbacks() {

	//masterController.registerCallback(masterController.detectPhonePresent)
	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "routeros/wificlients" {
			var wifiClients []routerosmqtt.WifiClient

			err := json.Unmarshal(ev.Payload.([]byte), &wifiClients)
			if err != nil {
				slog.Error("Could not parse payload", "topic", "routeros/wificlients", "error", err)
				return
			}
			found := false
			for _, wifiClient := range wifiClients {
				if wifiClient.MacAddress == "AA:73:49:2B:D8:45" {
					found = true
					break
				}
			}
			masterController.stateValueMap.setState("phonePresent", found)
		}
	})
	// masterController.registerCallback(masterController.detectNighttime)
	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "regelverk/ticker/timeofday" {
			masterController.stateValueMap.setState("nighttime", ev.Payload.(TimeOfDay) == Nighttime)
		}
	})

	// Livingroom
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "occupancy")
		},
		func(val any) (string, bool) { return "livingroomPresence", val.(bool) },
		nil,
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "battery")
		},
		func(val any) (string, bool) { return "livingroomPresenceBatteryLow", val.(float64) < 20 },
		func(val any) (string, float64) { return "livingroomPresenceBattery", val.(float64) },
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-presence", "illuminance_lux")
		},
		nil,
		func(val any) (string, float64) { return "livingroomPresenceIlluminanceLux", val.(float64) },
	))

	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/livingroom-floorlamp", "state")
		},
		func(val any) (string, bool) { return "livingroomFloorlamp", val.(string) == "ON" },
		nil,
	))
	masterController.registerEventCallback(masterController.detectTVPower)
	// masterController.registerEventCallback(masterController.createProcessEventFunc(
	// 	func(ev MQTTEvent) (any, bool) {
	// 		return processString(ev, "regelverk/state/tvpower")
	// 	},
	// 	func(val any) (string, bool) { b, _ := strconv.ParseBool(val.(string)); return "tvPower", b },
	// 	nil,
	// ))

	// Kitchen
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/kitchen-amp", "state")
		},
		func(val any) (string, bool) { return "kitchenAmpPower", val.(string) == "ON" },
		nil,
	))

	//masterController.registerCallback(masterController.detectKitchenAudioPlaying)
	masterController.registerEventCallback(func(ev MQTTEvent) {
		if ev.Topic == "kitchen/pulseaudio/state" {
			var pulseaudioState pulseaudiomqtt.PulseAudioState
			err := json.Unmarshal(ev.Payload.([]byte), &pulseaudioState)
			if err != nil {
				slog.Error("Could not parse payload", "topic", "kitchen/pulseaudio/state", "error", err)
				return
			}
			masterController.stateValueMap.setState("kitchenAudioPlaying", pulseaudioState.DefaultSink.State == 0)
		}
	})

	// Bedroom
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/blinds-bedroom", "position")
		},
		func(val any) (string, bool) { return "bedroomBlindsOpen", val.(float64) > 50 },
		func(val any) (string, float64) { return "bedroomBlindsPosition", val.(float64) },
	))

	// Balcony door
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/balcony-door", "contact")
		},
		func(val any) (string, bool) { return "balconyDoorOpen", !val.(bool) },
		nil,
	))
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "zigbee2mqtt/balcony-door", "battery")
		},
		func(val any) (string, bool) { return "balconyDoorBatteryLow", val.(float64) < 20 },
		func(val any) (string, float64) { return "balconyDoorBattery", val.(float64) },
	))

	// MPD
	masterController.registerEventCallback(masterController.createProcessEventFunc(
		func(ev MQTTEvent) (any, bool) {
			return processJSON(ev, "mpd/status", "state")
		},
		func(val any) (string, bool) { return "mpdPlay", val.(string) == "play" },
		nil,
	))

}
