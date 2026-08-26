package main

import (
	"net/http"
	"os"
)

func newServer(store *WorkStore, ops *opsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/maintenance-tasks", workHandler{store: store})
	mux.HandleFunc("/api/maintenance-tasks/", maintenanceFallback)
	mux.Handle("/api/ops/", ops)
	mux.HandleFunc("/", staticHandler)
	return mux
}

// maintenanceFallback serves the static page for any maintenance sub-path.
func maintenanceFallback(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("web/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
