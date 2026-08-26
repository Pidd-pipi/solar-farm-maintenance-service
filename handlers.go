package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type workRequest struct {
	Status string `json:"status"`
}

type workHandler struct{ store *WorkStore }

func (h workHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/maintenance-tasks" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"maintenance_tasks": h.store.List()})
		return
	}
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/maintenance-tasks/") {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/maintenance-tasks/"), "/")
	if len(parts) != 2 || parts[1] != "status" || parts[0] == "" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	var request workRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateWorkStatus(request.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := h.store.Update(parts[0], request.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrWorkNotFound):
			writeError(w, http.StatusNotFound, "maintenance task not found")
		case errors.Is(err, ErrWorkTransition):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, order)
}
