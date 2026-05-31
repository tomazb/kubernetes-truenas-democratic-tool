package inventorycache

import (
	"context"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
)

const (
	opTrueNASDatasets  = "truenas_datasets"
	opTrueNASSnapshots = "truenas_snapshots"
)

type cachedTrueNASClient struct {
	base  truenas.Client
	cache *Cache
}

// WrapTrueNASClient returns a caching decorator when enabled, otherwise the base client.
func WrapTrueNASClient(base truenas.Client, cache *Cache) truenas.Client {
	if cache == nil || !cache.Enabled() {
		return base
	}
	return &cachedTrueNASClient{base: base, cache: cache}
}

func (c *cachedTrueNASClient) ListVolumes(ctx context.Context) ([]truenas.Volume, error) {
	res, err := GetOrLoad(c.cache, opTrueNASDatasets, opTrueNASDatasets, func() ([]truenas.Volume, error) {
		return c.base.ListVolumes(ctx)
	})
	if err != nil {
		return nil, err
	}
	return cloneSlice(res), nil
}

func (c *cachedTrueNASClient) ListSnapshots(ctx context.Context) ([]truenas.Snapshot, error) {
	res, err := GetOrLoad(c.cache, opTrueNASSnapshots, opTrueNASSnapshots, func() ([]truenas.Snapshot, error) {
		return c.base.ListSnapshots(ctx)
	})
	if err != nil {
		return nil, err
	}
	return cloneSlice(res), nil
}

func (c *cachedTrueNASClient) ListPools(ctx context.Context) ([]truenas.Pool, error) {
	return c.base.ListPools(ctx)
}

func (c *cachedTrueNASClient) GetSystemInfo(ctx context.Context) (*truenas.SystemInfo, error) {
	return c.base.GetSystemInfo(ctx)
}

func (c *cachedTrueNASClient) TestConnection(ctx context.Context) error {
	return c.base.TestConnection(ctx)
}
