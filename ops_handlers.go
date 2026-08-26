package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// opsHandler exposes the maintenance operations domain over HTTP.
type opsHandler struct {
	service  *OpsService
	notifier *Notifier
	metrics  *MetricsCollector
	tracer   *OpsTracer
	pipeline *TelemetryPipeline
}

// newOpsHandler builds the operations stack with sensible defaults.
func newOpsHandler() *opsHandler {
	seed := []OpsRecord{
		{ID: "op-001", Subject: "inverter a03 maintenance", Owner: "lin", Status: OpsStatusActive, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "A-03", "evidence": "ok"}},
		{ID: "op-002", Subject: "panel cleaning batch", Owner: "wang", Status: OpsStatusQueued, Priority: OpsPriorityNormal, Labels: map[string]string{"site": "A-04"}},
		{ID: "op-003", Subject: "combiner box inspection", Owner: "zhao", Status: OpsStatusPaused, Priority: OpsPriorityLow, Labels: map[string]string{"site": "B-01", "evidence": "ok"}},
	}
	service := newOpsService(seed)
	notifier := newNotifier(3)
	metrics := newMetricsCollector(200)
	tracer := newOpsTracer(1000)
	pipeline := newTelemetryPipeline(2, metrics, notifier, service.audit)
	pipeline.Start(context.Background())
	return &opsHandler{service: service, notifier: notifier, metrics: metrics, tracer: tracer, pipeline: pipeline}
}

func (h *opsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.tracer.Record(traceEntryFor(opsRequestID(r), r.URL.Path, 0))
	path := strings.TrimPrefix(r.URL.Path, "/api/ops")
	switch {
	case path == "/records" && r.Method == http.MethodGet:
		h.listRecords(w, r)
	case path == "/records" && r.Method == http.MethodPost:
		h.createRecord(w, r)
	case strings.HasPrefix(path, "/records/") && strings.HasSuffix(path, "/transition") && r.Method == http.MethodPost:
		h.transitionRecord(w, r)
	case strings.HasPrefix(path, "/records/") && strings.HasSuffix(path, "/audit") && r.Method == http.MethodGet:
		h.recordAudit(w, r)
	case strings.HasPrefix(path, "/records/") && strings.HasSuffix(path, "/compliance") && r.Method == http.MethodGet:
		h.recordCompliance(w, r)
	case path == "/telemetry" && r.Method == http.MethodPost:
		h.ingestTelemetry(w, r)
	case path == "/metrics" && r.Method == http.MethodGet:
		h.metricsSummary(w, r)
	case path == "/schedule" && r.Method == http.MethodGet:
		h.schedulePlan(w, r)
	case path == "/health" && r.Method == http.MethodGet:
		h.opsHealth(w, r)
	default:
		opsJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
	}
}

type createRecordRequest struct {
	Subject  string            `json:"subject"`
	Owner    string            `json:"owner"`
	Status   string            `json:"status"`
	Priority string            `json:"priority"`
	Labels   map[string]string `json:"labels"`
}

func (h *opsHandler) listRecords(w http.ResponseWriter, r *http.Request) {
	query := OpsQuery{Subject: r.URL.Query().Get("subject"), Owner: r.URL.Query().Get("owner")}
	if value := r.URL.Query().Get("status"); value != "" {
		query.Status = OpsStatus(value)
	}
	page, err := h.service.Search(r.Context(), query)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, map[string]any{"page": page})
}

func (h *opsHandler) createRecord(w http.ResponseWriter, r *http.Request) {
	var request createRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	record := OpsRecord{
		Subject:  request.Subject,
		Owner:    request.Owner,
		Status:   OpsStatus(request.Status),
		Priority: OpsPriority(request.Priority),
		Labels:   request.Labels,
	}
	created, err := h.service.Create(r.Context(), record)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusCreated, created)
}

type transitionRequest struct {
	Target   string `json:"target"`
	Expected int    `json:"expected"`
	Actor    string `json:"actor"`
}

func (h *opsHandler) transitionRecord(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ops/records/"), "/")
	id = strings.TrimSuffix(id, "/transition")
	var request transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	record, err := h.service.Transition(r.Context(), id, request.Expected, OpsStatus(request.Target), request.Actor)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, record)
}

func (h *opsHandler) recordAudit(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ops/records/"), "/")
	id = strings.TrimSuffix(id, "/audit")
	events := h.service.Audit(id)
	opsJSON(w, http.StatusOK, map[string]any{"record_id": id, "events": events})
}

func (h *opsHandler) recordCompliance(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/ops/records/"), "/")
	id = strings.TrimSuffix(id, "/compliance")
	record, err := h.service.Get(r.Context(), id)
	if err != nil {
		opsWriteError(w, err)
		return
	}
	report, err := checkRecordCompliance(record, opsRules())
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, report)
}

func (h *opsHandler) ingestTelemetry(w http.ResponseWriter, r *http.Request) {
	var batch TelemetryBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		opsJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.pipeline.Ingest(ctx, batch); err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusAccepted, map[string]any{"accepted": true, "samples": len(batch.Samples)})
}

func (h *opsHandler) metricsSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.metrics.Summary()
	opsJSON(w, http.StatusOK, map[string]any{"arrays": summary, "count": len(summary), "traced": h.tracer.Len()})
}

func (h *opsHandler) schedulePlan(w http.ResponseWriter, r *http.Request) {
	records, err := h.service.store.List(r.Context())
	if err != nil {
		opsWriteError(w, err)
		return
	}
	plan, err := buildSchedulePlan(records, loadSettings())
	if err != nil {
		opsWriteError(w, err)
		return
	}
	opsJSON(w, http.StatusOK, plan)
}

func (h *opsHandler) opsHealth(w http.ResponseWriter, r *http.Request) {
	stats := h.pipeline.Stats()
	opsJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"records":   h.service.Count(),
		"audit":     h.service.audit.Count(),
		"telemetry": stats,
		"traced":    h.tracer.Len(),
		"notify":    h.notifier.Pending(),
	})
}

// opsStatusForError maps a domain error to the correct HTTP status.
func opsStatusForError(err error) int {
	switch opsCode(err) {
	case "not_found":
		return http.StatusNotFound
	case "conflict":
		return http.StatusConflict
	case "invalid":
		return http.StatusBadRequest
	case "transition":
		return http.StatusBadRequest
	case "policy":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func opsWriteError(w http.ResponseWriter, err error) {
	opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
}
