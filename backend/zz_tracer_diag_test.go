package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestTracerStaysBoundedDiagnosis(t *testing.T) {
	tracer := newOpsActivityLog(5)
	for i := 0; i < 500; i++ {
		tracer.Record(traceEntryFor("r", "/probe", int64(i)))
	}
	if got := tracer.Len(); got != 5 {
		t.Fatalf("tracer len = %d, want 5 (unbounded growth)", got)
	}
}

func TestTracerRecentReturnsCopy(t *testing.T) {
	tracer := newOpsActivityLog(20)
	for i := 0; i < 10; i++ {
		tracer.Record(traceEntryFor("r", "/probe", int64(i)))
	}
	recent := tracer.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("recent len = %d", len(recent))
	}
	recent[0].RequestID = "tampered"
	again := tracer.Recent(3)
	if again[0].RequestID == "tampered" {
		t.Fatalf("Recent returned internal slice; mutation leaked into tracer")
	}
}

func TestGlobalTracerBoundedViaHTTP(t *testing.T) {
	server := httptest.NewServer(newServer(newWorkStore(), newOpsHandler()))
	defer server.Close()
	client := &http.Client{}
	for i := 0; i < 2000; i++ {
		resp, err := client.Get(server.URL + "/healthz")
		if err == nil {
			resp.Body.Close()
		}
	}
	if got := globalOpsActivityLog.Len(); got > 1500 {
		t.Fatalf("global tracer grew unbounded through HTTP entry points: %d", got)
	}
}

func TestStaticCacheBounded(t *testing.T) {
	server := httptest.NewServer(newServer(newWorkStore(), newOpsHandler()))
	defer server.Close()
	client := &http.Client{}
	for i := 0; i < 500; i++ {
		resp, err := client.Get(server.URL + "/app.js?v=" + strconv.Itoa(i))
		if err == nil {
			resp.Body.Close()
		}
	}
	staticCacheMu.Lock()
	size := len(staticCache)
	staticCacheMu.Unlock()
	if size > 10 {
		t.Fatalf("static cache grew unbounded: %d entries", size)
	}
}

func TestHealthProbeHistoryBounded(t *testing.T) {
	server := httptest.NewServer(newServer(newWorkStore(), newOpsHandler()))
	defer server.Close()
	client := &http.Client{}
	for i := 0; i < 1000; i++ {
		resp, err := client.Get(server.URL + "/healthz")
		if err == nil {
			resp.Body.Close()
		}
	}
	if got := healthProbeCount(); got > 300 {
		t.Fatalf("health probe history grew unbounded: %d", got)
	}
}
