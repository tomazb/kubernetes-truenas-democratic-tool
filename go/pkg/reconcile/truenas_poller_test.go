package reconcile

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

type fakeTruenasClient struct {
	volumeCalls   atomic.Int32
	snapshotCalls atomic.Int32
	volumes       []truenas.Volume
	snapshots     []truenas.Snapshot
}

func (f *fakeTruenasClient) ListVolumes(context.Context) ([]truenas.Volume, error) {
	f.volumeCalls.Add(1)
	return append([]truenas.Volume(nil), f.volumes...), nil
}

func (f *fakeTruenasClient) ListSnapshots(context.Context) ([]truenas.Snapshot, error) {
	f.snapshotCalls.Add(1)
	return append([]truenas.Snapshot(nil), f.snapshots...), nil
}

func (f *fakeTruenasClient) ListPools(context.Context) ([]truenas.Pool, error) {
	return nil, nil
}

func (f *fakeTruenasClient) GetSystemInfo(context.Context) (*truenas.SystemInfo, error) {
	return &truenas.SystemInfo{}, nil
}

func (f *fakeTruenasClient) TestConnection(context.Context) error {
	return nil
}

func TestTruenasPollerUpdatesSnapshot(t *testing.T) {
	inner := &fakeTruenasClient{
		volumes:   []truenas.Volume{{Name: "vol-a"}},
		snapshots: []truenas.Snapshot{{Name: "snap-a"}},
	}
	poller := NewTruenasPoller(inner, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go poller.Run(ctx)

	require.Eventually(t, func() bool {
		snap := poller.Snapshot()
		return !snap.UpdatedAt.IsZero() && len(snap.Volumes) == 1
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, int32(1), inner.volumeCalls.Load())
	assert.Equal(t, int32(1), inner.snapshotCalls.Load())

	wrapped := WrapTruenasWithSnapshot(inner, poller)
	vols, err := wrapped.ListVolumes(ctx)
	require.NoError(t, err)
	assert.Equal(t, "vol-a", vols[0].Name)
	assert.Equal(t, int32(1), inner.volumeCalls.Load())
}
