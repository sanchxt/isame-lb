package ratelimit

import (
	"sync"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

type requestRecord struct {
	timestamp time.Time
}

type clientLimiter struct {
	requests []requestRecord
	mu       sync.Mutex
}

type RateLimiter struct {
	config  *config.RateLimitConfig
	clients map[string]*clientLimiter
	mu      sync.RWMutex
}

func New(cfg *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:  cfg,
		clients: make(map[string]*clientLimiter),
	}
}

func (rl *RateLimiter) Allow(clientIP string) bool {
	if rl.config == nil || !rl.config.Enabled {
		return true
	}

	rl.mu.Lock()
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &clientLimiter{
			requests: make([]requestRecord, 0),
		}
		rl.clients[clientIP] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.config.WindowSize)

	validRequests := make([]requestRecord, 0)
	for _, req := range client.requests {
		if req.timestamp.After(windowStart) {
			validRequests = append(validRequests, req)
		}
	}
	client.requests = validRequests

	if len(client.requests) >= rl.config.RequestsPerIP {
		return false
	}

	client.requests = append(client.requests, requestRecord{
		timestamp: now,
	})

	return true
}

func (rl *RateLimiter) GetUsage(clientIP string) int {
	if rl.config == nil || !rl.config.Enabled {
		return 0
	}

	rl.mu.RLock()
	client, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if !exists {
		return 0
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.config.WindowSize)

	count := 0
	for _, req := range client.requests {
		if req.timestamp.After(windowStart) {
			count++
		}
	}

	return count
}

func (rl *RateLimiter) Cleanup() {
	if rl.config == nil || !rl.config.Enabled {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.config.WindowSize)

	for clientIP, client := range rl.clients {
		client.mu.Lock()
		hasValidRequests := false
		for _, req := range client.requests {
			if req.timestamp.After(windowStart) {
				hasValidRequests = true
				break
			}
		}
		client.mu.Unlock()

		if !hasValidRequests {
			delete(rl.clients, clientIP)
		}
	}
}
