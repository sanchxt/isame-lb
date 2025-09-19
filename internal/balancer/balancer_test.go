package balancer

import (
	"net/http"
	"testing"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewLoadBalancer(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		expectErr bool
		expectAlg string
	}{
		{
			name:      "round_robin",
			algorithm: "round_robin",
			expectErr: false,
			expectAlg: "round_robin",
		},
		{
			name:      "empty string defaults to round_robin",
			algorithm: "",
			expectErr: false,
			expectAlg: "round_robin",
		},
		{
			name:      "invalid algorithm",
			algorithm: "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb, err := NewLoadBalancer(tt.algorithm)

			if (err != nil) != tt.expectErr {
				t.Errorf("NewLoadBalancer() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if lb == nil {
					t.Error("Expected load balancer to be non-nil")
					return
				}
				if lb.Algorithm() != tt.expectAlg {
					t.Errorf("Expected algorithm %s, got %s", tt.expectAlg, lb.Algorithm())
				}
			}
		})
	}
}

func TestRoundRobinSelectBackend(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
		{URL: "http://backend3.com", Weight: 1},
	}

	rr := NewRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)
	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": true,
		"http://backend3.com": true,
	}

	selections := make(map[string]int)
	for i := 0; i < 9; i++ {
		backend, err := rr.SelectBackend(req, backends, healthStatus)
		if err != nil {
			t.Errorf("SelectBackend() unexpected error: %v", err)
		}
		if backend == nil {
			t.Error("SelectBackend() returned nil backend")
			continue
		}
		selections[backend.URL]++
	}

	for _, backend := range backends {
		if count := selections[backend.URL]; count != 3 {
			t.Errorf("Backend %s selected %d times, expected 3", backend.URL, count)
		}
	}
}

func TestRoundRobinWithUnhealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
		{URL: "http://backend3.com", Weight: 1},
	}

	rr := NewRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": false,
		"http://backend3.com": true,
	}

	selections := make(map[string]int)
	for i := 0; i < 6; i++ {
		backend, err := rr.SelectBackend(req, backends, healthStatus)
		if err != nil {
			t.Errorf("SelectBackend() unexpected error: %v", err)
		}
		if backend == nil {
			t.Error("SelectBackend() returned nil backend")
			continue
		}

		if backend.URL == "http://backend2.com" {
			t.Error("Selected unhealthy backend")
		}
		selections[backend.URL]++
	}

	if selections["http://backend2.com"] != 0 {
		t.Error("Unhealthy backend was selected")
	}
	if selections["http://backend1.com"] != 3 {
		t.Errorf("Backend1 selected %d times, expected 3", selections["http://backend1.com"])
	}
	if selections["http://backend3.com"] != 3 {
		t.Errorf("Backend3 selected %d times, expected 3", selections["http://backend3.com"])
	}
}

func TestRoundRobinNoHealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
	}

	rr := NewRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": false,
		"http://backend2.com": false,
	}

	backend, err := rr.SelectBackend(req, backends, healthStatus)
	if err == nil {
		t.Error("Expected error when no healthy backends available")
	}
	if backend != nil {
		t.Error("Expected nil backend when no healthy backends available")
	}
	if err != ErrNoHealthyBackends {
		t.Errorf("Expected ErrNoHealthyBackends, got %v", err)
	}
}

func TestRoundRobinEmptyBackends(t *testing.T) {
	rr := NewRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)
	healthStatus := make(map[string]bool)

	backend, err := rr.SelectBackend(req, []config.Backend{}, healthStatus)
	if err == nil {
		t.Error("Expected error when no backends available")
	}
	if backend != nil {
		t.Error("Expected nil backend when no backends available")
	}
	if err != ErrNoHealthyBackends {
		t.Errorf("Expected ErrNoHealthyBackends, got %v", err)
	}
}

func TestRoundRobinWithNoHealthStatus(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
	}

	rr := NewRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := make(map[string]bool)

	selections := make(map[string]int)
	for i := 0; i < 4; i++ {
		backend, err := rr.SelectBackend(req, backends, healthStatus)
		if err != nil {
			t.Errorf("SelectBackend() unexpected error: %v", err)
		}
		if backend == nil {
			t.Error("SelectBackend() returned nil backend")
			continue
		}
		selections[backend.URL]++
	}

	if selections["http://backend1.com"] != 2 {
		t.Errorf("Backend1 selected %d times, expected 2", selections["http://backend1.com"])
	}
	if selections["http://backend2.com"] != 2 {
		t.Errorf("Backend2 selected %d times, expected 2", selections["http://backend2.com"])
	}
}
