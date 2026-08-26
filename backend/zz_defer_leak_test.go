package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) { return 0, errors.New("disk full") }

func TestRecordBatchReleaseAfterError(t *testing.T) {
	collector := newMetricsCollector(10)
	invalid := []TelemetrySample{{ArrayID: "A-01", Value: 500, At: timeNowOps()}}
	if err := collector.RecordBatch(invalid); err == nil {
		t.Fatal("expected invalid batch error")
	}
	valid := []TelemetrySample{{ArrayID: "A-01", Value: 60, At: timeNowOps()}}
	if err := collector.RecordBatch(valid); err != nil {
		t.Fatalf("second batch rejected after first error: %v", err)
	}
	if summary := collector.Summary(); len(summary) != 1 {
		t.Fatalf("expected one tracked array, got %d", len(summary))
	}
}

func TestFlushSnapshotPreservesError(t *testing.T) {
	collector := newMetricsCollector(10)
	_ = collector.Record(TelemetrySample{ArrayID: "A-01", Value: 60, At: timeNowOps()})
	err := collector.FlushSnapshot(failingWriter{})
	if err == nil {
		t.Fatal("flush error was swallowed")
	}
}

func TestPeriodicFlushStopsOnCancel(t *testing.T) {
	collector := newMetricsCollector(10)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		periodicMetricsFlush(ctx, collector, 20*time.Millisecond)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic flush kept running after cancel")
	}
}

func TestRecordBatchRejectsOverCapacity(t *testing.T) {
	collector := newMetricsCollector(2)
	batch := []TelemetrySample{
		{ArrayID: "A-01", Value: 60, At: timeNowOps()},
		{ArrayID: "A-02", Value: 60, At: timeNowOps()},
		{ArrayID: "A-03", Value: 60, At: timeNowOps()},
	}
	if err := collector.RecordBatch(batch); err == nil {
		t.Fatal("expected capacity error for batch exceeding cap")
	}
	if collector.Count() != 2 {
		t.Fatalf("count = %d", collector.Count())
	}
}
