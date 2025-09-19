package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewChecker(t *testing.T) {
	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           10 * time.Second,
		Timeout:            5 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 3,
		HealthyThreshold:   2,
	}

	checker := NewChecker(cfg)

	if checker == nil {
		t.Error("NewChecker should return non-nil checker")
		return
	}

	if checker.config.Interval != cfg.Interval {
		t.Errorf("Expected interval %v, got %v", cfg.Interval, checker.config.Interval)
	}

	if checker.client.Timeout != cfg.Timeout {
		t.Errorf("Expected timeout %v, got %v", cfg.Timeout, checker.client.Timeout)
	}

	checker.Stop()
}

func TestCheckerIsHealthy(t *testing.T) {
	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           1 * time.Second,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	if !checker.IsHealthy("http://unknown.com") {
		t.Error("Unknown backend should be healthy by default")
	}

	upstreams := []config.Upstream{{
		Name: "test",
		Backends: []config.Backend{{
			URL: "http://test.com",
		}},
	}}

	checker.Start(upstreams)

	if !checker.IsHealthy("http://test.com") {
		t.Error("Backend should be healthy initially")
	}

	checker.updateBackendStatus("http://test.com", false)
	checker.updateBackendStatus("http://test.com", false)

	if checker.IsHealthy("http://test.com") {
		t.Error("Backend should be unhealthy after failures")
	}
}

func TestCheckerGetAllStatuses(t *testing.T) {
	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           1 * time.Second,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	upstreams := []config.Upstream{{
		Name: "test",
		Backends: []config.Backend{
			{URL: "http://backend1.com"},
			{URL: "http://backend2.com"},
		},
	}}

	checker.Start(upstreams)

	statuses := checker.GetAllStatuses()

	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	if !statuses["http://backend1.com"] {
		t.Error("Backend1 should be healthy initially")
	}

	if !statuses["http://backend2.com"] {
		t.Error("Backend2 should be healthy initially")
	}
}

func TestHealthCheckHTTPRequests(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"error"}`))
	}))
	defer unhealthyServer.Close()

	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           100 * time.Millisecond,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	upstreams := []config.Upstream{{
		Name: "test",
		Backends: []config.Backend{
			{URL: healthyServer.URL},
			{URL: unhealthyServer.URL},
		},
	}}

	checker.Start(upstreams)

	time.Sleep(500 * time.Millisecond)

	if !checker.IsHealthy(healthyServer.URL) {
		t.Error("Healthy server should be marked as healthy")
	}

	if checker.IsHealthy(unhealthyServer.URL) {
		t.Error("Unhealthy server should be marked as unhealthy")
	}
}

func TestHealthCheckThresholds(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           50 * time.Millisecond,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	upstreams := []config.Upstream{{
		Name:     "test",
		Backends: []config.Backend{{URL: server.URL}},
	}}

	checker.Start(upstreams)

	if !checker.IsHealthy(server.URL) {
		t.Error("Server should be healthy initially")
	}

	time.Sleep(150 * time.Millisecond)

	if checker.IsHealthy(server.URL) {
		t.Error("Server should be unhealthy after threshold failures")
	}

	time.Sleep(200 * time.Millisecond)

	if !checker.IsHealthy(server.URL) {
		t.Error("Server should be healthy again after threshold successes")
	}
}

func TestCheckerDisabled(t *testing.T) {
	cfg := config.HealthConfig{
		Enabled: false,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	upstreams := []config.Upstream{{
		Name:     "test",
		Backends: []config.Backend{{URL: "http://nonexistent.com"}},
	}}

	checker.Start(upstreams)

	if !checker.IsHealthy("http://nonexistent.com") {
		t.Error("Disabled health checker should assume all backends are healthy")
	}
}

func TestCheckerTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           100 * time.Millisecond,
		Timeout:            50 * time.Millisecond,
		Path:               "/health",
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	upstreams := []config.Upstream{{
		Name:     "test",
		Backends: []config.Backend{{URL: server.URL}},
	}}

	checker.Start(upstreams)

	time.Sleep(300 * time.Millisecond)

	if checker.IsHealthy(server.URL) {
		t.Error("Server should be unhealthy due to timeout")
	}
}

func TestGetStatus(t *testing.T) {
	cfg := config.HealthConfig{
		Enabled:            true,
		Interval:           1 * time.Second,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 2,
		HealthyThreshold:   2,
	}

	checker := NewChecker(cfg)
	defer checker.Stop()

	status := checker.GetStatus("http://unknown.com")
	if !status.Healthy {
		t.Error("Unknown backend should have healthy status")
	}
	if !status.LastCheck.IsZero() {
		t.Error("Unknown backend should have zero LastCheck time")
	}

	upstreams := []config.Upstream{{
		Name:     "test",
		Backends: []config.Backend{{URL: "http://test.com"}},
	}}
	checker.Start(upstreams)

	status = checker.GetStatus("http://test.com")
	if !status.Healthy {
		t.Error("Known backend should be healthy initially")
	}
	if status.LastCheck.IsZero() {
		t.Error("Known backend should have non-zero LastCheck time")
	}
}
