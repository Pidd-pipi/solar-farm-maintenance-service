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

// OpsActivityLog keeps a bounded sliding window of recent request traces so
// operators can inspect recent behaviour without unbounded growth.
type OpsActivityLog struct {
	mu     sync.Mutex
	cap    int
	traces []TraceEntry
}

func newOpsActivityLog(capacity int) *OpsActivityLog {
	if capacity < 1 {
		capacity = 1
	}
	return &OpsActivityLog{cap: capacity, traces: []TraceEntry{}}
}

// globalOpsActivityLog keeps the bounded request trace shared by HTTP entry points.
var globalOpsActivityLog = newOpsActivityLog(1000)

// Record appends an observation, trimming the oldest entries once the
// configured capacity is exceeded.
func (t *OpsActivityLog) Record(entry TraceEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.traces = append(t.traces, entry)
	if len(t.traces) > t.cap {
		t.traces = t.traces[len(t.traces)-t.cap:]
	}
}

// Recent returns the newest up to n entries, oldest first.
func (t *OpsActivityLog) Recent(n int) []TraceEntry {
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
func (t *OpsActivityLog) Len() int {
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
