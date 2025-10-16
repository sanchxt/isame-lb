package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
	"github.com/sanchxt/isame-lb/internal/health"
	"github.com/sanchxt/isame-lb/internal/metrics"
	"github.com/sanchxt/isame-lb/internal/proxy"
	"github.com/sanchxt/isame-lb/internal/tls"
)

type LoadBalancerServer struct {
	config        *config.Config
	httpServer    *http.Server
	httpsServer   *http.Server
	healthChecker *health.Checker
	metrics       *metrics.Collector
	proxy         *proxy.Handler
	tlsManager    *tls.Manager
}

func New(cfg *config.Config) (*LoadBalancerServer, error) {
	healthChecker := health.NewChecker(cfg.Health)

	metricsCollector := metrics.NewCollector(cfg.Metrics)

	proxyHandler, err := proxy.NewHandler(cfg, healthChecker, metricsCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}

	var tlsMgr *tls.Manager
	if cfg.TLS.Enabled {
		tlsMgr, err = tls.NewManager(tls.Config{
			CertPath:     cfg.TLS.CertFile,
			KeyPath:      cfg.TLS.KeyFile,
			MinVersion:   cfg.TLS.MinVersion,
			CipherSuites: cfg.TLS.CipherSuites,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize TLS: %w", err)
		}

		if err := tlsMgr.ValidateCertificate(); err != nil {
			return nil, fmt.Errorf("certificate validation failed: %w", err)
		}
	}

	return &LoadBalancerServer{
		config:        cfg,
		healthChecker: healthChecker,
		metrics:       metricsCollector,
		proxy:         proxyHandler,
		tlsManager:    tlsMgr,
	}, nil
}

func (s *LoadBalancerServer) Start() error {
	log.Printf("Starting %s v%s", s.config.Service, s.config.Version)

	if err := s.metrics.Start(); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	s.healthChecker.Start(s.config.Upstreams)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/status", s.statusHandler)
	mux.Handle("/", s.proxy)

	httpAddr := fmt.Sprintf(":%d", s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:           httpAddr,
		Handler:        mux,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		IdleTimeout:    s.config.Server.IdleTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	log.Printf("HTTP server starting on %s", httpAddr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	if s.config.TLS.Enabled && s.tlsManager != nil {
		httpsAddr := fmt.Sprintf(":%d", s.config.Server.HTTPSPort)

		tlsConfig, err := s.tlsManager.GetTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to get TLS config: %w", err)
		}

		s.httpsServer = &http.Server{
			Addr:           httpsAddr,
			Handler:        mux,
			TLSConfig:      tlsConfig,
			ReadTimeout:    s.config.Server.ReadTimeout,
			WriteTimeout:   s.config.Server.WriteTimeout,
			IdleTimeout:    s.config.Server.IdleTimeout,
			MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
		}

		log.Printf("HTTPS server starting on %s", httpsAddr)
		go func() {
			if err := s.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		}()
	}

	s.waitForShutdown()

	return nil
}

func (s *LoadBalancerServer) Shutdown(ctx context.Context) error {
	log.Println("Shutting down load balancer...")

	if s.httpServer != nil {
		log.Println("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		}
	}

	if s.httpsServer != nil {
		log.Println("Shutting down HTTPS server...")
		if err := s.httpsServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTPS server: %v", err)
		}
	}

	s.healthChecker.Stop()

	if err := s.metrics.Stop(); err != nil {
		log.Printf("Error stopping metrics server: %v", err)
	}

	log.Println("Load balancer shut down complete")
	return nil
}

func (s *LoadBalancerServer) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.Shutdown(ctx)
}

func (s *LoadBalancerServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"` + s.config.Service + `"}`))
}

func (s *LoadBalancerServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	statuses := s.healthChecker.GetAllStatuses()

	healthyCount := 0
	totalCount := 0

	for _, upstream := range s.config.Upstreams {
		for _, backend := range upstream.Backends {
			totalCount++
			if healthy, exists := statuses[backend.URL]; exists && healthy {
				healthyCount++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := fmt.Sprintf(`{
		"service": "%s",
		"version": "%s",
		"upstreams": %d,
		"backends": {
			"total": %d,
			"healthy": %d,
			"unhealthy": %d
		},
		"health_checks_enabled": %t,
		"metrics_enabled": %t
	}`,
		s.config.Service,
		s.config.Version,
		len(s.config.Upstreams),
		totalCount,
		healthyCount,
		totalCount-healthyCount,
		s.config.Health.Enabled,
		s.config.Metrics.Enabled,
	)

	w.Write([]byte(status))
}
