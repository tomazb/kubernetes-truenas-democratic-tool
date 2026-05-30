package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/metrics"
)

func TestService_UpdateMetrics_NilExporterDoesNotPanic(t *testing.T) {
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	svc := &Service{
		logger:          logger,
		scanInterval:    time.Minute,
		metricsExporter: nil,
	}

	svc.updateMetrics(&ScanResult{
		Timestamp: time.Now(),
		TotalPVs:  1,
	}, nil)
}

func TestService_Stop_NilExporterWhenNotRunning(t *testing.T) {
	svc := &Service{metricsExporter: nil}
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestNewService_UsesConfiguredThresholds(t *testing.T) {
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	svc, err := NewService(Config{
		Logger:            logger,
		ScanInterval:      time.Minute,
		OrphanThreshold:   48 * time.Hour,
		SnapshotRetention: 168 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if svc.orphanDetector == nil {
		t.Fatal("expected orphan detector")
	}
}

func TestService_UpdateMetrics_RecordsHistogram(t *testing.T) {
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	exporter := metrics.NewExporter(metrics.Config{Enabled: true, Port: 0, Path: "/metrics"})
	svc := &Service{
		logger:          logger,
		scanInterval:    time.Minute,
		metricsExporter: exporter,
	}

	svc.updateMetrics(&ScanResult{
		Timestamp:    time.Now(),
		ScanDuration: 2 * time.Second,
		TotalPVs:     3,
	}, map[string]time.Duration{"k8s_pvs": 500 * time.Millisecond})

	families, err := exporter.GatherForTest()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	var histFound bool
	for _, family := range families {
		if family.GetName() == "truenas_monitor_scan_duration_histogram_seconds" {
			histFound = true
			if family.GetMetric()[0].GetHistogram().GetSampleCount() != 1 {
				t.Fatalf("expected one histogram sample")
			}
		}
	}
	if !histFound {
		t.Fatal("scan histogram not found")
	}
}
