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
