package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func startWorkServer() *httptest.Server {
	return httptest.NewServer(newServer(newWorkStore(), newOpsHandler()))
}

func postStatus(t *testing.T, server *httptest.Server, id, status string) int {
	t.Helper()
	response, err := http.Post(server.URL+"/api/maintenance-tasks/"+id+"/status", "application/json", strings.NewReader(`{"status":"`+status+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	return response.StatusCode
}

func listStatus(t *testing.T, server *httptest.Server, id string) string {
	t.Helper()
	response, err := http.Get(server.URL + "/api/maintenance-tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload struct {
		MaintenanceTasks []WorkOrder `json:"maintenance_tasks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, order := range payload.MaintenanceTasks {
		if order.ID == id {
			return order.Status
		}
	}
	return ""
}

func TestWorkOrderNoSkipTransition(t *testing.T) {
	server := startWorkServer()
	defer server.Close()
	if got := postStatus(t, server, "WO-301", "completed"); got != http.StatusConflict {
		t.Fatalf("open->completed should be 409, got %d", got)
	}
}

func TestWorkOrderScheduledAdvances(t *testing.T) {
	server := startWorkServer()
	defer server.Close()
	if got := postStatus(t, server, "WO-302", "in_progress"); got != http.StatusOK {
		t.Fatalf("scheduled->in_progress should be 200, got %d", got)
	}
	if got := listStatus(t, server, "WO-302"); got != "in_progress" {
		t.Fatalf("WO-302 status = %q after advance", got)
	}
}

func TestWorkOrderInvalidStatusRejected(t *testing.T) {
	server := startWorkServer()
	defer server.Close()
	if got := postStatus(t, server, "WO-301", "bogus"); got != http.StatusBadRequest {
		t.Fatalf("invalid status should be 400, got %d", got)
	}
	if got := listStatus(t, server, "WO-301"); got != "open" {
		t.Fatalf("WO-301 status changed to %q", got)
	}
}
