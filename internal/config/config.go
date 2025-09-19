package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// main config for the load balancer
type Config struct {
	Version   string        `yaml:"version" json:"version"`
	Service   string        `yaml:"service" json:"service"`
	Server    ServerConfig  `yaml:"server" json:"server"`
	Upstreams []Upstream    `yaml:"upstreams" json:"upstreams"`
	Health    HealthConfig  `yaml:"health" json:"health"`
	Metrics   MetricsConfig `yaml:"metrics" json:"metrics"`
}

// server settings
type ServerConfig struct {
	Port           int           `yaml:"port" json:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" json:"max_header_bytes"`
}

// server group
type Upstream struct {
	Name      string    `yaml:"name" json:"name"`
	Algorithm string    `yaml:"algorithm" json:"algorithm"`
	Backends  []Backend `yaml:"backends" json:"backends"`
}

// individual server
type Backend struct {
	URL    string `yaml:"url" json:"url"`
	Weight int    `yaml:"weight" json:"weight"`
}

// health check config
type HealthConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	Interval           time.Duration `yaml:"interval" json:"interval"`
	Timeout            time.Duration `yaml:"timeout" json:"timeout"`
	Path               string        `yaml:"path" json:"path"`
	UnhealthyThreshold int           `yaml:"unhealthy_threshold" json:"unhealthy_threshold"`
	HealthyThreshold   int           `yaml:"healthy_threshold" json:"healthy_threshold"`
}

// metrics config
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Port    int    `yaml:"port" json:"port"`
	Path    string `yaml:"path" json:"path"`
}

// config with defaults
func NewDefaultConfig() *Config {
	return &Config{
		Version: "0.1.0",
		Service: "isame-lb",
		Server: ServerConfig{
			Port:           8080,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1MB
		},
		Upstreams: []Upstream{},
		Health: HealthConfig{
			Enabled:            true,
			Interval:           30 * time.Second,
			Timeout:            5 * time.Second,
			Path:               "/health",
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
	}
}

func (c *Config) Validate() error {
	// apply defaults
	if c.Service == "" {
		c.Service = "isame-lb"
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}

	// validate server config
	if err := c.validateServerConfig(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	// validate upstreams
	if err := c.validateUpstreams(); err != nil {
		return fmt.Errorf("upstreams validation failed: %w", err)
	}

	// validate health config
	if err := c.validateHealthConfig(); err != nil {
		return fmt.Errorf("health config validation failed: %w", err)
	}

	// validate metrics config
	if err := c.validateMetricsConfig(); err != nil {
		return fmt.Errorf("metrics config validation failed: %w", err)
	}

	return nil
}

func (c *Config) validateServerConfig() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return errors.New("server port must be between 1 and 65535")
	}

	if c.Server.ReadTimeout <= 0 {
		c.Server.ReadTimeout = 15 * time.Second
	}
	if c.Server.WriteTimeout <= 0 {
		c.Server.WriteTimeout = 15 * time.Second
	}
	if c.Server.IdleTimeout <= 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Server.MaxHeaderBytes <= 0 {
		c.Server.MaxHeaderBytes = 1 << 20 // 1MB
	}

	return nil
}

func (c *Config) validateUpstreams() error {
	if len(c.Upstreams) == 0 {
		return errors.New("at least one upstream must be configured")
	}

	for i, upstream := range c.Upstreams {
		if upstream.Name == "" {
			return fmt.Errorf("upstream[%d]: name is required", i)
		}

		if upstream.Algorithm == "" {
			c.Upstreams[i].Algorithm = "round_robin"
		}

		if len(upstream.Backends) == 0 {
			return fmt.Errorf("upstream[%d]: at least one backend is required", i)
		}

		for j, backend := range upstream.Backends {
			if err := c.validateBackend(backend, i, j); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) validateBackend(backend Backend, upstreamIdx, backendIdx int) error {
	if backend.URL == "" {
		return fmt.Errorf("upstream[%d].backend[%d]: URL is required", upstreamIdx, backendIdx)
	}

	parsedURL, err := url.Parse(backend.URL)
	if err != nil {
		return fmt.Errorf("upstream[%d].backend[%d]: invalid URL %q: %w", upstreamIdx, backendIdx, backend.URL, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("upstream[%d].backend[%d]: URL scheme must be http or https", upstreamIdx, backendIdx)
	}

	if backend.Weight <= 0 {
		c.Upstreams[upstreamIdx].Backends[backendIdx].Weight = 1
	}

	return nil
}

func (c *Config) validateHealthConfig() error {
	if c.Health.Interval <= 0 {
		c.Health.Interval = 30 * time.Second
	}
	if c.Health.Timeout <= 0 {
		c.Health.Timeout = 5 * time.Second
	}
	if c.Health.Path == "" {
		c.Health.Path = "/health"
	}
	if c.Health.UnhealthyThreshold <= 0 {
		c.Health.UnhealthyThreshold = 3
	}
	if c.Health.HealthyThreshold <= 0 {
		c.Health.HealthyThreshold = 2
	}

	return nil
}

func (c *Config) validateMetricsConfig() error {
	if c.Metrics.Enabled {
		if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
			c.Metrics.Port = 9090
		}
		if c.Metrics.Path == "" {
			c.Metrics.Path = "/metrics"
		}
	}

	return nil
}

/*
 * loads config from yaml
 */
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

/*
 * loads config from file
 * if it doesnt exist, return default config
 */
func LoadConfigWithDefaults(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewDefaultConfig(), nil
	}

	return LoadConfig(path)
}
