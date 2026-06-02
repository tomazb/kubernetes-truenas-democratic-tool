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

	wantOrphan := 48 * time.Hour
	wantRetention := 168 * time.Hour

	svc, err := NewService(Config{
		Logger:            logger,
		ScanInterval:      time.Minute,
		OrphanThreshold:   wantOrphan,
		SnapshotRetention: wantRetention,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	gotOrphan, gotRetention := svc.DetectorThresholds()
	if gotOrphan != wantOrphan {
		t.Fatalf("orphan threshold: got %v want %v", gotOrphan, wantOrphan)
	}
	if gotRetention != wantRetention {
		t.Fatalf("snapshot retention: got %v want %v", gotRetention, wantRetention)
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

	var scanHistFound bool
	var phaseHistFound bool
	for _, family := range families {
		switch family.GetName() {
		case "truenas_monitor_scan_duration_histogram_seconds":
			scanHistFound = true
			if family.GetMetric()[0].GetHistogram().GetSampleCount() != 1 {
				t.Fatalf("expected one scan histogram sample")
			}
		case "truenas_monitor_list_duration_seconds":
			for _, metric := range family.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "phase" && label.GetValue() == "k8s_pvs" {
						phaseHistFound = true
						if metric.GetHistogram().GetSampleCount() != 1 {
							t.Fatalf("expected one phase histogram sample")
						}
					}
				}
			}
		}
	}
	if !scanHistFound {
		t.Fatal("scan histogram not found")
	}
	if !phaseHistFound {
		t.Fatal("phase histogram sample for k8s_pvs not found")
	}
}

func TestService_UpdateMetrics_RecordsPerformanceBudgetBreaches(t *testing.T) {
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	exporter := metrics.NewExporter(metrics.Config{Enabled: true, Port: 0, Path: "/metrics"})
	svc := &Service{
		logger:               logger,
		scanInterval:         time.Minute,
		metricsExporter:      exporter,
		scanBudgetSeconds:    0.001,
		listP95BudgetSeconds: 0.001,
		memoryBudgetMB:       1,
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

	var breachMetricFound bool
	for _, family := range families {
		if family.GetName() == "truenas_monitor_performance_budget_breaches_total" {
			breachMetricFound = true
			if len(family.GetMetric()) == 0 {
				t.Fatalf("expected breach metrics")
			}
		}
	}

	if !breachMetricFound {
		t.Fatal("performance budget breach counter not found")
	}
}

func BenchmarkService_UpdateMetrics(b *testing.B) {
	logger, err := logging.NewLogger(logging.Config{Level: "error", Encoding: "json"})
	if err != nil {
		b.Fatalf("logger: %v", err)
	}
	exporter := metrics.NewExporter(metrics.Config{Enabled: true, Port: 0, Path: "/metrics"})
	svc := &Service{
		logger:               logger,
		metricsExporter:      exporter,
		scanBudgetSeconds:    300,
		listP95BudgetSeconds: 2,
		memoryBudgetMB:       2048,
	}
	result := &ScanResult{
		Timestamp:      time.Now(),
		ScanDuration:   2 * time.Second,
		TotalPVs:       1000,
		TotalPVCs:      1000,
		TotalSnapshots: 1000,
	}
	phaseTimings := map[string]time.Duration{
		"k8s_pvs":           200 * time.Millisecond,
		"k8s_pvcs:*":        250 * time.Millisecond,
		"k8s_snapshots:*":   300 * time.Millisecond,
		"truenas_datasets":  150 * time.Millisecond,
		"truenas_snapshots": 175 * time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.updateMetrics(result, phaseTimings)
	}
}
