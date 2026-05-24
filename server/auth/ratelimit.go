package auth

import (
	"sync"
	"time"
)

// RateLimiter is a per-key token-bucket rate limiter safe for concurrent use.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rlEntry
	rate    float64 // tokens per second
	burst   float64
}

type rlEntry struct {
	tokens   float64
	lastSeen time.Time
}

// NewRateLimiter creates a limiter that allows burst requests immediately and
// then refills at rate requests per second.
func NewRateLimiter(ratePerSec float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rlEntry),
		rate:    ratePerSec,
		burst:   float64(burst),
	}
	go rl.cleanup()
	return rl
}

// Allow returns true if the request for key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[key]
	if !ok {
		rl.entries[key] = &rlEntry{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(e.lastSeen).Seconds()
	e.tokens = min64(rl.burst, e.tokens+elapsed*rl.rate)
	e.lastSeen = now

	if e.tokens < 1 {
		return false
	}
	e.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, e := range rl.entries {
			if e.lastSeen.Before(cutoff) {
				delete(rl.entries, k)
			}
		}
		rl.mu.Unlock()
	}
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
