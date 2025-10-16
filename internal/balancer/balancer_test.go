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
			name:      "weighted_round_robin",
			algorithm: "weighted_round_robin",
			expectErr: false,
			expectAlg: "weighted_round_robin",
		},
		{
			name:      "least_connections",
			algorithm: "least_connections",
			expectErr: false,
			expectAlg: "least_connections",
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

func TestWeightedRoundRobinSelectBackend(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 3},
		{URL: "http://backend2.com", Weight: 2},
		{URL: "http://backend3.com", Weight: 1},
	}

	wrr := NewWeightedRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)
	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": true,
		"http://backend3.com": true,
	}

	selections := make(map[string]int)
	for i := 0; i < 60; i++ {
		backend, err := wrr.SelectBackend(req, backends, healthStatus)
		if err != nil {
			t.Errorf("SelectBackend() unexpected error: %v", err)
		}
		if backend == nil {
			t.Error("SelectBackend() returned nil backend")
			continue
		}
		selections[backend.URL]++
	}

	// backend1:backend2:backend3 should be 3:2:1
	expected := map[string]int{
		"http://backend1.com": 30, // 50% of 60
		"http://backend2.com": 20, // 33% of 60
		"http://backend3.com": 10, // 17% of 60
	}

	for url, expectedCount := range expected {
		actualCount := selections[url]
		if actualCount != expectedCount {
			t.Errorf("Backend %s selected %d times, expected %d", url, actualCount, expectedCount)
		}
	}
}

func TestWeightedRoundRobinWithUnhealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 3},
		{URL: "http://backend2.com", Weight: 2},
		{URL: "http://backend3.com", Weight: 1},
	}

	wrr := NewWeightedRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": false, // unhealthy
		"http://backend3.com": true,
	}

	// 40 iterations (3+1)*10 = 40 (excluding unhealthy backend2)
	selections := make(map[string]int)
	for i := 0; i < 40; i++ {
		backend, err := wrr.SelectBackend(req, backends, healthStatus)
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

	// verify backend2 was never selected
	if selections["http://backend2.com"] != 0 {
		t.Error("Unhealthy backend was selected")
	}

	// verify distribution of remaining backends (3:1 ratio)
	if selections["http://backend1.com"] != 30 {
		t.Errorf("Backend1 selected %d times, expected 30", selections["http://backend1.com"])
	}
	if selections["http://backend3.com"] != 10 {
		t.Errorf("Backend3 selected %d times, expected 10", selections["http://backend3.com"])
	}
}

func TestWeightedRoundRobinNoHealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 2},
		{URL: "http://backend2.com", Weight: 1},
	}

	wrr := NewWeightedRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": false,
		"http://backend2.com": false,
	}

	backend, err := wrr.SelectBackend(req, backends, healthStatus)
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

func TestWeightedRoundRobinSmoothDistribution(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://a.com", Weight: 5},
		{URL: "http://b.com", Weight: 1},
		{URL: "http://c.com", Weight: 1},
	}

	wrr := NewWeightedRoundRobin()
	req, _ := http.NewRequest("GET", "/test", nil)
	healthStatus := map[string]bool{
		"http://a.com": true,
		"http://b.com": true,
		"http://c.com": true,
	}

	var selectionOrder []string
	for i := 0; i < 7; i++ {
		backend, err := wrr.SelectBackend(req, backends, healthStatus)
		if err != nil {
			t.Fatalf("SelectBackend() unexpected error: %v", err)
		}
		selectionOrder = append(selectionOrder, backend.URL)
	}

	maxConsecutiveA := 0
	currentConsecutiveA := 0
	for _, url := range selectionOrder {
		if url == "http://a.com" {
			currentConsecutiveA++
			if currentConsecutiveA > maxConsecutiveA {
				maxConsecutiveA = currentConsecutiveA
			}
		} else {
			currentConsecutiveA = 0
		}
	}

	if maxConsecutiveA > 2 {
		t.Errorf("Backend 'a' appeared %d times consecutively, expected <= 2 for smooth distribution. Order: %v", maxConsecutiveA, selectionOrder)
	}
}

func TestLeastConnectionsSelectBackend(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
		{URL: "http://backend3.com", Weight: 1},
	}

	lc := NewLeastConnections()
	req, _ := http.NewRequest("GET", "/test", nil)
	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": true,
		"http://backend3.com": true,
	}

	backend1, err := lc.SelectBackend(req, backends, healthStatus)
	if err != nil {
		t.Fatalf("SelectBackend() unexpected error: %v", err)
	}

	lc.IncrementConnections(backend1.URL)

	backend2, err := lc.SelectBackend(req, backends, healthStatus)
	if err != nil {
		t.Fatalf("SelectBackend() unexpected error: %v", err)
	}
	if backend2.URL == backend1.URL {
		t.Error("Expected different backend with fewer connections")
	}

	lc.IncrementConnections(backend2.URL)

	backend3, err := lc.SelectBackend(req, backends, healthStatus)
	if err != nil {
		t.Fatalf("SelectBackend() unexpected error: %v", err)
	}
	if backend3.URL == backend1.URL || backend3.URL == backend2.URL {
		t.Error("Expected backend3 with fewest connections")
	}

	lc.IncrementConnections(backend3.URL)

	lc.DecrementConnections(backend1.URL)

	backend4, err := lc.SelectBackend(req, backends, healthStatus)
	if err != nil {
		t.Fatalf("SelectBackend() unexpected error: %v", err)
	}
	if backend4.URL != backend1.URL {
		t.Errorf("Expected backend1 with fewest connections, got %s", backend4.URL)
	}
}

func TestLeastConnectionsWithUnhealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
		{URL: "http://backend3.com", Weight: 1},
	}

	lc := NewLeastConnections()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": true,
		"http://backend2.com": false, // unhealthy
		"http://backend3.com": true,
	}

	for i := 0; i < 5; i++ {
		lc.IncrementConnections("http://backend1.com")
	}

	backend, err := lc.SelectBackend(req, backends, healthStatus)
	if err != nil {
		t.Fatalf("SelectBackend() unexpected error: %v", err)
	}
	if backend.URL == "http://backend2.com" {
		t.Error("Selected unhealthy backend")
	}
	if backend.URL == "http://backend1.com" {
		t.Error("Selected backend with more connections instead of backend3")
	}
}

func TestLeastConnectionsNoHealthyBackends(t *testing.T) {
	backends := []config.Backend{
		{URL: "http://backend1.com", Weight: 1},
		{URL: "http://backend2.com", Weight: 1},
	}

	lc := NewLeastConnections()
	req, _ := http.NewRequest("GET", "/test", nil)

	healthStatus := map[string]bool{
		"http://backend1.com": false,
		"http://backend2.com": false,
	}

	backend, err := lc.SelectBackend(req, backends, healthStatus)
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

func TestLeastConnectionsConnectionTracking(t *testing.T) {
	lc := NewLeastConnections()

	lc.IncrementConnections("http://test.com")
	lc.IncrementConnections("http://test.com")
	lc.IncrementConnections("http://test.com")

	if count := lc.GetConnections("http://test.com"); count != 3 {
		t.Errorf("Expected 3 connections, got %d", count)
	}

	lc.DecrementConnections("http://test.com")
	if count := lc.GetConnections("http://test.com"); count != 2 {
		t.Errorf("Expected 2 connections after decrement, got %d", count)
	}

	lc.DecrementConnections("http://test.com")
	lc.DecrementConnections("http://test.com")
	lc.DecrementConnections("http://test.com")
	if count := lc.GetConnections("http://test.com"); count != 0 {
		t.Errorf("Expected 0 connections (should not go negative), got %d", count)
	}
}
