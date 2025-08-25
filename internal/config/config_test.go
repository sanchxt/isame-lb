package config

import "testing"

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
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		hasErr bool
	}{
		{
			name:   "valid config",
			config: &Config{Version: "1.0.0", Service: "test-service"},
			hasErr: false,
		},
		{
			name:   "empty config gets defaults",
			config: &Config{},
			hasErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.hasErr {
				t.Errorf("Validate() error = %v, hasErr %v", err, tt.hasErr)
			}

			// Check that defaults were applied
			if tt.config.Service == "" {
				t.Error("Service should not be empty after validation")
			}
			if tt.config.Version == "" {
				t.Error("Version should not be empty after validation")
			}
		})
	}
}
