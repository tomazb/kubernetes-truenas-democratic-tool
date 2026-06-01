package inventorycache

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache_GetOrLoad_HitAndMiss(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	var hits, misses int64

	cache := NewCache(Config{
		Enabled: true,
		TTL:     time.Minute,
		MaxSize: 10,
		Now:     func() time.Time { return now },
		Stats: func(_ string, hit bool) {
			if hit {
				atomic.AddInt64(&hits, 1)
			} else {
				atomic.AddInt64(&misses, 1)
			}
		},
	})

	var loads int32
	loader := func() (string, error) {
		atomic.AddInt32(&loads, 1)
		return "value", nil
	}

	v1, err := GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)
	require.Equal(t, "value", v1)
	require.Equal(t, int32(1), loads)
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(1), misses)

	v2, err := GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)
	require.Equal(t, "value", v2)
	require.Equal(t, int32(1), loads)
	require.Equal(t, int64(1), hits)
	require.Equal(t, int64(1), misses)
}

func TestCache_GetOrLoad_Expiry(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	cache := NewCache(Config{
		Enabled: true,
		TTL:     time.Minute,
		MaxSize: 10,
		Now:     func() time.Time { return now },
	})

	var loads int32
	loader := func() (int, error) {
		atomic.AddInt32(&loads, 1)
		return 1, nil
	}

	_, err := GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)

	now = now.Add(2 * time.Minute)

	_, err = GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)
	require.Equal(t, int32(2), loads)
}

func TestCache_GetOrLoad_MaxSizeEviction(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	cache := NewCache(Config{
		Enabled: true,
		TTL:     time.Hour,
		MaxSize: 2,
		Now:     func() time.Time { return now },
	})

	_, err := GetOrLoad(cache, "op", "key1", func() (string, error) { return "a", nil })
	require.NoError(t, err)
	now = now.Add(time.Second)
	_, err = GetOrLoad(cache, "op", "key2", func() (string, error) { return "b", nil })
	require.NoError(t, err)
	now = now.Add(time.Second)
	_, err = GetOrLoad(cache, "op", "key3", func() (string, error) { return "c", nil })
	require.NoError(t, err)

	cache.mu.Lock()
	require.Len(t, cache.entries, 2)
	_, hasKey1 := cache.entries["key1"]
	cache.mu.Unlock()
	require.False(t, hasKey1, "oldest entry should be evicted")
}

func TestCache_DisabledPassthrough(t *testing.T) {
	cache := NewCache(Config{Enabled: false})

	var loads int32
	loader := func() (string, error) {
		atomic.AddInt32(&loads, 1)
		return "direct", nil
	}

	_, err := GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)
	_, err = GetOrLoad(cache, "k8s_pvs", "k8s_pvs", loader)
	require.NoError(t, err)
	require.Equal(t, int32(2), loads)
}

func TestNamespaceKey(t *testing.T) {
	require.Equal(t, "k8s_pvcs:*", NamespaceKey("k8s_pvcs", ""))
	require.Equal(t, "k8s_pvcs:apps", NamespaceKey("k8s_pvcs", "apps"))
}
