package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

var (
	healthProbeMu      sync.Mutex
	healthProbeHistory []string
	healthProbeCap     = 200
)

// healthProbeCount returns how many health probes are retained.
func healthProbeCount() int {
	healthProbeMu.Lock()
	defer healthProbeMu.Unlock()
	return len(healthProbeHistory)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	globalOpsTracer.Record(traceEntryFor(opsRequestID(r), r.URL.Path, 0))
	healthProbeMu.Lock()
	healthProbeHistory = append(healthProbeHistory, r.URL.Path)
	if len(healthProbeHistory) > healthProbeCap {
		healthProbeHistory = append([]string(nil), healthProbeHistory[len(healthProbeHistory)-healthProbeCap:]...)
	}
	healthProbeMu.Unlock()
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "solar-farm-maintenance"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
