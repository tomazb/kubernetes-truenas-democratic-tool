package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
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
	})
}

func TestService_Stop_NilExporterWhenNotRunning(t *testing.T) {
	svc := &Service{metricsExporter: nil}
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
