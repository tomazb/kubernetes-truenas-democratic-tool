package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	defaultRequestsPerMinute = 100
	defaultRateBurst         = 100
	defaultClientIdleTTL     = 10 * time.Minute
	defaultMaxClientEntries  = 4096
)

type perClientRateLimiter struct {
	mu         sync.Mutex
	clients    map[string]*clientLimiterEntry
	limit      rate.Limit
	burst      int
	idleTTL    time.Duration
	maxEntries int
}

type clientLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newPerClientRateLimiter(requestsPerMinute, burst int, idleTTL time.Duration, maxEntries int) *perClientRateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = defaultRequestsPerMinute
	}
	if burst <= 0 {
		burst = requestsPerMinute
	}
	if idleTTL <= 0 {
		idleTTL = defaultClientIdleTTL
	}
	if maxEntries <= 0 {
		maxEntries = defaultMaxClientEntries
	}

	return &perClientRateLimiter{
		clients:    make(map[string]*clientLimiterEntry),
		limit:      rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:      burst,
		idleTTL:    idleTTL,
		maxEntries: maxEntries,
	}
}

func (p *perClientRateLimiter) allow(clientKey string) bool {
	now := time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.clients[clientKey]
	if !ok {
		p.evictLocked(now)
		entry = &clientLimiterEntry{
			limiter:  rate.NewLimiter(p.limit, p.burst),
			lastSeen: now,
		}
		p.clients[clientKey] = entry
		if len(p.clients) > p.maxEntries {
			p.evictLocked(now)
		}
	}

	entry.lastSeen = now
	return entry.limiter.Allow()
}

func (p *perClientRateLimiter) retryAfter() time.Duration {
	if p.limit <= 0 {
		return time.Second
	}
	return time.Duration(float64(time.Second) / float64(p.limit))
}

func (p *perClientRateLimiter) evictLocked(now time.Time) {
	for key, entry := range p.clients {
		if now.Sub(entry.lastSeen) > p.idleTTL {
			delete(p.clients, key)
		}
	}

	if len(p.clients) <= p.maxEntries {
		return
	}

	for len(p.clients) > p.maxEntries {
		var oldestKey string
		var oldestTime time.Time
		first := true
		for key, entry := range p.clients {
			if first || entry.lastSeen.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.lastSeen
				first = false
			}
		}
		if oldestKey == "" {
			return
		}
		delete(p.clients, oldestKey)
	}
}

func (p *perClientRateLimiter) clientCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.clients)
}

func clientRateLimitKey(c *gin.Context) string {
	// ClientIP() honors engine trusted-proxy settings configured in NewServer.
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	return "unknown"
}

func perClientRateLimitMiddleware(limiter *perClientRateLimiter) gin.HandlerFunc {
	if limiter == nil {
		limiter = newPerClientRateLimiter(defaultRequestsPerMinute, defaultRateBurst, defaultClientIdleTTL, defaultMaxClientEntries)
	}

	return func(c *gin.Context) {
		clientKey := clientRateLimitKey(c)

		if !limiter.allow(clientKey) {
			retryAfter := limiter.retryAfter()
			retryAfterSec := int(retryAfter.Seconds())
			if retryAfterSec < 1 {
				retryAfterSec = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSec))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.String(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
