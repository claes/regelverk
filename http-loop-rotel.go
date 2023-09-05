package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
)

//go:embed templates/rotel.html
var content2 embed.FS
var rotelStateUpdated = make(chan struct{})

type rotelHttpLoop struct {
	statusLoop
	rotelState map[string]interface{}
}

func (l *rotelHttpLoop) Init() {
	fmt.Println("Rotel http init")
	http.HandleFunc("/", l.mainHandler)
	http.HandleFunc("/rotel/state", l.stateHandler)
	http.HandleFunc("/rotel/state/init", l.rotelStateInitWs)
	http.HandleFunc("/rotel/state/ws", l.rotelStateWs)
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

	pageVariables := PageVariables{}

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

	execErr := t.Execute(w, pageVariables)
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

var upgrader = websocket.Upgrader{} // use default options

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

		socketWriter.Close()

		select {
		case <-rotelStateUpdated:
			continue
		}
	}
}
