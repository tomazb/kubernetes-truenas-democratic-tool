package api

import (
	"testing"
	"time"
)

func TestPerClientRateLimiter_IsolatesClients(t *testing.T) {
	limiter := newPerClientRateLimiter(60, 2, time.Minute, 128)

	if !limiter.allow("client-a") {
		t.Fatal("client-a first request should be allowed")
	}
	if !limiter.allow("client-a") {
		t.Fatal("client-a second request should be allowed within burst")
	}
	if limiter.allow("client-a") {
		t.Fatal("client-a should be rate limited after burst")
	}

	if !limiter.allow("client-b") {
		t.Fatal("client-b first request should be allowed")
	}
	if !limiter.allow("client-b") {
		t.Fatal("client-b second request should be allowed within burst")
	}
}

func TestPerClientRateLimiter_RetryAfterUsesConfiguredRate(t *testing.T) {
	limiter := newPerClientRateLimiter(120, 2, time.Minute, 128)
	got := limiter.retryAfter()
	want := 500 * time.Millisecond
	if got != want {
		t.Fatalf("retryAfter() = %v, want %v", got, want)
	}
}

func TestPerClientRateLimiter_EnforcesMaxEntries(t *testing.T) {
	limiter := newPerClientRateLimiter(60, 1, time.Hour, 2)

	limiter.allow("a")
	limiter.allow("b")
	limiter.allow("c")

	if limiter.clientCount() > 2 {
		t.Fatalf("client count = %d, want <= 2", limiter.clientCount())
	}
}

func TestPerClientRateLimiter_EvictsIdleClients(t *testing.T) {
	limiter := newPerClientRateLimiter(60, 1, 10*time.Millisecond, 128)

	if !limiter.allow("ephemeral") {
		t.Fatal("expected first request allowed")
	}
	if limiter.clientCount() != 1 {
		t.Fatalf("client count = %d, want 1", limiter.clientCount())
	}

	time.Sleep(15 * time.Millisecond)
	if !limiter.allow("other") {
		t.Fatal("expected other client allowed")
	}
	if limiter.clientCount() != 1 {
		t.Fatalf("expected idle client evicted, count = %d", limiter.clientCount())
	}
}
