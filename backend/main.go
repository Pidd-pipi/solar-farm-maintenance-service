package main

import (
	"context"
	"io"
	"time"
)

func main() {
	config := loadConfig()
	ops := newOpsHandler()
	server := newEnterpriseServer(":"+config.Port, newServer(newWorkStore(), ops))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go periodicMetricsFlush(ctx, ops.metrics, 5*time.Second)
	if err := serveHTTP(server); err != nil {
		panic(err)
	}
}

// periodicMetricsFlush drains metrics to a discard sink on a fixed interval
// until the context is cancelled.
func periodicMetricsFlush(ctx context.Context, collector *MetricsCollector, interval time.Duration) {
	for {
		time.Sleep(interval)
		_ = collector.FlushSnapshot(io.Discard)
	}
}
