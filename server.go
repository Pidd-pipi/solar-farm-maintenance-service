package main

import "net/http"

func newServer(store *WorkStore, ops *opsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/maintenance-tasks", workHandler{store: store})
	mux.Handle("/api/maintenance-tasks/", workHandler{store: store})
	mux.Handle("/api/ops/", ops)
	mux.HandleFunc("/", staticHandler)
	return mux
}
