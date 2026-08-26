package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TelemetrySample is one reading from an inverter or panel string.
type TelemetrySample struct {
	ArrayID string  `json:"array_id"`
	Kind    string  `json:"kind"`
	Value   float64 `json:"value"`
	At      string  `json:"at"`
}

// TelemetryBatch groups samples collected in one polling interval.
type TelemetryBatch struct {
	Source  string            `json:"source"`
	Samples []TelemetrySample `json:"samples"`
}

// TelemetryStats is a point-in-time view of pipeline activity.
type TelemetryStats struct {
	Processed int64
	Rejected  int64
	Inflight  int64
}

// TelemetryPipeline ingests telemetry batches through a bounded channel and
// processes them with a fixed worker pool. Workers observe the long-lived
// context supplied at Start; per-request contexts only gate the enqueue.
type TelemetryPipeline struct {
	batches   chan TelemetryBatch
	workers   int
	wg        sync.WaitGroup
	metrics   *MetricsCollector
	notifier  *Notifier
	audit     *OpsAudit
	stopped   atomic.Bool
	processed int64
	rejected  int64
	inflight  int64
}

func newTelemetryPipeline(workers int, metrics *MetricsCollector, notifier *Notifier, audit *OpsAudit) *TelemetryPipeline {
	if workers < 1 {
		workers = 1
	}
	return &TelemetryPipeline{
		batches:  make(chan TelemetryBatch),
		workers:  workers,
		metrics:  metrics,
		notifier: notifier,
		audit:    audit,
	}
}

// Start launches the worker pool; cancellation of ctx stops all workers.
func (p *TelemetryPipeline) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *TelemetryPipeline) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-p.batches:
			if !ok {
				return
			}
			p.handle(batch)
		}
	}
}

// Ingest queues a batch for processing, honouring the caller context.
func (p *TelemetryPipeline) Ingest(ctx context.Context, batch TelemetryBatch) error {
	if p.stopped.Load() {
		return fmt.Errorf("%w: pipeline stopped", ErrOpsPolicy)
	}
	if strings.TrimSpace(batch.Source) == "" {
		return ErrOpsInvalid
	}
	atomic.AddInt64(&p.inflight, 1)
	select {
	case p.batches <- batch:
		return nil
	case <-ctx.Done():
		atomic.AddInt64(&p.inflight, -1)
		return ctx.Err()
	}
}

func (p *TelemetryPipeline) handle(batch TelemetryBatch) {
	defer atomic.AddInt64(&p.inflight, -1)
	for _, sample := range batch.Samples {
		if !telemetrySampleValid(sample) {
			atomic.AddInt64(&p.rejected, 1)
			continue
		}
		if err := p.metrics.Record(sample); err != nil {
			atomic.AddInt64(&p.rejected, 1)
			continue
		}
		p.audit.Add("telemetry", "sample_ingested", batch.Source)
		if sample.Value > 85 {
			p.notifier.Enqueue("telemetry", "overheat", fmt.Sprintf("%s reading %.1f", sample.ArrayID, sample.Value))
		}
	}
	atomic.AddInt64(&p.processed, 1)
}

func telemetrySampleValid(sample TelemetrySample) bool {
	return strings.TrimSpace(sample.ArrayID) != "" && sample.Value >= -50 && sample.Value <= 200
}

// Stats returns a snapshot of pipeline counters.
func (p *TelemetryPipeline) Stats() TelemetryStats {
	return TelemetryStats{
		Processed: atomic.LoadInt64(&p.processed),
		Rejected:  atomic.LoadInt64(&p.rejected),
		Inflight:  atomic.LoadInt64(&p.inflight),
	}
}

// WaitPending blocks until all queued batches have been fully processed.
func (p *TelemetryPipeline) WaitPending(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&p.inflight) == 0 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// Close marks the pipeline stopped so no further batches are accepted.
func (p *TelemetryPipeline) Close() {
	p.stopped.Store(true)
}
