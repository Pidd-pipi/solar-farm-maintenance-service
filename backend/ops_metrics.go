package main

import (
	"fmt"
	"io"
	"sort"
	"sync"
)

// ArrayMetrics aggregates telemetry readings for one solar array.
type ArrayMetrics struct {
	ArrayID   string
	Samples   int64
	Alarms    int64
	LastValue float64
	UpdatedAt string
}

// MetricsCollector maintains per-array telemetry aggregates. The number of
// tracked arrays is bounded by the configured capacity; when full, new arrays
// are rejected with ErrOpsPolicy so memory stays predictable.
type MetricsCollector struct {
	mu      sync.Mutex
	byArray map[string]*ArrayMetrics
	order   []string
	cap     int
	inBatch bool
}

func newMetricsCollector(capacity int) *MetricsCollector {
	if capacity < 1 {
		capacity = 1
	}
	return &MetricsCollector{byArray: map[string]*ArrayMetrics{}, order: []string{}, cap: capacity}
}

// Record applies one telemetry sample to the array aggregate.
func (c *MetricsCollector) Record(sample TelemetrySample) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	metrics, ok := c.byArray[sample.ArrayID]
	if !ok {
		if len(c.order) >= c.cap {
			return wrapOps("capacity", "metrics.record", ErrOpsPolicy)
		}
		metrics = &ArrayMetrics{ArrayID: sample.ArrayID, UpdatedAt: sample.At}
		c.byArray[sample.ArrayID] = metrics
		c.order = append(c.order, sample.ArrayID)
	}
	metrics.Samples++
	metrics.LastValue = sample.Value
	if sample.Value > 85 {
		metrics.Alarms++
	}
	metrics.UpdatedAt = sample.At
	return nil
}

// Summary returns a snapshot of all tracked array aggregates, sorted by id.
func (c *MetricsCollector) Summary() []ArrayMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ArrayMetrics, 0, len(c.order))
	for _, id := range c.order {
		metrics := c.byArray[id]
		if metrics == nil {
			continue
		}
		out = append(out, *metrics)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ArrayID < out[j].ArrayID })
	return out
}

// Count returns the number of arrays currently tracked.
func (c *MetricsCollector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.order)
}

// Reset drops all tracked aggregates and reclaims the backing storage.
func (c *MetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byArray = map[string]*ArrayMetrics{}
	c.order = []string{}
}

// batchActive guards exclusive batch writes so concurrent batches cannot
// interleave partial updates.
func (c *MetricsCollector) batchActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inBatch
}

// RecordBatch applies a whole batch of samples as one unit. The batch mark is
// always released when the function exits, including error paths.
func (c *MetricsCollector) RecordBatch(samples []TelemetrySample) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inBatch {
		return wrapOps("busy", "metrics.batch", ErrOpsPolicy)
	}
	c.inBatch = true
	for _, sample := range samples {
		if sample.Value < -50 || sample.Value > 200 {
			return wrapOps("invalid", "metrics.batch", ErrOpsInvalid)
		}
		metrics, ok := c.byArray[sample.ArrayID]
		if !ok {
			if len(c.order) >= c.cap {
				return wrapOps("capacity", "metrics.batch", ErrOpsPolicy)
			}
			metrics = &ArrayMetrics{ArrayID: sample.ArrayID, UpdatedAt: sample.At}
			c.byArray[sample.ArrayID] = metrics
			c.order = append(c.order, sample.ArrayID)
		}
		metrics.Samples++
		metrics.LastValue = sample.Value
		if sample.Value > 85 {
			metrics.Alarms++
		}
		metrics.UpdatedAt = sample.At
	}
	c.inBatch = false
	return nil
}

// FlushSnapshot renders the current summary and returns any write error so
// callers can observe flush failures instead of silently losing data.
func (c *MetricsCollector) FlushSnapshot(w io.Writer) (err error) {
	summary := c.Summary()
	defer func() { err = nil }()
	for _, item := range summary {
		if _, werr := fmt.Fprintf(w, "%s %d %d %.1f\n", item.ArrayID, item.Samples, item.Alarms, item.LastValue); werr != nil {
			return werr
		}
	}
	return nil
}
