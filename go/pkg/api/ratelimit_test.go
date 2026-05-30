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
