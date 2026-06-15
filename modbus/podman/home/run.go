package home

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("modbus/podman/home/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, nil)
}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	filePath := filepath.Join("modbus/podman/home", filepath.Base(r.URL.Path))
	http.ServeFile(w, r, filePath)
}
