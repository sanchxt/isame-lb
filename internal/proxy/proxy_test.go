package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
	"github.com/sanchxt/isame-lb/internal/health"
	"github.com/sanchxt/isame-lb/internal/metrics"
)

func TestNewHandler(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
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
	}

	healthChecker := health.NewChecker(config.HealthConfig{Enabled: false})
	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	handler, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	if handler == nil {
		t.Error("Handler should not be nil")
		return
	}

	if len(handler.loadBalancers) != 1 {
		t.Errorf("Expected 1 load balancer, got %d", len(handler.loadBalancers))
	}

	if _, exists := handler.loadBalancers["test-upstream"]; !exists {
		t.Error("Load balancer for test-upstream should exist")
	}
}

func TestNewHandlerInvalidAlgorithm(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
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

	healthChecker := health.NewChecker(config.HealthConfig{Enabled: false})
	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	_, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err == nil {
		t.Error("Expected error for invalid algorithm")
	}
}

func TestHandlerServeHTTP(t *testing.T) {
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Backend", "backend1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response from backend1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Backend", "backend2")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response from backend2"))
	}))
	defer backend2.Close()

	cfg := &config.Config{
		Service: "test-lb",
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "round_robin",
				Backends: []config.Backend{
					{URL: backend1.URL, Weight: 1},
					{URL: backend2.URL, Weight: 1},
				},
			},
		},
	}

	healthChecker := health.NewChecker(config.HealthConfig{Enabled: false})
	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	handler, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	responses := make([]string, 4)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		responses[i] = string(body)
	}

	backend1Count := 0
	backend2Count := 0
	for _, response := range responses {
		if strings.Contains(response, "backend1") {
			backend1Count++
		} else if strings.Contains(response, "backend2") {
			backend2Count++
		}
	}

	if backend1Count != 2 {
		t.Errorf("Expected 2 requests to backend1, got %d", backend1Count)
	}
	if backend2Count != 2 {
		t.Errorf("Expected 2 requests to backend2, got %d", backend2Count)
	}
}

func TestHandlerNoUpstreams(t *testing.T) {
	cfg := &config.Config{
		Service:   "test-lb",
		Upstreams: []config.Upstream{},
	}

	healthChecker := health.NewChecker(config.HealthConfig{Enabled: false})
	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	handler, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "No upstreams configured") {
		t.Error("Expected error message about no upstreams")
	}
}

func TestHandlerNoHealthyBackends(t *testing.T) {
	cfg := &config.Config{
		Service: "test-lb",
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "round_robin",
				Backends: []config.Backend{
					{URL: "http://backend1.com", Weight: 1},
				},
			},
		},
	}

	healthChecker := health.NewChecker(config.HealthConfig{
		Enabled:            true,
		Interval:           1 * time.Second,
		Timeout:            1 * time.Second,
		Path:               "/health",
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	})
	healthChecker.Start(cfg.Upstreams)

	// manually mark backend as unhealthy
	for i := 0; i < 3; i++ {
		// a bit of a hack for testing...health checker would normally do this
		// healthChecker.updateBackendStatus("http://backend1.com", false)
	}

	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	handler, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "No healthy backends available") {
			t.Error("Expected error message about no healthy backends")
		}
	}
}

func TestHandlerProxyHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Forwarded-For-Echo", r.Header.Get("X-Forwarded-For"))
		w.Header().Set("X-Forwarded-Proto-Echo", r.Header.Get("X-Forwarded-Proto"))
		w.Header().Set("X-Forwarded-Host-Echo", r.Header.Get("X-Forwarded-Host"))
		w.Header().Set("X-Load-Balancer-Echo", r.Header.Get("X-Load-Balancer"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := &config.Config{
		Service: "test-lb",
		Upstreams: []config.Upstream{
			{
				Name:      "test-upstream",
				Algorithm: "round_robin",
				Backends:  []config.Backend{{URL: backend.URL, Weight: 1}},
			},
		},
	}

	healthChecker := health.NewChecker(config.HealthConfig{Enabled: false})
	metricsCollector := metrics.NewCollector(config.MetricsConfig{Enabled: false})

	handler, err := NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Host = "example.com"

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if xForwardedFor := resp.Header.Get("X-Forwarded-For-Echo"); xForwardedFor == "" {
		t.Error("X-Forwarded-For header was not set")
	}

	if xForwardedProto := resp.Header.Get("X-Forwarded-Proto-Echo"); xForwardedProto != "http" {
		t.Errorf("Expected X-Forwarded-Proto 'http', got '%s'", xForwardedProto)
	}

	if xForwardedHost := resp.Header.Get("X-Forwarded-Host-Echo"); xForwardedHost != "example.com" {
		t.Errorf("Expected X-Forwarded-Host 'example.com', got '%s'", xForwardedHost)
	}

	if xLoadBalancer := resp.Header.Get("X-Load-Balancer-Echo"); xLoadBalancer != "test-lb" {
		t.Errorf("Expected X-Load-Balancer 'test-lb', got '%s'", xLoadBalancer)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "192.168.1.200"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.200",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "10.0.0.1:12345",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
				"X-Real-IP":       "192.168.1.200",
			},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			clientIP := getClientIP(req)
			if clientIP != tt.expectedIP {
				t.Errorf("Expected client IP %s, got %s", tt.expectedIP, clientIP)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected default status code 200, got %d", rw.statusCode)
	}

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", rw.statusCode)
	}
}
