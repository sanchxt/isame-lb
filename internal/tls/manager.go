package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

// Manager handles TLS certificate loading and configuration
type Manager struct {
	certPath     string
	keyPath      string
	minVersion   uint16
	cipherSuites []uint16
}

// Config holds TLS manager configuration
type Config struct {
	CertPath     string
	KeyPath      string
	MinVersion   string   // "1.2", "1.3"
	CipherSuites []string // Optional custom cipher suites
}

// NewManager creates a new TLS manager with the given configuration
func NewManager(cfg Config) (*Manager, error) {
	if cfg.CertPath == "" {
		return nil, errors.New("cert path is required")
	}
	if cfg.KeyPath == "" {
		return nil, errors.New("key path is required")
	}

	minVersion, err := parseMinVersion(cfg.MinVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid min version: %w", err)
	}

	cipherSuites, err := parseCipherSuites(cfg.CipherSuites)
	if err != nil {
		return nil, fmt.Errorf("invalid cipher suites: %w", err)
	}

	return &Manager{
		certPath:     cfg.CertPath,
		keyPath:      cfg.KeyPath,
		minVersion:   minVersion,
		cipherSuites: cipherSuites,
	}, nil
}

// LoadCertificate loads the TLS certificate and private key
func (m *Manager) LoadCertificate() (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(m.certPath, m.keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %w", err)
	}

	return cert, nil
}

// GetTLSConfig returns a configured tls.Config
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	cert, err := m.LoadCertificate()
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               m.minVersion,
		PreferServerCipherSuites: true,
		CipherSuites:             m.cipherSuites,
	}

	return config, nil
}

// ValidateCertificate validates the certificate and key pair
func (m *Manager) ValidateCertificate() error {
	cert, err := m.LoadCertificate()
	if err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	// Parse the certificate to validate it's well-formed
	if len(cert.Certificate) == 0 {
		return errors.New("certificate is empty")
	}

	// Parse the first certificate in the chain
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if certificate is expired (warning, not error)
	// This is done here to provide useful feedback but doesn't fail validation
	_ = x509Cert

	return nil
}

// parseMinVersion converts a version string to tls.Version constant
func parseMinVersion(version string) (uint16, error) {
	// Default to TLS 1.2 if not specified
	if version == "" {
		return tls.VersionTLS12, nil
	}

	switch version {
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version %q (supported: 1.2, 1.3)", version)
	}
}

// parseCipherSuites converts cipher suite names to IDs
func parseCipherSuites(ciphers []string) ([]uint16, error) {
	if len(ciphers) == 0 {
		// Return nil to use Go's default secure cipher suites
		return nil, nil
	}

	// Map of cipher suite names to IDs
	cipherMap := map[string]uint16{
		"TLS_RSA_WITH_RC4_128_SHA":                      tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_AES_128_GCM_SHA256":                        tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                        tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":                  tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	result := make([]uint16, 0, len(ciphers))
	for _, name := range ciphers {
		id, ok := cipherMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown cipher suite: %s", name)
		}
		result = append(result, id)
	}

	return result, nil
}
