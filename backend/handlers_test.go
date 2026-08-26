package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkHTTPFlow(t *testing.T) {
	ts := httptest.NewServer(newServer(newWorkStore(), newOpsHandler()))
	defer ts.Close()
	for _, path := range []string{"/healthz", "/api/maintenance-tasks", "/", "/app.js"} {
		response, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: got %d", path, response.StatusCode)
		}
	}
	post := func(id, body string) int {
		response, err := http.Post(ts.URL+"/api/maintenance-tasks/"+id+"/status", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if got := post("WO-301", `{"status":"scheduled"}`); got != http.StatusOK {
		t.Fatalf("valid status: got %d", got)
	}
	if got := post("WO-301", `{"status":"unknown"}`); got != http.StatusBadRequest {
		t.Fatalf("bad status: got %d", got)
	}
	if got := post("WO-301", `{"status":"completed"}`); got != http.StatusConflict {
		t.Fatalf("invalid jump: got %d", got)
	}
	if got := post("missing", `{"status":"scheduled"}`); got != http.StatusNotFound {
		t.Fatalf("missing maintenance task: got %d", got)
	}
}
