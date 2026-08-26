package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func seedOpsService() *OpsService {
	return newOpsService([]OpsRecord{
		{ID: "op-001", Owner: "lin", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "A-03", "evidence": "ok"}},
		{ID: "op-002", Owner: "wang", Status: OpsStatusPaused, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "A-04"}},
		{ID: "op-003", Owner: "zhao", Status: OpsStatusQueued, Priority: OpsPriorityLow, Labels: map[string]string{"site": "B-01", "evidence": "ok"}},
	})
}

func waitGroupOrTimeout(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("concurrent workload did not finish")
	}
}

func TestOpsGetConcurrentTouch(t *testing.T) {
	service := seedOpsService()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 80; j++ {
				record, err := service.Get(context.Background(), "op-001")
				if err == nil {
					_ = record.LabelValue("site")
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 80; j++ {
				_ = service.ApplyLabel(context.Background(), "op-001", "site", "A-03")
			}
		}()
	}
	close(start)
	waitGroupOrTimeout(t, &wg)
}

func TestOpsListConcurrentTouch(t *testing.T) {
	service := seedOpsService()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 80; j++ {
				page, _ := service.Search(context.Background(), OpsQuery{PageSize: 50})
				for _, item := range page.Items {
					_ = item.LabelValue("site")
				}
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 80; j++ {
				_ = service.ApplyLabel(context.Background(), "op-002", "region", "north")
			}
		}()
	}
	close(start)
	waitGroupOrTimeout(t, &wg)
}

func TestOpsRecentRequestsBounded(t *testing.T) {
	handler := opsEnterpriseMiddleware(requestIDMiddleware(recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				resp, err := client.Get(server.URL + "/probe")
				if err == nil {
					resp.Body.Close()
				}
			}
		}()
	}
	waitGroupOrTimeout(t, &wg)
	if got := opsRecentRequestCount(); got > 200 {
		t.Fatalf("recent request log grew unbounded: %d", got)
	}
}

func TestOpsReadsReturnCopies(t *testing.T) {
	service := seedOpsService()
	record, err := service.Get(context.Background(), "op-001")
	if err != nil {
		t.Fatal(err)
	}
	record.Labels["tamper"] = "yes"
	again, _ := service.Get(context.Background(), "op-001")
	if got := again.LabelValue("tamper"); got != "" {
		t.Fatalf("store record was mutated through returned reference: %q", got)
	}
}
