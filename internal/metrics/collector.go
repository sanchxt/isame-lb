package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sanchxt/isame-lb/internal/config"
)

type Collector struct {
	config   config.MetricsConfig
	server   *http.Server
	registry *prometheus.Registry

	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	upstreamHealthy   *prometheus.GaugeVec
	connectionsActive prometheus.Gauge

	mu sync.RWMutex
}

func NewCollector(cfg config.MetricsConfig) *Collector {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "isame_lb_requests_total",
			Help: "Total number of requests processed by the load balancer",
		},
		[]string{"upstream", "backend", "method", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "isame_lb_request_duration_seconds",
			Help:    "Time spent processing requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"upstream", "backend", "method"},
	)

	upstreamHealthy := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "isame_lb_upstream_healthy",
			Help: "Whether upstream backend is healthy (1 = healthy, 0 = unhealthy)",
		},
		[]string{"upstream", "backend"},
	)

	connectionsActive := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "isame_lb_active_connections",
			Help: "Current number of active connections",
		},
	)

	registry.MustRegister(requestsTotal)
	registry.MustRegister(requestDuration)
	registry.MustRegister(upstreamHealthy)
	registry.MustRegister(connectionsActive)

	return &Collector{
		config:            cfg,
		registry:          registry,
		requestsTotal:     requestsTotal,
		requestDuration:   requestDuration,
		upstreamHealthy:   upstreamHealthy,
		connectionsActive: connectionsActive,
	}
}

func (c *Collector) Start() error {
	if !c.config.Enabled {
		log.Println("Metrics collector disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(c.config.Path, promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{}))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", c.config.Port)
	c.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Starting metrics server on %s%s", addr, c.config.Path)

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	return nil
}

func (c *Collector) Stop() error {
	if c.server == nil {
		return nil
	}

	log.Println("Stopping metrics server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.server.Shutdown(ctx); err != nil {
		log.Printf("Error stopping metrics server: %v", err)
		return err
	}

	log.Println("Metrics server stopped")
	return nil
}

func (c *Collector) RecordRequest(upstream, backend, method, status string, duration time.Duration) {
	if !c.config.Enabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	c.requestsTotal.WithLabelValues(upstream, backend, method, status).Inc()
	c.requestDuration.WithLabelValues(upstream, backend, method).Observe(duration.Seconds())
}

func (c *Collector) UpdateBackendHealth(upstream, backend string, healthy bool) {
	if !c.config.Enabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	value := 0.0
	if healthy {
		value = 1.0
	}
	c.upstreamHealthy.WithLabelValues(upstream, backend).Set(value)
}

func (c *Collector) SetActiveConnections(count int) {
	if !c.config.Enabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	c.connectionsActive.Set(float64(count))
}

func (c *Collector) IncrementActiveConnections() {
	if !c.config.Enabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	c.connectionsActive.Inc()
}

func (c *Collector) DecrementActiveConnections() {
	if !c.config.Enabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	c.connectionsActive.Dec()
}
