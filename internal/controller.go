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
