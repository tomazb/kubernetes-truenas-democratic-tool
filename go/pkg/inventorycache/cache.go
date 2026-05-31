package inventorycache

import (
	"fmt"
	"sync"
	"time"
)

// StatsRecorder records cache hit/miss events for metrics.
type StatsRecorder func(operation string, hit bool)

// Config holds TTL cache settings.
type Config struct {
	Enabled bool
	TTL     time.Duration
	MaxSize int
	Now     func() time.Time
	Stats   StatsRecorder
}

// Cache is a thread-safe in-memory TTL cache with optional max size.
type Cache struct {
	enabled bool
	ttl     time.Duration
	maxSize int
	now     func() time.Time
	stats   StatsRecorder

	mu      sync.Mutex
	entries map[string]entry
}

type entry struct {
	value      any
	expiresAt  time.Time
	insertedAt time.Time
}

// NewCache creates a TTL cache. When disabled, GetOrLoad always calls the loader.
func NewCache(cfg Config) *Cache {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = 1000
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	return &Cache{
		enabled: cfg.Enabled,
		ttl:     ttl,
		maxSize: maxSize,
		now:     now,
		stats:   cfg.Stats,
		entries: make(map[string]entry),
	}
}

// Enabled reports whether caching is active.
func (c *Cache) Enabled() bool {
	return c != nil && c.enabled
}

// GetOrLoad returns a cached value or loads it via loader when missing or expired.
func GetOrLoad[T any](c *Cache, operation, key string, loader func() (T, error)) (T, error) {
	var zero T
	if c == nil || !c.enabled {
		return loader()
	}

	if value, ok := c.get(key); ok {
		typed, ok := value.(T)
		if !ok {
			return zero, fmt.Errorf("inventory cache type mismatch for key %q", key)
		}
		if c.stats != nil {
			c.stats(operation, true)
		}
		return typed, nil
	}

	loaded, err := loader()
	if err != nil {
		return zero, err
	}

	c.set(key, loaded)
	if c.stats != nil {
		c.stats(operation, false)
	}
	return loaded, nil
}

func (c *Cache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if !c.now().Before(item.expiresAt) {
		delete(c.entries, key)
		return nil, false
	}

	return item.value, true
}

func (c *Cache) set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	c.entries[key] = entry{
		value:      value,
		expiresAt:  now.Add(c.ttl),
		insertedAt: now,
	}

	if len(c.entries) <= c.maxSize {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	for k, item := range c.entries {
		if oldestKey == "" || item.insertedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = item.insertedAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// NamespaceKey builds a cache key with optional namespace scope.
func NamespaceKey(prefix, namespace string) string {
	if namespace == "" {
		return prefix + ":*"
	}
	return prefix + ":" + namespace
}
