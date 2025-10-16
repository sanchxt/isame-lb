package tls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager_Success(t *testing.T) {
	certPath := "testdata/server.crt"
	keyPath := "testdata/server.key"

	cfg := Config{
		CertPath: certPath,
		KeyPath:  keyPath,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v, want nil", err)
	}

	if mgr == nil {
		t.Fatal("NewManager() returned nil manager")
	}

	if mgr.certPath != certPath {
		t.Errorf("NewManager() certPath = %v, want %v", mgr.certPath, certPath)
	}

	if mgr.keyPath != keyPath {
		t.Errorf("NewManager() keyPath = %v, want %v", mgr.keyPath, keyPath)
	}
}

func TestNewManager_WithMinVersion(t *testing.T) {
	tests := []struct {
		name       string
		minVersion string
		want       uint16
		wantErr    bool
	}{
		{
			name:       "TLS 1.2",
			minVersion: "1.2",
			want:       tls.VersionTLS12,
			wantErr:    false,
		},
		{
			name:       "TLS 1.3",
			minVersion: "1.3",
			want:       tls.VersionTLS13,
			wantErr:    false,
		},
		{
			name:       "empty defaults to 1.2",
			minVersion: "",
			want:       tls.VersionTLS12,
			wantErr:    false,
		},
		{
			name:       "invalid version",
			minVersion: "1.1",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "invalid format",
			minVersion: "invalid",
			want:       0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				CertPath:   "testdata/server.crt",
				KeyPath:    "testdata/server.key",
				MinVersion: tt.minVersion,
			}

			mgr, err := NewManager(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && mgr.minVersion != tt.want {
				t.Errorf("NewManager() minVersion = %v, want %v", mgr.minVersion, tt.want)
			}
		})
	}
}

func TestLoadCertificate_Success(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/server.crt",
		KeyPath:  "testdata/server.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	cert, err := mgr.LoadCertificate()
	if err != nil {
		t.Fatalf("LoadCertificate() error = %v, want nil", err)
	}

	if len(cert.Certificate) == 0 {
		t.Error("LoadCertificate() returned empty certificate")
	}

	if cert.PrivateKey == nil {
		t.Error("LoadCertificate() returned nil private key")
	}
}

func TestLoadCertificate_FileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		certPath string
		keyPath  string
	}{
		{
			name:     "cert file not found",
			certPath: "testdata/nonexistent.crt",
			keyPath:  "testdata/server.key",
		},
		{
			name:     "key file not found",
			certPath: "testdata/server.crt",
			keyPath:  "testdata/nonexistent.key",
		},
		{
			name:     "both files not found",
			certPath: "testdata/nonexistent.crt",
			keyPath:  "testdata/nonexistent.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewManager(Config{
				CertPath: tt.certPath,
				KeyPath:  tt.keyPath,
			})
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}

			_, err = mgr.LoadCertificate()
			if err == nil {
				t.Error("LoadCertificate() error = nil, want error")
			}
		})
	}
}

func TestLoadCertificate_InvalidCertificate(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/invalid.crt",
		KeyPath:  "testdata/invalid.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	_, err = mgr.LoadCertificate()
	if err == nil {
		t.Error("LoadCertificate() error = nil, want error for invalid certificate")
	}
}

func TestLoadCertificate_MismatchedKeyPair(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/mismatched.crt",
		KeyPath:  "testdata/mismatched.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	_, err = mgr.LoadCertificate()
	if err == nil {
		t.Error("LoadCertificate() error = nil, want error for mismatched key pair")
	}
}

func TestGetTLSConfig_Success(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/server.crt",
		KeyPath:  "testdata/server.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tlsConfig, err := mgr.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig() error = %v, want nil", err)
	}

	if tlsConfig == nil {
		t.Fatal("GetTLSConfig() returned nil config")
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("GetTLSConfig() certificates count = %d, want 1", len(tlsConfig.Certificates))
	}

	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("GetTLSConfig() MinVersion = %v, want %v", tlsConfig.MinVersion, tls.VersionTLS12)
	}

	if !tlsConfig.PreferServerCipherSuites {
		t.Error("GetTLSConfig() PreferServerCipherSuites = false, want true")
	}
}

func TestGetTLSConfig_WithCustomMinVersion(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath:   "testdata/server.crt",
		KeyPath:    "testdata/server.key",
		MinVersion: "1.3",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tlsConfig, err := mgr.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig() error = %v, want nil", err)
	}

	if tlsConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("GetTLSConfig() MinVersion = %v, want %v", tlsConfig.MinVersion, tls.VersionTLS13)
	}
}

func TestGetTLSConfig_WithCipherSuites(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/server.crt",
		KeyPath:  "testdata/server.key",
		CipherSuites: []string{
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tlsConfig, err := mgr.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig() error = %v, want nil", err)
	}

	if len(tlsConfig.CipherSuites) != 2 {
		t.Errorf("GetTLSConfig() CipherSuites count = %d, want 2", len(tlsConfig.CipherSuites))
	}
}

func TestGetTLSConfig_InvalidCertificate(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/invalid.crt",
		KeyPath:  "testdata/invalid.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	_, err = mgr.GetTLSConfig()
	if err == nil {
		t.Error("GetTLSConfig() error = nil, want error for invalid certificate")
	}
}

func TestValidateCertificate_Success(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/server.crt",
		KeyPath:  "testdata/server.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = mgr.ValidateCertificate()
	if err != nil {
		t.Errorf("ValidateCertificate() error = %v, want nil", err)
	}
}

func TestValidateCertificate_FileNotFound(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/nonexistent.crt",
		KeyPath:  "testdata/server.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = mgr.ValidateCertificate()
	if err == nil {
		t.Error("ValidateCertificate() error = nil, want error")
	}
}

func TestValidateCertificate_InvalidCertificate(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/invalid.crt",
		KeyPath:  "testdata/invalid.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = mgr.ValidateCertificate()
	if err == nil {
		t.Error("ValidateCertificate() error = nil, want error for invalid certificate")
	}
}

func TestParseMinVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    uint16
		wantErr bool
	}{
		{
			name:    "TLS 1.2",
			version: "1.2",
			want:    tls.VersionTLS12,
			wantErr: false,
		},
		{
			name:    "TLS 1.3",
			version: "1.3",
			want:    tls.VersionTLS13,
			wantErr: false,
		},
		{
			name:    "empty string defaults to 1.2",
			version: "",
			want:    tls.VersionTLS12,
			wantErr: false,
		},
		{
			name:    "TLS 1.0 not supported",
			version: "1.0",
			want:    0,
			wantErr: true,
		},
		{
			name:    "TLS 1.1 not supported",
			version: "1.1",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid version string",
			version: "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "TLS 2.0 invalid",
			version: "2.0",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMinVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMinVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMinVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name    string
		ciphers []string
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil ciphers returns defaults",
			ciphers: nil,
			wantLen: 0, // Should use Go's default secure ciphers
			wantErr: false,
		},
		{
			name:    "empty ciphers returns defaults",
			ciphers: []string{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "valid cipher suite",
			ciphers: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "multiple valid ciphers",
			ciphers: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "invalid cipher suite",
			ciphers: []string{
				"INVALID_CIPHER_SUITE",
			},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "mix of valid and invalid",
			ciphers: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"INVALID_CIPHER_SUITE",
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCipherSuites(tt.ciphers)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCipherSuites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("parseCipherSuites() returned %d ciphers, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	mgr, err := NewManager(Config{
		CertPath: "testdata/server.crt",
		KeyPath:  "testdata/server.key",
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test concurrent access to GetTLSConfig
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := mgr.GetTLSConfig()
			if err != nil {
				t.Errorf("GetTLSConfig() error = %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewManager_EmptyPaths(t *testing.T) {
	tests := []struct {
		name     string
		certPath string
		keyPath  string
	}{
		{
			name:     "empty cert path",
			certPath: "",
			keyPath:  "testdata/server.key",
		},
		{
			name:     "empty key path",
			certPath: "testdata/server.crt",
			keyPath:  "",
		},
		{
			name:     "both paths empty",
			certPath: "",
			keyPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(Config{
				CertPath: tt.certPath,
				KeyPath:  tt.keyPath,
			})
			if err == nil {
				t.Error("NewManager() error = nil, want error for empty paths")
			}
		})
	}
}

func TestManager_RelativeAndAbsolutePaths(t *testing.T) {
	// Test with relative path
	t.Run("relative paths", func(t *testing.T) {
		mgr, err := NewManager(Config{
			CertPath: "testdata/server.crt",
			KeyPath:  "testdata/server.key",
		})
		if err != nil {
			t.Fatalf("NewManager() with relative paths error = %v", err)
		}

		_, err = mgr.LoadCertificate()
		if err != nil {
			t.Errorf("LoadCertificate() with relative paths error = %v", err)
		}
	})

	// Test with absolute path
	t.Run("absolute paths", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("os.Getwd() error = %v", err)
		}

		certPath := filepath.Join(wd, "testdata", "server.crt")
		keyPath := filepath.Join(wd, "testdata", "server.key")

		mgr, err := NewManager(Config{
			CertPath: certPath,
			KeyPath:  keyPath,
		})
		if err != nil {
			t.Fatalf("NewManager() with absolute paths error = %v", err)
		}

		_, err = mgr.LoadCertificate()
		if err != nil {
			t.Errorf("LoadCertificate() with absolute paths error = %v", err)
		}
	})
}
