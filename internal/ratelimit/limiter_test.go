package ratelimit

import (
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewRateLimiter(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 10,
		WindowSize:    1 * time.Minute,
	}

	rl := New(cfg)
	if rl == nil {
		t.Fatal("Expected rate limiter to be non-nil")
	}
}

func TestRateLimiterAllow(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 3,
		WindowSize:    1 * time.Second,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	if rl.Allow(clientIP) {
		t.Error("Request should be denied after exceeding limit")
	}
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 3,
		WindowSize:    500 * time.Millisecond,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	if rl.Allow(clientIP) {
		t.Error("Request should be denied")
	}

	time.Sleep(600 * time.Millisecond)

	if !rl.Allow(clientIP) {
		t.Error("Request should be allowed after window expiry")
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 2,
		WindowSize:    1 * time.Second,
	}

	rl := New(cfg)
	client1 := "192.168.1.1"
	client2 := "192.168.1.2"

	if !rl.Allow(client1) {
		t.Error("Client 1 request 1 should be allowed")
	}
	if !rl.Allow(client1) {
		t.Error("Client 1 request 2 should be allowed")
	}
	if rl.Allow(client1) {
		t.Error("Client 1 request 3 should be denied")
	}

	if !rl.Allow(client2) {
		t.Error("Client 2 request 1 should be allowed")
	}
	if !rl.Allow(client2) {
		t.Error("Client 2 request 2 should be allowed")
	}
	if rl.Allow(client2) {
		t.Error("Client 2 request 3 should be denied")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       false,
		RequestsPerIP: 2,
		WindowSize:    1 * time.Second,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	for i := 0; i < 100; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed when rate limiter is disabled", i+1)
		}
	}
}

func TestRateLimiterNilConfig(t *testing.T) {
	rl := New(nil)
	clientIP := "192.168.1.1"

	for i := 0; i < 10; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed with nil config", i+1)
		}
	}
}

func TestRateLimiterGetUsage(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 5,
		WindowSize:    1 * time.Second,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	for i := 0; i < 3; i++ {
		rl.Allow(clientIP)
	}

	usage := rl.GetUsage(clientIP)
	if usage != 3 {
		t.Errorf("Expected usage of 3, got %d", usage)
	}

	if usage := rl.GetUsage("192.168.1.2"); usage != 0 {
		t.Errorf("Expected usage of 0 for new client, got %d", usage)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 5,
		WindowSize:    100 * time.Millisecond,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	rl.Allow(clientIP)
	rl.Allow(clientIP)

	usage := rl.GetUsage(clientIP)
	if usage != 2 {
		t.Errorf("Expected usage of 2, got %d", usage)
	}

	time.Sleep(150 * time.Millisecond)

	usage = rl.GetUsage(clientIP)
	if usage != 0 {
		t.Errorf("Expected usage of 0 after window expiry, got %d", usage)
	}
}

func TestRateLimiterPartialWindow(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:       true,
		RequestsPerIP: 5,
		WindowSize:    1 * time.Second,
	}

	rl := New(cfg)
	clientIP := "192.168.1.1"

	rl.Allow(clientIP)
	rl.Allow(clientIP)
	rl.Allow(clientIP)

	time.Sleep(500 * time.Millisecond)

	rl.Allow(clientIP)
	rl.Allow(clientIP)

	if rl.Allow(clientIP) {
		t.Error("Should be at limit")
	}

	time.Sleep(600 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if !rl.Allow(clientIP) {
			t.Errorf("Request %d should be allowed after partial expiry", i+1)
		}
	}
}
