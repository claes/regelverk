package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/rotel.html
var content2 embed.FS

type rotelHttpLoop struct {
	statusLoop
	rotelState map[string]interface{}
}

func (l *rotelHttpLoop) Init() {
	fmt.Println("Rotel http init")
	http.HandleFunc("/", l.mainHandler)
	http.HandleFunc("/rotel/state", l.stateHandler)
}

func (l *rotelHttpLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	switch ev.Topic {
	case "rotel/state":
		l.rotelState = parseJSONPayload(ev)
		// l.volume = m["volume"].(string)
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
	fmt.Fprintf(w, "<div>")
	for key, value := range l.rotelState {
		fmt.Fprintf(w, "<div>%s: %s</div>", key, value)
	}
	fmt.Fprintf(w, "</div>")
}
