package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExporter_ObserveScanDuration(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})

	exporter.ObserveScanDuration(2.5)
	exporter.ObserveScanDuration(5.0)

	families, err := exporter.registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, family := range families {
		if family.GetName() == "truenas_monitor_scan_duration_histogram_seconds" {
			found = true
			require.Equal(t, uint64(2), family.GetMetric()[0].GetHistogram().GetSampleCount())
			require.InDelta(t, 7.5, family.GetMetric()[0].GetHistogram().GetSampleSum(), 0.001)
		}
	}
	require.True(t, found, "scan duration histogram not registered")
}

func TestExporter_ObserveListPhaseDuration(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})

	exporter.ObserveListPhaseDuration("k8s_pvs", 0.25)

	families, err := exporter.registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, family := range families {
		if family.GetName() != "truenas_monitor_list_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "phase" && label.GetValue() == "k8s_pvs" {
					found = true
					require.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
					require.InDelta(t, 0.25, metric.GetHistogram().GetSampleSum(), 0.001)
				}
			}
		}
	}
	require.True(t, found, "list phase histogram sample not found")
}

func TestExporter_RecordInventoryCacheAccess(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})

	exporter.RecordInventoryCacheAccess("k8s_pvcs", false)
	exporter.RecordInventoryCacheAccess("k8s_pvcs", true)

	families, err := exporter.registry.Gather()
	require.NoError(t, err)

	var hits, misses float64
	for _, family := range families {
		switch family.GetName() {
		case "truenas_monitor_inventory_cache_hits_total":
			hits = family.GetMetric()[0].GetCounter().GetValue()
		case "truenas_monitor_inventory_cache_misses_total":
			misses = family.GetMetric()[0].GetCounter().GetValue()
		}
	}
	require.Equal(t, float64(1), hits)
	require.Equal(t, float64(1), misses)
}

func TestExporter_ReconcileMetrics(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})

	exporter.SetReconcileMode("watch")
	exporter.RecordWatchEvent("pv")
	exporter.IncReconcileTrigger("debounce")
	exporter.SetTruenasSnapshotAge(12.5)

	families, err := exporter.registry.Gather()
	require.NoError(t, err)

	var watchCount, triggerCount float64
	var watchMode, pollMode, snapshotAge float64
	for _, family := range families {
		switch family.GetName() {
		case "truenas_monitor_watch_events_total":
			watchCount = family.GetMetric()[0].GetCounter().GetValue()
		case "truenas_monitor_reconcile_triggers_total":
			triggerCount = family.GetMetric()[0].GetCounter().GetValue()
		case "truenas_monitor_reconcile_mode":
			for _, metric := range family.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "mode" && label.GetValue() == "watch" {
						watchMode = metric.GetGauge().GetValue()
					}
					if label.GetName() == "mode" && label.GetValue() == "poll" {
						pollMode = metric.GetGauge().GetValue()
					}
				}
			}
		case "truenas_monitor_truenas_snapshot_age_seconds":
			snapshotAge = family.GetMetric()[0].GetGauge().GetValue()
		}
	}

	require.Equal(t, float64(1), watchCount)
	require.Equal(t, float64(1), triggerCount)
	require.Equal(t, float64(1), watchMode)
	require.Equal(t, float64(0), pollMode)
	require.InDelta(t, 12.5, snapshotAge, 0.001)
}

func TestExporter_RecordPerformanceBudgetStatus(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})
	now := time.Unix(1_700_000_000, 0)

	exporter.RecordPerformanceBudgetStatus("scan_duration", "all", false, now)
	exporter.RecordPerformanceBudgetStatus("scan_duration", "all", true, now)

	families, err := exporter.registry.Gather()
	require.NoError(t, err)

	var breaches, status, lastBreach float64
	for _, family := range families {
		switch family.GetName() {
		case "truenas_monitor_performance_budget_breaches_total":
			breaches = family.GetMetric()[0].GetCounter().GetValue()
		case "truenas_monitor_performance_budget_status":
			status = family.GetMetric()[0].GetGauge().GetValue()
		case "truenas_monitor_performance_budget_last_breach_timestamp":
			lastBreach = family.GetMetric()[0].GetGauge().GetValue()
		}
	}

	require.Equal(t, float64(1), breaches)
	require.Equal(t, float64(1), status)
	require.Equal(t, float64(now.Unix()), lastBreach)
}

func TestExporter_EstimateListPhaseP95(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})
	for i := 0; i < 20; i++ {
		exporter.ObserveListPhaseDuration("k8s_pvs", 0.2)
	}
	exporter.ObserveListPhaseDuration("k8s_pvs", 3.0)

	p95, ok := exporter.EstimateListPhaseP95("k8s_pvs")
	require.True(t, ok)
	require.InDelta(t, 0.25, p95, 0.05)
}

func TestExporter_EstimateListPhasesP95(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})
	for i := 0; i < 20; i++ {
		exporter.ObserveListPhaseDuration("k8s_pvs", 0.2)
	}
	exporter.ObserveListPhaseDuration("k8s_pvs", 3.0)

	p95s := exporter.EstimateListPhasesP95()
	p95, ok := p95s["k8s_pvs"]
	require.True(t, ok)
	require.InDelta(t, 0.25, p95, 0.05)
}

func TestExporter_EstimateListPhasesP95_ReturnsInfinityForOverflow(t *testing.T) {
	exporter := NewExporter(Config{Enabled: true, Port: 0, Path: "/metrics"})
	for i := 0; i < 21; i++ {
		exporter.ObserveListPhaseDuration("k8s_pvs", 120.0)
	}

	p95s := exporter.EstimateListPhasesP95()
	p95, ok := p95s["k8s_pvs"]
	require.True(t, ok)
	require.True(t, math.IsInf(p95, 1))
}
