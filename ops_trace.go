package main

import (
	"sync"
	"time"
)

// TraceEntry records one request observation for operational tracing.
type TraceEntry struct {
	RequestID string
	Path      string
	LatencyMs int64
	At        string
}

// OpsTracer keeps a bounded sliding window of recent request traces so
// operators can inspect recent behaviour without unbounded growth.
type OpsTracer struct {
	mu     sync.Mutex
	cap    int
	traces []TraceEntry
}

func newOpsTracer(capacity int) *OpsTracer {
	if capacity < 1 {
		capacity = 1
	}
	return &OpsTracer{cap: capacity, traces: []TraceEntry{}}
}

// globalOpsTracer keeps the bounded request trace shared by HTTP entry points.
var globalOpsTracer = newOpsTracer(1000)

// Record appends an observation, trimming the oldest entries once the
// configured capacity is exceeded.
func (t *OpsTracer) Record(entry TraceEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.traces = append(t.traces, entry)
	if len(t.traces) > t.cap {
		t.traces = t.traces[len(t.traces)-t.cap:]
	}
}

// Recent returns the newest up to n entries, oldest first.
func (t *OpsTracer) Recent(n int) []TraceEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n < 1 || len(t.traces) == 0 {
		return []TraceEntry{}
	}
	if n > len(t.traces) {
		n = len(t.traces)
	}
	out := make([]TraceEntry, n)
	copy(out, t.traces[len(t.traces)-n:])
	return out
}

// Len returns the number of entries currently retained.
func (t *OpsTracer) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.traces)
}

func traceEntryFor(requestID, path string, latencyMs int64) TraceEntry {
	return TraceEntry{
		RequestID: requestID,
		Path:      path,
		LatencyMs: latencyMs,
		At:        time.Now().UTC().Format(time.RFC3339Nano),
	}
}
