package regelverk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Controller interface {
	sync.Locker

	IsInitialized() bool
	Initialize(sm *MasterController) []MQTTPublish
	ProcessEvent(ev MQTTEvent) []MQTTPublish
	DebugState() ControllerDebugState
}

// ControllerDebugState provides a JSON-friendly snapshot of a controller.
type ControllerDebugState struct {
	Name                  string        `json:"name"`
	Initialized           bool          `json:"initialized"`
	StateMachineState     any           `json:"stateMachineState,omitempty"`
	StateMachineStateText string        `json:"stateMachineStateText,omitempty"`
	BackoffUntil          time.Time     `json:"backoffUntil,omitempty"`
	LastBackoffDuration   time.Duration `json:"lastBackoffDuration,omitempty"`
}

type MasterController struct {
	mqttClient     mqtt.Client
	stateValueMap  StateValueMap
	controllers    *[]Controller
	mu             sync.Mutex
	pushMetrics    bool
	metricsConfig  MetricsConfig
	config         Config
	eventCallbacks []func(MQTTEvent)
}

type MetricsConfig struct {
	CollectMetrics      bool
	CollectDebugMetrics bool
	MetricsAddress      string
	MetricsRealm        string
}

func CreateMasterController() MasterController {
	return MasterController{stateValueMap: NewStateValueMap()}
}

func (l *MasterController) Init() {
	l.registerEventCallbacks()
	if l.metricsConfig.CollectMetrics {
		slog.Info("Registering state value callback in master controller")
		l.stateValueMap.registerObserverCallback(l.StateValueCallback)
	}
}

func (l *MasterController) StateValueCallback(key StateKey, value, new, updated bool) {
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
	check := l.stateValueMap.currentlyTrue("mpdPlay")
	return check
}

func (l *MasterController) guardStateMPDOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("mpdPlay")
	return check
}

func (l *MasterController) guardStateSnapcastOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("snapcast")
	return check
}

func (l *MasterController) guardStateSnapcastOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("snapcast")
	return check
}

func (l *MasterController) guardTurnOnLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("phonePresent") &&
		l.stateValueMap.currentlyTrue("nighttime") &&
		l.stateValueMap.recentlyTrue("livingroomPresence", 10*time.Minute)
	return check
}

func (l *MasterController) guardTurnOffLivingroomLamp(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("phonePresent") ||
		l.stateValueMap.currentlyFalse("nighttime") ||
		!l.stateValueMap.recentlyTrue("livingroomPresence", 10*time.Minute)
	return check
}

func (l *MasterController) guardStateTvOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("tvPower")
	return check
}

func (l *MasterController) guardStateTvOff(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("tvPower")
	return check
}

func (l *MasterController) guardStateTvOffLong(_ context.Context, _ ...any) bool {
	check := !l.stateValueMap.recentlyTrue("tvPower", 30*time.Minute)
	return check
}

func (l *MasterController) guardStateKitchenAmpOn(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("kitchenAudioPlaying")
	return check
}

func (l *MasterController) guardStateKitchenAmpOff(_ context.Context, _ ...any) bool {
	check := !l.stateValueMap.recentlyTrue("kitchenAudioPlaying", 10*time.Minute)
	return check
}

func (l *MasterController) guardStateBedroomBlindsOpen(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyFalse("nighttime")
	return check
}

func (l *MasterController) guardStateBedroomBlindsClosed(_ context.Context, _ ...any) bool {
	check := l.stateValueMap.currentlyTrue("nighttime")
	return check
}

func (l *MasterController) requireTrueByKey(key StateKey) func(context.Context, ...any) bool {
	return func(_ context.Context, _ ...any) bool {
		check := l.stateValueMap.currentlyTrue(key)
		return check
	}
}

func (l *MasterController) requireTrueSinceByKey(key StateKey, duration time.Duration) func(context.Context, ...any) bool {
	return func(_ context.Context, _ ...any) bool {
		check := l.stateValueMap.continuouslyTrue(key, duration)
		return check
	}
}

func (l *MasterController) requireFalseByKey(key StateKey) func(context.Context, ...any) bool {
	return func(_ context.Context, _ ...any) bool {
		check := l.stateValueMap.currentlyFalse(key)
		return check
	}
}

// Detections

func setIkeaTretaktPower(topic string, on bool) MQTTPublish {
	state := "OFF"
	if on {
		state = "ON"
	}
	return MQTTPublish{
		Topic:    topic,
		Payload:  fmt.Sprintf(`{"state": "%s"}`, state),
		Qos:      2,
		Retained: true,
	}
}

func requestIkeaTretaktPower(topic string) MQTTPublish {
	return MQTTPublish{
		Topic:    topic,
		Payload:  `{"state": ""}`,
		Qos:      2,
		Retained: false,
	}
}
