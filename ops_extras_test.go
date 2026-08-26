package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelemetryPipelineIngest(t *testing.T) {
	audit := newOpsAudit()
	metrics := newMetricsCollector(100)
	notifier := newNotifier(3)
	pipeline := newTelemetryPipeline(2, metrics, notifier, audit)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)

	batch := TelemetryBatch{Source: "gateway-1", Samples: []TelemetrySample{
		{ArrayID: "A-01", Kind: "inverter", Value: 88, At: timeNowOps()},
		{ArrayID: "A-02", Kind: "inverter", Value: 60, At: timeNowOps()},
		{ArrayID: "", Kind: "inverter", Value: 70, At: timeNowOps()},
	}}
	if err := pipeline.Ingest(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if !pipeline.WaitPending(3 * time.Second) {
		t.Fatal("pipeline did not drain")
	}
	stats := pipeline.Stats()
	if stats.Processed != 1 {
		t.Fatalf("processed = %d", stats.Processed)
	}
	if stats.Rejected != 1 {
		t.Fatalf("rejected = %d", stats.Rejected)
	}
	summary := metrics.Summary()
	if len(summary) != 2 {
		t.Fatalf("tracked arrays = %d", len(summary))
	}
	if notifier.Pending() != 1 {
		t.Fatalf("alarm notifications = %d", notifier.Pending())
	}
}

func TestSchedulePlanBuild(t *testing.T) {
	records := []OpsRecord{
		{ID: "op-001", Owner: "lin", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "A-03"}},
		{ID: "op-002", Owner: "wang", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Labels: map[string]string{}},
		{ID: "op-003", Owner: "zhao", Status: OpsStatusClosed, Priority: OpsPriorityLow, Labels: map[string]string{"site": "B-01"}},
	}
	settings := loadSettings()
	plan, err := buildSchedulePlan(records, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 2 {
		t.Fatalf("entries = %d", len(plan.Entries))
	}
	if plan.Entries[1].ArrayID != settings.LabelDefaults["site"] {
		t.Fatalf("default site not applied: %q", plan.Entries[1].ArrayID)
	}
	if plan.ByTech["lin"] != 1 {
		t.Fatalf("technician count = %v", plan.ByTech)
	}
}

func TestComplianceTerminalRule(t *testing.T) {
	record := OpsRecord{ID: "op-001", Status: OpsStatusActive, Labels: map[string]string{"site": "A-03", "operator": "lin", "evidence": "ok"}}
	report, err := checkRecordCompliance(record, opsRules())
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed == 0 {
		t.Fatalf("expected terminal rules to fail for active record, report passed=%d failed=%d", report.Passed, report.Failed)
	}
	result, ok := complianceByCode(report, "OPS-0104")
	if !ok {
		t.Fatal("OPS-0104 missing from report")
	}
	if result.Passed {
		t.Fatalf("OPS-0104 is terminal but record is not closed: %+v", result)
	}
}

func TestNotifierDispatchContext(t *testing.T) {
	notifier := newNotifier(3)
	notifier.Enqueue("op-001", "overheat", "high temp")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	sent := 0
	err := notifier.Dispatch(ctx, func(item Notification) error {
		sent++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sent != 1 || notifier.Pending() != 0 {
		t.Fatalf("sent=%d pending=%d", sent, notifier.Pending())
	}
}

func TestMetricsCollectorCapacity(t *testing.T) {
	collector := newMetricsCollector(2)
	for _, id := range []string{"A-01", "A-02", "A-03"} {
		_ = collector.Record(TelemetrySample{ArrayID: id, Value: 10, At: timeNowOps()})
	}
	if collector.Count() != 2 {
		t.Fatalf("count = %d", collector.Count())
	}
}

func TestOpsTracerBounded(t *testing.T) {
	tracer := newOpsTracer(5)
	for i := 0; i < 50; i++ {
		tracer.Record(traceEntryFor("r", "/x", int64(i)))
	}
	if tracer.Len() != 5 {
		t.Fatalf("len = %d", tracer.Len())
	}
	recent := tracer.Recent(10)
	if len(recent) != 5 {
		t.Fatalf("recent = %d", len(recent))
	}
}

func TestOpsHTTPFlow(t *testing.T) {
	handler := newOpsHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.Get(server.URL + "/api/ops/records")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list records: got %d", response.StatusCode)
	}

	create := func(body string) int {
		response, err := http.Post(server.URL+"/api/ops/records", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if got := create(`{"subject":"test job","owner":"lin","status":"queued","priority":"high","labels":{"site":"A-09"}}`); got != http.StatusCreated {
		t.Fatalf("create record: got %d", got)
	}
	if got := create(`{"subject":"dup","owner":"lin","status":"queued","priority":"high","labels":{"site":"A-09"}}`); got != http.StatusConflict {
		t.Fatalf("duplicate create should conflict, got %d", got)
	}
	if got := create(`{"subject":"dup"`); got != http.StatusBadRequest {
		t.Fatalf("bad json should 400, got %d", got)
	}

	transition := func(id, body string) int {
		response, err := http.Post(server.URL+"/api/ops/records/"+id+"/transition", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if got := transition("op-001", `{"target":"paused","expected":0,"actor":"lin"}`); got != http.StatusOK {
		t.Fatalf("transition: got %d", got)
	}
	if got := transition("missing", `{"target":"paused","expected":0,"actor":"lin"}`); got != http.StatusNotFound {
		t.Fatalf("missing record transition: got %d", got)
	}
	if got := transition("op-001", `{"target":"bogus","expected":0,"actor":"lin"}`); got != http.StatusBadRequest {
		t.Fatalf("bogus target transition: got %d", got)
	}
}

func TestAuditRetentionBehavior(t *testing.T) {
	audit := newOpsAudit()
	for i := 0; i < 300; i++ {
		audit.Add("r-1", "created", "lin")
	}
	audit.Trim(50)
	if audit.Count() != 50 {
		t.Fatalf("audit trim count = %d, want 50", audit.Count())
	}
	audit.Clear()
	if audit.Capacity() > 64 {
		t.Fatalf("audit capacity not reclaimed after Clear: %d", audit.Capacity())
	}
}

func TestStateHistoryRetentionBehavior(t *testing.T) {
	machine := newOpsStateMachine()
	moves := [][2]OpsStatus{
		{OpsStatusQueued, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
		{OpsStatusPaused, OpsStatusActive},
		{OpsStatusActive, OpsStatusPaused},
	}
	for i := 0; i < 300; i++ {
		pair := moves[i%len(moves)]
		if err := machine.Move(pair[0], pair[1], "cycle"); err != nil {
			t.Fatal(err)
		}
	}
	machine.TrimHistory(50)
	if got := len(machine.History()); got != 50 {
		t.Fatalf("history trim count = %d, want 50", got)
	}
	machine.Reset()
	if got := machine.Capacity(); got > 64 {
		t.Fatalf("history capacity not reclaimed after Reset: %d", got)
	}
}
