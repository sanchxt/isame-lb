package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.NewDefaultConfig()
	srv, err := New(cfg)

	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if srv == nil {
		t.Fatal("New() returned nil")
	}

	if srv.config != cfg {
		t.Error("New() did not set config correctly")
	}
}

func TestNewWithUpstreams(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
		Version: "1.0.0",
		Server: config.ServerConfig{
			Port: 8080,
		},
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "round_robin",
				Backends: []config.Backend{
					{URL: "http://backend1.com", Weight: 1},
					{URL: "http://backend2.com", Weight: 1},
				},
			},
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if srv == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWithInvalidUpstream(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
		Version: "1.0.0",
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "invalid-algorithm",
				Backends: []config.Backend{
					{URL: "http://backend1.com", Weight: 1},
				},
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("Expected error for invalid algorithm")
	}
}

func TestLoadBalancerServer_healthHandler(t *testing.T) {
	cfg := config.NewDefaultConfig()
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	srv.healthHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("healthHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("healthHandler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("healthHandler response should contain status ok: got %v", body)
	}

	if !strings.Contains(body, `"service":"isame-lb"`) {
		t.Errorf("healthHandler response should contain service name: got %v", body)
	}
}

func TestLoadBalancerServer_statusHandler(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
		Version: "1.0.0",
		Server: config.ServerConfig{
			Port: 8080,
		},
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "round_robin",
				Backends: []config.Backend{
					{URL: "http://backend1.com", Weight: 1},
					{URL: "http://backend2.com", Weight: 1},
				},
			},
		},
		Health: config.HealthConfig{
			Enabled: false,
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	srv.statusHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("statusHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("statusHandler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}

	body := rr.Body.String()

	expectedFields := []string{
		`"service": "test-lb"`,
		`"version": "1.0.0"`,
		`"upstreams": 1`,
		`"total": 2`,
		`"healthy": 0`,
		`"unhealthy": 2`,
		`"health_checks_enabled": false`,
		`"metrics_enabled": false`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("statusHandler response should contain %s: got %v", field, body)
		}
	}
}
