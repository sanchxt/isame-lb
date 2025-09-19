package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	if cfg.Version == "" {
		t.Error("Default config should have a version")
	}

	if cfg.Service == "" {
		t.Error("Default config should have a service name")
	}

	expectedService := "isame-lb"
	if cfg.Service != expectedService {
		t.Errorf("Expected service %s, got %s", expectedService, cfg.Service)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Health.Interval != 30*time.Second {
		t.Errorf("Expected default health interval 30s, got %v", cfg.Health.Interval)
	}

	if cfg.Metrics.Port != 9090 {
		t.Errorf("Expected default metrics port 9090, got %d", cfg.Metrics.Port)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		hasErr bool
	}{
		{
			name: "valid config with upstreams",
			config: &Config{
				Version: "1.0.0",
				Service: "test-service",
				Server:  ServerConfig{Port: 8080},
				Upstreams: []Upstream{{
					Name:      "test",
					Algorithm: "round_robin",
					Backends:  []Backend{{URL: "http://localhost:3000", Weight: 1}},
				}},
				Health:  HealthConfig{Enabled: true},
				Metrics: MetricsConfig{Enabled: true},
			},
			hasErr: false,
		},
		{
			name:   "empty config should fail validation (no upstreams)",
			config: &Config{},
			hasErr: true,
		},
		{
			name: "invalid server port",
			config: &Config{
				Server: ServerConfig{Port: -1},
				Upstreams: []Upstream{{
					Name:     "test",
					Backends: []Backend{{URL: "http://localhost:3000", Weight: 1}},
				}},
			},
			hasErr: true,
		},
		{
			name: "invalid backend URL",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Upstreams: []Upstream{{
					Name:     "test",
					Backends: []Backend{{URL: "not-a-valid-url", Weight: 1}},
				}},
			},
			hasErr: true,
		},
		{
			name: "missing upstream name",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Upstreams: []Upstream{{
					Backends: []Backend{{URL: "http://localhost:3000", Weight: 1}},
				}},
			},
			hasErr: true,
		},
		{
			name: "no backends in upstream",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Upstreams: []Upstream{{
					Name:     "test",
					Backends: []Backend{},
				}},
			},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.hasErr {
				t.Errorf("Validate() error = %v, hasErr %v", err, tt.hasErr)
			}

			if !tt.hasErr {
				if tt.config.Service == "" {
					t.Error("Service should not be empty after validation")
				}
				if tt.config.Version == "" {
					t.Error("Version should not be empty after validation")
				}
			}
		})
	}
}

func TestBackendValidation(t *testing.T) {
	tests := []struct {
		name    string
		backend Backend
		hasErr  bool
	}{
		{
			name:    "valid HTTP backend",
			backend: Backend{URL: "http://localhost:3000", Weight: 1},
			hasErr:  false,
		},
		{
			name:    "valid HTTPS backend",
			backend: Backend{URL: "https://api.example.com", Weight: 5},
			hasErr:  false,
		},
		{
			name:    "invalid scheme",
			backend: Backend{URL: "ftp://localhost:3000", Weight: 1},
			hasErr:  true,
		},
		{
			name:    "empty URL",
			backend: Backend{URL: "", Weight: 1},
			hasErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Port: 8080},
				Upstreams: []Upstream{{
					Name:     "test",
					Backends: []Backend{tt.backend},
				}},
			}

			err := cfg.Validate()
			if (err != nil) != tt.hasErr {
				t.Errorf("Backend validation error = %v, hasErr %v", err, tt.hasErr)
			}

			if !tt.hasErr && cfg.Upstreams[0].Backends[0].Weight <= 0 {
				t.Error("Backend weight should be > 0 after validation")
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// temp directory for test files
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		yaml     string
		hasErr   bool
		expected func(*Config) bool
	}{
		{
			name: "valid config",
			yaml: `
version: "1.0.0"
service: "test-lb"
server:
  port: 8080
  read_timeout: "10s"
upstreams:
  - name: "api"
    algorithm: "round_robin"
    backends:
      - url: "http://localhost:3000"
        weight: 1
      - url: "http://localhost:3001"
        weight: 2
health:
  enabled: true
  interval: "30s"
  timeout: "5s"
  path: "/health"
metrics:
  enabled: true
  port: 9090
`,
			hasErr: false,
			expected: func(c *Config) bool {
				return c.Version == "1.0.0" &&
					c.Service == "test-lb" &&
					c.Server.Port == 8080 &&
					len(c.Upstreams) == 1 &&
					c.Upstreams[0].Name == "api" &&
					len(c.Upstreams[0].Backends) == 2
			},
		},
		{
			name: "invalid yaml",
			yaml: `
version: "1.0.0"
upstreams:
  - name: api
    backends: invalid_structure
`,
			hasErr: true,
		},
		{
			name: "config fails validation",
			yaml: `
version: "1.0.0"
service: "test-lb"
server:
  port: -1
upstreams: []
`,
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.yaml), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			config, err := LoadConfig(configPath)
			if (err != nil) != tt.hasErr {
				t.Errorf("LoadConfig() error = %v, hasErr %v", err, tt.hasErr)
				return
			}

			if !tt.hasErr {
				if config == nil {
					t.Error("Expected config to be non-nil")
					return
				}
				if tt.expected != nil && !tt.expected(config) {
					t.Error("Config validation failed")
				}
			}
		})
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	nonExistentPath := "/path/that/does/not/exist/config.yaml"
	config, err := LoadConfigWithDefaults(nonExistentPath)
	if err != nil {
		t.Errorf("LoadConfigWithDefaults() with non-existent file error = %v", err)
	}
	if config == nil {
		t.Error("Expected default config to be non-nil")
		return
	}
	if config.Service != "isame-lb" {
		t.Errorf("Expected default service name, got %s", config.Service)
	}

	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configYaml := `
version: "2.0.0"
service: "custom-lb"
server:
  port: 9080
upstreams:
  - name: "test"
    backends:
      - url: "http://localhost:4000"
`
	err = os.WriteFile(configPath, []byte(configYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err = LoadConfigWithDefaults(configPath)
	if err != nil {
		t.Errorf("LoadConfigWithDefaults() with existing file error = %v", err)
	}
	if config == nil {
		t.Error("Expected config to be non-nil")
		return
	}
	if config.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", config.Version)
	}
	if config.Service != "custom-lb" {
		t.Errorf("Expected service custom-lb, got %s", config.Service)
	}
}
