package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

//go:embed templates/hello.html
var content embed.FS

type PageVariables struct {
	Name string
}

func inithttp() {

	http.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		data, err := content.ReadFile("hello.html")
		if err != nil {
			http.Error(w, "Unable to read embedded content", http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/sources", sourcesHandler)

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)

}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	pageVariables := PageVariables{
		Name: name,
	}

	data, readErr := content.ReadFile("templates/hello.html")
	if readErr != nil {
		http.Error(w, "Failed to read embedded template", http.StatusInternalServerError)
		return
	}

	t, parseErr := template.New("hello").Parse(string(data))
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

func sourcesHandler(w http.ResponseWriter, r *http.Request) {

	sources := []string{"opt1", "aux1", "opt2", "aux2"}
	for _, source := range sources {
		fmt.Fprintf(w, "<option value='%s'>%s</option>", source, strings.ToUpper(source))
	}
}
