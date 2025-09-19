package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewCollector(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9090,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)

	if collector == nil {
		t.Error("NewCollector should return non-nil collector")
		return
	}

	if collector.config.Port != cfg.Port {
		t.Errorf("Expected port %d, got %d", cfg.Port, collector.config.Port)
	}

	if collector.registry == nil {
		t.Error("Registry should be initialized")
	}

	if collector.requestsTotal == nil {
		t.Error("requestsTotal metric should be initialized")
	}

	if collector.requestDuration == nil {
		t.Error("requestDuration metric should be initialized")
	}

	if collector.upstreamHealthy == nil {
		t.Error("upstreamHealthy metric should be initialized")
	}

	if collector.connectionsActive == nil {
		t.Error("connectionsActive metric should be initialized")
	}
}

func TestCollectorStartStop(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    0,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)

	err := collector.Start()
	if err != nil {
		t.Errorf("Start() unexpected error: %v", err)
	}

	err = collector.Stop()
	if err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}
}

func TestCollectorDisabled(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: false,
		Port:    9090,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)

	err := collector.Start()
	if err != nil {
		t.Errorf("Start() with disabled config unexpected error: %v", err)
	}

	if collector.server != nil {
		t.Error("Server should not be initialized when metrics are disabled")
	}

	collector.RecordRequest("test", "backend", "GET", "200", time.Second)
	collector.UpdateBackendHealth("test", "backend", true)
	collector.SetActiveConnections(5)
	collector.IncrementActiveConnections()
	collector.DecrementActiveConnections()
}

func TestRecordRequest(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9091,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)
	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	collector.RecordRequest("web", "backend1", "GET", "200", 100*time.Millisecond)
	collector.RecordRequest("web", "backend1", "GET", "200", 200*time.Millisecond)
	collector.RecordRequest("web", "backend2", "POST", "404", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.Path))
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	content := string(body)

	if !strings.Contains(content, "isame_lb_requests_total") {
		t.Error("requests_total metric not found in output")
	}

	if !strings.Contains(content, "isame_lb_request_duration_seconds") {
		t.Error("request_duration metric not found in output")
	}

	if !strings.Contains(content, `isame_lb_requests_total{backend="backend1",method="GET",status="200",upstream="web"} 2`) {
		t.Error("Expected 2 requests for backend1 GET 200")
	}

	if !strings.Contains(content, `isame_lb_requests_total{backend="backend2",method="POST",status="404",upstream="web"} 1`) {
		t.Error("Expected 1 request for backend2 POST 404")
	}
}

func TestUpdateBackendHealth(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9092,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)
	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	collector.UpdateBackendHealth("web", "backend1", true)
	collector.UpdateBackendHealth("web", "backend2", false)
	collector.UpdateBackendHealth("api", "backend3", true)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.Path))
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	content := string(body)

	if !strings.Contains(content, `isame_lb_upstream_healthy{backend="backend1",upstream="web"} 1`) {
		t.Error("Expected backend1 to be healthy (value 1)")
	}

	if !strings.Contains(content, `isame_lb_upstream_healthy{backend="backend2",upstream="web"} 0`) {
		t.Error("Expected backend2 to be unhealthy (value 0)")
	}

	if !strings.Contains(content, `isame_lb_upstream_healthy{backend="backend3",upstream="api"} 1`) {
		t.Error("Expected backend3 to be healthy (value 1)")
	}
}

func TestActiveConnections(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9093,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)
	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	collector.SetActiveConnections(10)
	collector.IncrementActiveConnections()
	collector.IncrementActiveConnections()
	collector.DecrementActiveConnections()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.Path))
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	content := string(body)

	if !strings.Contains(content, "isame_lb_active_connections 11") {
		t.Error("Expected active connections to be 11")
	}
}

func TestMetricsHealthEndpoint(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    9094,
		Path:    "/metrics",
	}

	collector := NewCollector(cfg)
	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", cfg.Port))
	if err != nil {
		t.Fatalf("Failed to fetch health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := `{"status":"ok"}`
	if string(body) != expected {
		t.Errorf("Expected %s, got %s", expected, string(body))
	}
}
