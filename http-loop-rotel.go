package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed templates/rotel.html templates/styles.css
var content embed.FS

var rotelStateUpdated = make(chan struct{})

type rotelHttpLoop struct {
	statusLoop
	mqttMessageHandler *mqttMessageHandler
	rotelPreviousState map[string]interface{}
	rotelState         map[string]interface{}
}

func (l *rotelHttpLoop) Init(m *mqttMessageHandler) {
	l.mqttMessageHandler = m
	http.HandleFunc("/", l.mainHandler)
	http.HandleFunc("/rotel/state/init", l.rotelStateInitWs)
	http.HandleFunc("/rotel/state/ws", l.rotelStateWs)
	http.HandleFunc("/rotel/source", l.rotelSourceHandler)
	http.HandleFunc("/rotel/tone", l.rotelToneHandler)
	http.HandleFunc("/rotel/mute", l.rotelMuteHandler)
	http.HandleFunc("/rotel/volume", l.rotelVolumeHandler)
	http.HandleFunc("/rotel/balance", l.rotelBalanceHandler)
	http.HandleFunc("/rotel/bass", l.rotelBassHandler)
	http.HandleFunc("/rotel/treble", l.rotelTrebleHandler)
	http.HandleFunc("/rotel/power", l.rotelPowerHandler)
	http.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		data, _ := content.ReadFile("templates/styles.css")
		w.Header().Add("Content-Type", "text/css")
		w.Write(data)
	})

	l.mqttMessageHandler.client.Publish("rotel/command/initialize", 2, false, "true")
}

func (l *rotelHttpLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "rotel/state":
		l.rotelPreviousState = l.rotelState
		l.rotelState = parseJSONPayload(ev)
		rotelStateUpdated <- struct{}{}
	case "regelverk/ticker/1s":
		_, _, second := time.Now().Clock()
		if second%10 == 0 {
			returnList := []MQTTPublish{
				{
					Topic:    "rotel/command/initialize",
					Payload:  "true",
					Qos:      2,
					Retained: false,
					Wait:     0 * time.Second,
				},
			}
			return returnList
		}
	}
	return nil
}

func (l *rotelHttpLoop) mainHandler(w http.ResponseWriter, r *http.Request) {

	data, readErr := content.ReadFile("templates/rotel.html")
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

func (l *rotelHttpLoop) rotelSourceHandler(w http.ResponseWriter, r *http.Request) {
	selectedSource := r.FormValue("rotel-source")
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, selectedSource+"!")
	l.rotelSourceRenderer(w, selectedSource)
}

func (l *rotelHttpLoop) rotelSourceRenderer(w io.Writer, currentSource string) {
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

func (l *rotelHttpLoop) rotelToneHandler(w http.ResponseWriter, r *http.Request) {
	tone := r.FormValue("rotel-tone")
	if tone != "on" {
		tone = "off"
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "tone_"+tone+"!")
	l.rotelToneRenderer(w, tone)
}

func (l *rotelHttpLoop) rotelToneRenderer(w io.Writer, currentTone string) {
	checked := ""
	if currentTone == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-tone' name='rotel-tone' value='on' hx-post='/rotel/tone' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *rotelHttpLoop) rotelMuteHandler(w http.ResponseWriter, r *http.Request) {
	mute := r.FormValue("rotel-mute")
	if mute != "on" {
		mute = "off"
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "mute_"+mute+"!")
	l.rotelMuteRenderer(w, mute)
}

func (l *rotelHttpLoop) rotelMuteRenderer(w io.Writer, currentMute string) {
	checked := ""
	if currentMute == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-mute' name='rotel-mute' value='on' hx-post='/rotel/mute' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *rotelHttpLoop) rotelPowerHandler(w http.ResponseWriter, r *http.Request) {
	power := r.FormValue("rotel-power")
	if power != "on" {
		power = "off"
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "power_"+power+"!")
	l.mqttMessageHandler.client.Publish("rotel/command/initialize", 2, false, "true")
	l.rotelPowerRenderer(w, power)
}

func (l *rotelHttpLoop) rotelPowerRenderer(w io.Writer, currentPower string) {
	checked := ""
	if currentPower == "on" {
		checked = "checked"
	}
	fmt.Fprintf(w, "<input type='checkbox' %s id='rotel-power' name='rotel-power' value='on' hx-post='/rotel/power' hx-trigger='change' hx-swap-oob='true' />", checked)
}

func (l *rotelHttpLoop) rotelVolumeHandler(w http.ResponseWriter, r *http.Request) {
	volume := r.FormValue("rotel-volume")
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "volume_"+volume+"!")
	//	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "get_display!")
	l.rotelVolumeRenderer(w, volume)
}

func (l *rotelHttpLoop) rotelVolumeRenderer(w io.Writer, currentVolume string) {
	fmt.Fprintf(w, "<input type='range' id='rotel-volume' name='rotel-volume' value='%s' min='0' max='96' hx-post='/rotel/volume' hx-trigger='change' hx-swap-oob='true' />", currentVolume)
}

func (l *rotelHttpLoop) rotelBalanceHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-balance"))
	if err != nil {
		fmt.Println("Could not parse balance:", err)
		return
	}
	balance, err := intToBalance(b)
	if err != nil {
		fmt.Println("Could not parse balance:", err)
		return
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "balance_"+balance+"!")
	l.rotelBalanceRenderer(w, balance)
}

func (l *rotelHttpLoop) rotelBalanceRenderer(w io.Writer, currentBalance string) {
	// L15 -- 000 -- R15
	balance, err := balanceToInt(currentBalance)
	if err != nil {
		fmt.Printf("Error %v\n", err)
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

func (l *rotelHttpLoop) rotelTrebleHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-treble"))
	if err != nil {
		fmt.Println("Could not parse treble:", err)
		return
	}
	treble, err := intToBassOrTreble(b)
	if err != nil {
		fmt.Println("Could not parse treble:", err)
		return
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "treble_"+treble+"!")
	l.rotelTrebleRenderer(w, treble)
}

func (l *rotelHttpLoop) rotelTrebleRenderer(w io.Writer, currentTreble string) {
	// -10 -- 000 -- +10
	treble, err := bassOrTrebleToInt(currentTreble)
	if err != nil {
		fmt.Printf("Error %v\n", err)
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

func (l *rotelHttpLoop) rotelBassHandler(w http.ResponseWriter, r *http.Request) {
	b, err := strconv.Atoi(r.FormValue("rotel-bass"))
	if err != nil {
		fmt.Println("Could not parse bass:", err)
		return
	}
	bass, err := intToBassOrTreble(b)
	if err != nil {
		fmt.Println("Could not parse bass:", err)
		return
	}
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "bass_"+bass+"!")
	l.rotelBassRenderer(w, bass)
}

func (l *rotelHttpLoop) rotelBassRenderer(w io.Writer, currentBass string) {
	// -10 -- 000 -- +10
	bass, err := bassOrTrebleToInt(currentBass)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return
	}
	fmt.Fprintf(w, "<input type='range' id='rotel-bass' name='rotel-bass' value='%d' min='-10' max='10' hx-post='/rotel/bass' hx-trigger='change' hx-swap-oob='true' />", bass)
}

func (l *rotelHttpLoop) rotelDisplayRenderer(w io.Writer, text string) {

	pos := 20
	var t string
	if len(text) >= 20 {
		t = text[:pos] + "\n" + text[pos:]
	} else {
		t = text
	}
	fmt.Fprintf(w, "<pre class='lcd-display' id='rotel-display' name='rotel-display' hx-swap-oob='true'>%s</pre>", t)
}

var upgrader = websocket.Upgrader{}

func (l *rotelHttpLoop) rotelStateInitWs(w http.ResponseWriter, req *http.Request) {
	var responseTemplate = `
	<div id="ws-output" hx-ext="ws" ws-connect="/rotel/state/ws">	
		<div id="rotel-state"></div>
	</div>
	`
	tmpl := template.New("ws-output")
	tmpl.Parse(responseTemplate)

	tmpl.Execute(w, nil)
}

func (l *rotelHttpLoop) rotelStateWs(w http.ResponseWriter, req *http.Request) {

	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Printf("Error trying to upgrade: %v\n", err)
		return
	}
	defer c.Close()

	for {
		socketWriter, err := c.NextWriter(websocket.TextMessage)

		if err != nil {
			fmt.Printf("Error getting socket writer %v\n", err)
			break
		}

		l.rotelDisplayRenderer(socketWriter, l.rotelState["display"].(string))

		l.rotelSourceRenderer(socketWriter, l.rotelState["source"].(string))

		l.rotelToneRenderer(socketWriter, l.rotelState["tone"].(string))

		l.rotelMuteRenderer(socketWriter, l.rotelState["mute"].(string))

		l.rotelVolumeRenderer(socketWriter, l.rotelState["volume"].(string))

		l.rotelBalanceRenderer(socketWriter, l.rotelState["balance"].(string))

		l.rotelBassRenderer(socketWriter, l.rotelState["bass"].(string))

		l.rotelTrebleRenderer(socketWriter, l.rotelState["treble"].(string))

		l.rotelPowerRenderer(socketWriter, l.rotelState["state"].(string))

		socketWriter.Close()

		select {
		case <-rotelStateUpdated:
			continue
		}
	}
}
