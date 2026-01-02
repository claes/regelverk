package regelverk

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/VictoriaMetrics/metrics"
	pulsemqtt "github.com/claes/mqtt-bridges/pulseaudio-mqtt/lib"
	rotelmqtt "github.com/claes/mqtt-bridges/rotel-mqtt/lib"
	"github.com/gorilla/websocket"
	"github.com/qmuntal/stateless"
)

//go:embed templates/rotel.html templates/styles.css
var webContent embed.FS

type webState int

const (
	initialWebState webState = iota
)

type WebController struct {
	BaseController
	rotelState             rotelmqtt.RotelState
	pulseAudioState        pulsemqtt.PulseAudioState
	kitchenPlug            IkeaInspelning
	upgrader               websocket.Upgrader
	rotelStateUpdated      chan struct{}
	pulseaudioStateUpdated chan struct{}
}

func IsInitialized() bool {
	return false
}

func (l *WebController) Initialize(masterController *MasterController) []MQTTPublish {
	l.Name = "web"
	l.masterController = masterController

	// Define a dummy state machine as the basecontroller assumes that
	l.stateMachine = stateless.NewStateMachine(initialWebState)
	l.stateMachine.SetTriggerParameters("mqttEvent", reflect.TypeOf(MQTTEvent{}))
	l.stateMachine.Configure(initialWebState)

	slog.Info("Setting up HTTP handlers")
	http.HandleFunc("/", l.mainHandler)
	http.HandleFunc("/web/state/init", l.rotelStateInitWs)
	http.HandleFunc("/web/state/ws", l.rotelStateWs)

	http.HandleFunc("/rotel/source", l.rotelSourceHandler)
	http.HandleFunc("/rotel/tone", l.rotelToneHandler)
	http.HandleFunc("/rotel/mute", l.rotelMuteHandler)
	http.HandleFunc("/rotel/volume", l.rotelVolumeHandler)
	http.HandleFunc("/rotel/balance", l.rotelBalanceHandler)
	http.HandleFunc("/rotel/bass", l.rotelBassHandler)
	http.HandleFunc("/rotel/treble", l.rotelTrebleHandler)
	http.HandleFunc("/rotel/power", l.rotelPowerHandler)

	http.HandleFunc("/pulseaudio/sink", l.pulseaudioSinkHandler)
	http.HandleFunc("/pulseaudio/profile", l.pulseaudioProfileHandler)

	http.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		data, _ := webContent.ReadFile("templates/styles.css")
		w.Header().Add("Content-Type", "text/css")
		w.Write(data)
	})
	slog.Info("Finished setting up HTTP handlers")

	go func() {
		slog.Info("Initializing HTTP server", "address", masterController.config.WebAddress)

		err := http.ListenAndServe(masterController.config.WebAddress, nil)

		if err != nil {
			slog.Error("Error initializing HTTP server",
				"listenAddr", masterController.config.WebAddress, "error", err)
			os.Exit(1)
		}
	}()

	// Needed?
	l.rotelStateUpdated = make(chan struct{}, 1)
	l.pulseaudioStateUpdated = make(chan struct{}, 1)

	l.SetInitialized()

	return []MQTTPublish{
		{
			Topic:    "rotel/command/initialize",
			Payload:  "true",
			Qos:      2,
			Retained: false,
		},
		{
			Topic:    "pulseaudio/initialize",
			Payload:  "true",
			Qos:      2,
			Retained: false,
		},
	}
}

func (l *WebController) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "rotel/state":
		err := json.Unmarshal(ev.Payload.([]byte), &l.rotelState)
		if err != nil {
			slog.Error("Could not unmarshal rotel state", "rotelstate", ev.Payload)
		} else {
			l.notifyRotelStateUpdated()
		}
	case "pulseaudio/state":
		err := json.Unmarshal(ev.Payload.([]byte), &l.pulseAudioState)
		if err != nil {
			slog.Error("Could not unmarshal pulseaudio state", "pulseaudiostate", ev.Payload)
		} else {
			l.notifyPulseaudioStateUpdated()
		}
	case "zigbee2mqtt/kitchen-sink":
		var err error
		l.kitchenPlug, err = UnmarshalIkeaInspelning(ev.Payload.([]byte))
		if err != nil {
			slog.Error("Could not unmarshal kitchen sink power plug state", "payload", ev.Payload)
		} else {
			topic := "zigbee2mqtt/kitchen-sink"
			l.logMetrics(&l.kitchenPlug, topic)
		}

		// case "regelverk/ticker/1s":
		// 	_, _, second := time.Now().Clock()
		// 	if second%10 == 0 {
		// 		returnList := []MQTTPublish{
		// 			{
		// 				Topic:    "rotel/command/initialize",
		// 				Payload:  "true",
		// 				Qos:      2,
		// 				Retained: false,
		// 				Wait:     0 * time.Second,
		// 			},
		// 			{
		// 				Topic:    "pulseaudio/initialize",
		// 				Payload:  "true",
		// 				Qos:      2,
		// 				Retained: false,
		// 				Wait:     0 * time.Second,
		// 			},
		// 		}
		// 		return returnList
		// 	}
	}
	return nil
}

func (l *WebController) logMetrics(r interface{}, topic string) {
	v := reflect.ValueOf(r)
	t := reflect.TypeOf(r)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		value := v.Field(i).Interface()
		switch v.Field(i).Kind() {
		case reflect.Float64:
			gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`zigbee_state{topic="%s",attribute="%s"}`,
				topic, jsonTag), nil)
			gauge.Set(value.(float64))
			l.masterController.pushMetrics = true
		case reflect.Int64:
			gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`zigbee_state{topic="%s",attribute="%s"}`,
				topic, jsonTag), nil)
			gauge.Set(float64(value.(int64)))
			l.masterController.pushMetrics = true
		case reflect.Bool:
			gauge := metrics.GetOrCreateGauge(fmt.Sprintf(`zigbee_state{topic="%s",attribute="%s"}`,
				topic, jsonTag), nil)
			if value.(bool) {
				gauge.Set(1.0)
			} else {
				gauge.Set(0.0)
			}
			l.masterController.pushMetrics = true
		}
	}
}

func (l *WebController) notifyRotelStateUpdated() {
	select {
	case l.rotelStateUpdated <- struct{}{}:
	default:
	}
}

func (l *WebController) notifyPulseaudioStateUpdated() {
	select {
	case l.pulseaudioStateUpdated <- struct{}{}:
	default:
	}
}

func (l *WebController) mainHandler(w http.ResponseWriter, r *http.Request) {

	data, readErr := webContent.ReadFile("templates/rotel.html") //TODO rename
	if readErr != nil {
		http.Error(w, "Failed to read embedded template", http.StatusInternalServerError)
		return
	}

	t, parseErr := template.New("rotel").Parse(string(data))
	if parseErr != nil {
		http.Error(w, "Failed to parse template: "+parseErr.Error(), http.StatusInternalServerError)
		return
	}

	execErr := t.Execute(w, nil)
	if execErr != nil {
		http.Error(w, "Failed to render template: "+execErr.Error(), http.StatusInternalServerError)
		return
	}
}

func (l *WebController) rotelSourceHandler(w http.ResponseWriter, r *http.Request) {
	selectedSource := r.FormValue("rotel-source")
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, selectedSource+"!")
	l.rotelSourceRenderer(w, selectedSource)
}

func (l *WebController) rotelSourceRenderer(w io.Writer, currentSource string) {
	var sources = []string{"opt1", "opt2", "coax1", "coax2"}
	fmt.Fprintf(w, "<select id='rotel-source' name='rotel-source' hx-post='/rotel/source' hx-trigger='change' hx-swap-oob='true'>")
	for _, source := range sources {
		selected := ""
		if source == currentSource {
			selected = "selected"
		}
		fmt.Fprintf(w, "<option value='%s' %s >%s</option>", source, selected, source)
	}
	fmt.Fprintf(w, "</select>")
}

func (l *WebController) pulseaudioSinkHandler(w http.ResponseWriter, r *http.Request) {
	selectedSink := r.FormValue("pulseaudio-sink")
	l.masterController.mqttClient.Publish("pulseaudio/sink/default/set", 2, false, selectedSink)
	l.pulseaudioSinkRenderer(w, selectedSink)
}

func (l *WebController) pulseaudioSinkRenderer(w io.Writer, currentSink string) {
	fmt.Fprintf(w, "<select id='pulseaudio-sink' name='pulseaudio-sink' hx-post='/pulseaudio/sink' hx-trigger='change' hx-swap-oob='true'>")
	for _, sink := range l.pulseAudioState.Sinks {
		selected := ""
		if sink.Id == currentSink {
			selected = "selected"
		}
		fmt.Fprintf(w, "<option value='%s' %s >%s</option>", sink.Id, selected, sink.Name)
	}
	fmt.Fprintf(w, "</select>")
}

func (l *WebController) pulseaudioProfileHandler(w http.ResponseWriter, r *http.Request) {
	selectedProfile := r.FormValue("pulseaudio-profile")
	l.masterController.mqttClient.Publish("pulseaudio/cardprofile/0/set", 2, false, selectedProfile)
	l.pulseaudioProfileRenderer(w, selectedProfile)
}

func (l *WebController) pulseaudioProfileRenderer(w io.Writer, currentProfile string) {
	if len(l.pulseAudioState.Cards) > 0 {
		fmt.Fprintf(w, "<select id='pulseaudio-profile' name='pulseaudio-profile' hx-post='/pulseaudio/profile' hx-trigger='change' hx-swap-oob='true'>")

		for _, profile := range l.pulseAudioState.Cards[0].Profiles {
			selected := ""
			if profile.Name == currentProfile {
				selected = "selected"
			}
			fmt.Fprintf(w, "<option value='%s' %s >%s</option>", profile.Name, selected, profile.Name)
		}
		fmt.Fprintf(w, "</select>")
	}
}

func (l *WebController) rotelToneHandler(w http.ResponseWriter, r *http.Request) {
	tone := r.FormValue("rotel-tone")
	if tone != "on" {
		tone = "off"
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "tone_"+tone+"!")
	l.rotelToneRenderer(w, tone)
}

func (l *WebController) rotelToneRenderer(w io.Writer, currentTone string) {
	checked := ""
	if currentTone == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-tone' name='rotel-tone' value='on' hx-post='/rotel/tone' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *WebController) rotelMuteHandler(w http.ResponseWriter, r *http.Request) {
	mute := r.FormValue("rotel-mute")
	if mute != "on" {
		mute = "off"
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "mute_"+mute+"!")
	l.rotelMuteRenderer(w, mute)
}

func (l *WebController) rotelMuteRenderer(w io.Writer, currentMute string) {
	checked := ""
	if currentMute == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-mute' name='rotel-mute' value='on' hx-post='/rotel/mute' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *WebController) rotelPowerHandler(w http.ResponseWriter, r *http.Request) {
	power := r.FormValue("rotel-power")
	if power != "on" {
		power = "off"
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "power_"+power+"!")
	l.masterController.mqttClient.Publish("rotel/command/initialize", 2, false, "true")
	l.rotelPowerRenderer(w, power)
}

func (l *WebController) rotelPowerRenderer(w io.Writer, currentPower string) {
	checked := ""
	if currentPower == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-power' name='rotel-power' value='on' hx-post='/rotel/power' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *WebController) rotelVolumeHandler(w http.ResponseWriter, r *http.Request) {
	volume := r.FormValue("rotel-volume")
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "volume_"+volume+"!")
	//	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "get_display!")
	l.rotelVolumeRenderer(w, volume)
}

func (l *WebController) rotelVolumeRenderer(w io.Writer, currentVolume string) {
	fmt.Fprintf(w, "<input type='range' id='rotel-volume' name='rotel-volume' value='%s' min='0' max='96' hx-post='/rotel/volume' hx-trigger='change' hx-swap-oob='true' />", currentVolume)
}

func (l *WebController) rotelBalanceHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-balance"))
	if err != nil {
		slog.Error("Could not parse balance", "error", err)
		return
	}
	balance, err := intToBalance(b)
	if err != nil {
		slog.Error("Could not parse balance", "error", err)
		return
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "balance_"+balance+"!")
	l.rotelBalanceRenderer(w, balance)
}

func (l *WebController) rotelBalanceRenderer(w io.Writer, currentBalance string) {
	// L15 -- 000 -- R15
	balance, err := balanceToInt(currentBalance)
	if err != nil {
		slog.Error("Could not parse balance as int", "error", err, "value", currentBalance)
		return
	}
	fmt.Fprintf(w, "<input type='range' id='rotel-balance' name='rotel-balance' value='%d' min='-15' max='15' hx-post='/rotel/balance' hx-trigger='change' hx-swap-oob='true' />", balance)
}

func balanceToInt(s string) (int, error) {
	switch {
	case strings.HasPrefix(s, "L"):
		val, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}
		return -val, nil
	case strings.HasPrefix(s, "R"):
		val, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}
		return val, nil
	case s == "000":
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid string format, %s", s)
	}
}

func intToBalance(n int) (string, error) {
	if n < -15 || n > 15 {
		return "", fmt.Errorf("number out of range")
	}
	switch {
	case n < 0:
		return fmt.Sprintf("L%02d", -n), nil
	case n > 0:
		return fmt.Sprintf("R%02d", n), nil
	case n == 0:
		return "000", nil
	}
	return "", fmt.Errorf("Unexpected number value: %d", n)
}

// treble

func (l *WebController) rotelTrebleHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-treble"))
	if err != nil {
		slog.Error("Could not parse treble", "error", err)
		return
	}
	treble, err := intToBassOrTreble(b)
	if err != nil {
		slog.Error("Could not parse treble", "error", err)
		return
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "treble_"+treble+"!")
	l.rotelTrebleRenderer(w, treble)
}

func (l *WebController) rotelTrebleRenderer(w io.Writer, currentTreble string) {
	// -10 -- 000 -- +10
	treble, err := bassOrTrebleToInt(currentTreble)
	if err != nil {
		slog.Error("Could not parse treble as int", "error", err, "value", currentTreble)
		return
	}
	fmt.Fprintf(w, "<input type='range' id='rotel-treble' name='rotel-treble' value='%d' min='-10' max='10' hx-post='/rotel/treble' hx-trigger='change' hx-swap-oob='true' />", treble)
}

func bassOrTrebleToInt(s string) (int, error) {
	switch {
	case strings.HasPrefix(s, "-"):
		val, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}
		return -val, nil
	case strings.HasPrefix(s, "+"):
		val, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}
		return val, nil
	case s == "000":
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid string format, %s", s)
	}
}

func intToBassOrTreble(n int) (string, error) {
	if n < -10 || n > 10 {
		return "", fmt.Errorf("number out of range")
	}
	switch {
	case n < 0:
		return fmt.Sprintf("-%02d", -n), nil
	case n > 0:
		return fmt.Sprintf("+%02d", n), nil
	case n == 0:
		return "000", nil
	}
	return "", fmt.Errorf("Unexpected number value: %d", n)
}

// bass

func (l *WebController) rotelBassHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-bass"))
	if err != nil {
		slog.Error("Could not parse bass", "error", err)
		return
	}
	bass, err := intToBassOrTreble(b)
	if err != nil {
		slog.Error("Could not parse bass", "error", err)
		return
	}
	l.masterController.mqttClient.Publish("rotel/command/send", 2, false, "bass_"+bass+"!")
	l.rotelBassRenderer(w, bass)
}

func (l *WebController) rotelBassRenderer(w io.Writer, currentBass string) {
	// -10 -- 000 -- +10
	bass, err := bassOrTrebleToInt(currentBass)
	if err != nil {
		slog.Error("Could not parse bass as int", "error", err, "value", currentBass)
		return
	}
	fmt.Fprintf(w, "<input type='range' id='rotel-bass' name='rotel-bass' value='%d' min='-10' max='10' hx-post='/rotel/bass' hx-trigger='change' hx-swap-oob='true' />", bass)
}

func (l *WebController) rotelDisplayRenderer(w io.Writer, text string) {

	pos := 20
	var t string
	if len(text) >= 20 {
		t = text[:pos] + "\n" + text[pos:]
	} else {
		t = text
	}
	fmt.Fprintf(w, "<pre class='lcd-display' id='rotel-display' name='rotel-display' hx-swap-oob='true'>%s</pre>", t)
}

func (l *WebController) rotelStateInitWs(w http.ResponseWriter, req *http.Request) {
	var responseTemplate = `
	<div id="ws-output" hx-ext="ws" ws-connect="/web/state/ws">	
		<div id="rotel-state"></div>
	</div>
	`
	tmpl := template.New("ws-output")
	tmpl.Parse(responseTemplate)

	tmpl.Execute(w, nil)
}

func (l *WebController) rotelStateWs(w http.ResponseWriter, req *http.Request) {

	c, err := l.upgrader.Upgrade(w, req, nil)
	if err != nil {
		slog.Error("Error trying to upgrade", "error", err)
		return
	}
	defer c.Close()

	for {
		socketWriter, err := c.NextWriter(websocket.TextMessage)

		if err != nil {
			slog.Error("Error getting socket writer", "error", err)
			break
		}

		l.rotelDisplayRenderer(socketWriter, l.rotelState.Display)

		l.rotelSourceRenderer(socketWriter, l.rotelState.Source)

		l.rotelToneRenderer(socketWriter, l.rotelState.Tone)

		l.rotelMuteRenderer(socketWriter, l.rotelState.Mute)

		l.rotelVolumeRenderer(socketWriter, l.rotelState.Volume)

		l.rotelBalanceRenderer(socketWriter, l.rotelState.Balance)

		l.rotelBassRenderer(socketWriter, l.rotelState.Bass)

		l.rotelTrebleRenderer(socketWriter, l.rotelState.Treble)

		l.rotelPowerRenderer(socketWriter, l.rotelState.State)

		l.pulseaudioSinkRenderer(socketWriter, l.pulseAudioState.DefaultSink.Id)

		if len(l.pulseAudioState.ActiveProfilePerCard) > 0 {
			l.pulseaudioProfileRenderer(socketWriter, l.pulseAudioState.ActiveProfilePerCard[0])
		}

		socketWriter.Close()

		select {
		case <-l.rotelStateUpdated:
			continue
		case <-l.pulseaudioStateUpdated:
			continue
		}
	}
}
