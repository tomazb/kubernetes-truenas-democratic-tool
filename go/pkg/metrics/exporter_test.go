package metrics

import (
	"testing"

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
