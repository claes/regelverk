package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
)

//go:embed templates/rotel.html
var content2 embed.FS
var rotelStateUpdated = make(chan struct{})

type rotelHttpLoop struct {
	statusLoop
	mqttMessageHandler *mqttMessageHandler
	rotelState         map[string]interface{}
}

func (l *rotelHttpLoop) Init(m *mqttMessageHandler) {
	l.mqttMessageHandler = m
	http.HandleFunc("/", l.mainHandler)
	http.HandleFunc("/rotel/state", l.stateHandler)
	http.HandleFunc("/rotel/state/init", l.rotelStateInitWs)
	http.HandleFunc("/rotel/state/ws", l.rotelStateWs)
	http.HandleFunc("/rotel/source", l.rotelSourceHandler)
	http.HandleFunc("/rotel/volume", l.rotelVolumeHandler)
}

func (l *rotelHttpLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "rotel/state":
		l.rotelState = parseJSONPayload(ev)
		rotelStateUpdated <- struct{}{}
	}
	return nil
}

func (l *rotelHttpLoop) mainHandler(w http.ResponseWriter, r *http.Request) {

	data, readErr := content2.ReadFile("templates/rotel.html")
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

func (l *rotelHttpLoop) stateHandler(w http.ResponseWriter, r *http.Request) {

	var keys []string
	for k := range l.rotelState {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintf(w, "<div>")
	for _, key := range keys {
		fmt.Fprintf(w, "<div>%s: %s</div>", key, l.rotelState[key])
	}
	fmt.Fprintf(w, "</div>")
}

func (l *rotelHttpLoop) rotelSourceHandler(w http.ResponseWriter, r *http.Request) {
	selectedSource := r.FormValue("rotel-source")
	fmt.Println("Selected: " + selectedSource)
	//TODO - refactor?
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

func (l *rotelHttpLoop) rotelVolumeHandler(w http.ResponseWriter, r *http.Request) {
	volume := r.FormValue("rotel-volume")
	fmt.Println("Selected: " + volume)
	//TODO - refactor?
	l.mqttMessageHandler.client.Publish("rotel/command/send", 2, false, "volume_"+volume+"!")
	l.rotelVolumeRenderer(w, volume)
}

func (l *rotelHttpLoop) rotelVolumeRenderer(w io.Writer, currentVolume string) {
	fmt.Fprintf(w, "<input type='range' id='rotel-volume' name='rotel-volume' value='%s' min='0' max='96' hx-post='/rotel/volume' hx-trigger='change' hx-swap-oob='true' />", currentVolume)
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

		var keys []string
		for k := range l.rotelState {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(socketWriter, "<div id='rotel-state' hx-swap-oob='true'>")
		for _, key := range keys {
			fmt.Fprintf(socketWriter, "<div>%s: %s</div>", key, l.rotelState[key])
		}
		fmt.Fprintf(socketWriter, "</div>")

		l.rotelSourceRenderer(socketWriter, l.rotelState["source"].(string))

		l.rotelVolumeRenderer(socketWriter, l.rotelState["volume"].(string))

		socketWriter.Close()

		select {
		case <-rotelStateUpdated:
			continue
		}
	}
}
