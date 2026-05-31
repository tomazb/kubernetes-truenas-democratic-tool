package inventorycache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClonePersistentVolumes_ReturnsIndependentCopy(t *testing.T) {
	original := []corev1.PersistentVolume{{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "pv-1",
			Labels: map[string]string{"app": "demo"},
		},
	}}
	copied := clonePersistentVolumes(original)
	require.Len(t, copied, 1)
	copied[0].Name = "mutated"
	copied[0].Labels["app"] = "changed"
	require.Equal(t, "pv-1", original[0].Name)
	require.Equal(t, "demo", original[0].Labels["app"])
}

func TestCache_GetOrLoad_ExpiresAtBoundary(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	cache := NewCache(Config{
		Enabled: true,
		TTL:     time.Minute,
		MaxSize: 10,
		Now:     func() time.Time { return now },
	})

	var loads int
	loader := func() (int, error) {
		loads++
		return loads, nil
	}

	_, err := GetOrLoad(cache, "op", "key", loader)
	require.NoError(t, err)
	require.Equal(t, 1, loads)

	now = now.Add(time.Minute)
	_, err = GetOrLoad(cache, "op", "key", loader)
	require.NoError(t, err)
	require.Equal(t, 2, loads)
}
