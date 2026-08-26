package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func startLifecycleServer() *httptest.Server {
	handler := newServer(newWorkStore(), newOpsHandler())
	return httptest.NewServer(newEnterpriseServer("", handler).Handler)
}

func TestWorkOrderStatusRouteJSON(t *testing.T) {
	server := startLifecycleServer()
	defer server.Close()
	response, err := http.Post(server.URL+"/api/maintenance-tasks/WO-301/status", "application/json", strings.NewReader(`{"status":"scheduled"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status update should be 200, got %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("status route should serve JSON, got Content-Type %q", contentType)
	}
}

func TestWorkOrderUpdatePersists(t *testing.T) {
	store := newWorkStore()
	_, exists, changed := store.Update("WO-301", "scheduled")
	if !exists || !changed {
		t.Fatalf("update failed: exists=%v changed=%v", exists, changed)
	}
	status := ""
	for _, order := range store.List() {
		if order.ID == "WO-301" {
			status = order.Status
		}
	}
	if status != "scheduled" {
		t.Fatalf("WO-301 status = %q after Update; change was not persisted", status)
	}
}

func TestRequestIDEchoedInResponse(t *testing.T) {
	server := startLifecycleServer()
	defer server.Close()
	request, err := http.NewRequest(http.MethodGet, server.URL+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Request-ID", "req-abc-123")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got := response.Header.Get("X-Request-ID"); got != "req-abc-123" {
		t.Fatalf("X-Request-ID not echoed: %q", got)
	}
}

func TestRecoveryReturnsServerError(t *testing.T) {
	wrapped := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	server := httptest.NewServer(wrapped)
	defer server.Close()
	response, err := http.Get(server.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("recovered panic should be 500, got %d", response.StatusCode)
	}
}
