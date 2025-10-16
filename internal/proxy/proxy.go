package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/sanchxt/isame-lb/internal/balancer"
	"github.com/sanchxt/isame-lb/internal/circuitbreaker"
	"github.com/sanchxt/isame-lb/internal/config"
	"github.com/sanchxt/isame-lb/internal/health"
	"github.com/sanchxt/isame-lb/internal/metrics"
	"github.com/sanchxt/isame-lb/internal/ratelimit"
	"github.com/sanchxt/isame-lb/internal/retry"
)

type Handler struct {
	config         *config.Config
	loadBalancers  map[string]balancer.LoadBalancer
	healthChecker  *health.Checker
	metrics        *metrics.Collector
	circuitBreaker *circuitbreaker.CircuitBreaker
	retrier        *retry.Retrier
	rateLimiters   map[string]*ratelimit.RateLimiter // per-upstream rate limiters
}

func NewHandler(cfg *config.Config, healthChecker *health.Checker, metricsCollector *metrics.Collector) (*Handler, error) {
	loadBalancers := make(map[string]balancer.LoadBalancer)
	rateLimiters := make(map[string]*ratelimit.RateLimiter)

	for _, upstream := range cfg.Upstreams {
		lb, err := balancer.NewLoadBalancer(upstream.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer for upstream %s: %w", upstream.Name, err)
		}
		loadBalancers[upstream.Name] = lb

		if upstream.RateLimit != nil {
			rateLimiters[upstream.Name] = ratelimit.New(upstream.RateLimit)
		}
	}

	return &Handler{
		config:         cfg,
		loadBalancers:  loadBalancers,
		healthChecker:  healthChecker,
		metrics:        metricsCollector,
		circuitBreaker: circuitbreaker.New(cfg.CircuitBreaker),
		retrier:        retry.New(cfg.Retry),
		rateLimiters:   rateLimiters,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h.metrics != nil {
		h.metrics.IncrementActiveConnections()
		defer h.metrics.DecrementActiveConnections()
	}

	if len(h.config.Upstreams) == 0 {
		h.writeError(w, r, "No upstreams configured", http.StatusServiceUnavailable, start)
		return
	}

	upstream := &h.config.Upstreams[0]

	clientIP := getClientIP(r)
	if rateLimiter, exists := h.rateLimiters[upstream.Name]; exists {
		if !rateLimiter.Allow(clientIP) {
			h.writeError(w, r, "Rate limit exceeded", http.StatusTooManyRequests, start)
			return
		}
	}

	lb := h.loadBalancers[upstream.Name]

	var healthStatus map[string]bool
	if h.healthChecker != nil {
		healthStatus = h.healthChecker.GetAllStatuses()
	} else {
		healthStatus = make(map[string]bool)
	}

	var wrappedWriter *responseWriter
	var lastBackendURL string

	err := h.retrier.Do(func() error {
		selectedBackend, err := lb.SelectBackend(r, upstream.Backends, healthStatus)
		if err != nil {
			return err
		}

		lastBackendURL = selectedBackend.URL

		if !h.circuitBreaker.CanAttempt(selectedBackend.URL) {
			log.Printf("Circuit breaker open for backend %s", selectedBackend.URL)
			return fmt.Errorf("circuit breaker open for %s", selectedBackend.URL)
		}

		if lcLB, ok := lb.(*balancer.LeastConnections); ok {
			lcLB.IncrementConnections(selectedBackend.URL)
			defer lcLB.DecrementConnections(selectedBackend.URL)
		}

		backendURL, err := url.Parse(selectedBackend.URL)
		if err != nil {
			return fmt.Errorf("invalid backend URL: %w", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(backendURL)

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			h.setProxyHeaders(req, r)
		}

		proxyErr := false
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("Proxy error for backend %s: %v", selectedBackend.URL, err)
			proxyErr = true
		}

		wrappedWriter = &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		proxy.ServeHTTP(wrappedWriter, r)

		if proxyErr || wrappedWriter.statusCode >= 500 {
			h.circuitBreaker.RecordFailure(selectedBackend.URL)
			return fmt.Errorf("backend error: status %d", wrappedWriter.statusCode)
		}

		h.circuitBreaker.RecordSuccess(selectedBackend.URL)
		return nil
	})

	if err != nil {
		if wrappedWriter == nil || wrappedWriter.statusCode == http.StatusOK {
			h.writeError(w, r, "Service temporarily unavailable", http.StatusServiceUnavailable, start)
		}
		return
	}

	if h.metrics != nil && wrappedWriter != nil {
		duration := time.Since(start)
		status := strconv.Itoa(wrappedWriter.statusCode)
		h.metrics.RecordRequest(upstream.Name, lastBackendURL, r.Method, status, duration)
	}
}

func (h *Handler) setProxyHeaders(proxyReq *http.Request, originalReq *http.Request) {
	if clientIP := getClientIP(originalReq); clientIP != "" {
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}

	scheme := "http"
	if originalReq.TLS != nil {
		scheme = "https"
	}
	proxyReq.Header.Set("X-Forwarded-Proto", scheme)

	proxyReq.Header.Set("X-Forwarded-Host", originalReq.Host)

	proxyReq.Header.Set("X-Load-Balancer", h.config.Service)
}

func getClientIP(r *http.Request) string {
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		return xForwardedFor
	}

	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	return r.RemoteAddr
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, message string, statusCode int, start time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := fmt.Sprintf(`{"error":"%s","code":%d}`, message, statusCode)
	w.Write([]byte(errorResponse))

	if h.metrics != nil && len(h.config.Upstreams) > 0 {
		duration := time.Since(start)
		status := strconv.Itoa(statusCode)
		h.metrics.RecordRequest(h.config.Upstreams[0].Name, "error", r.Method, status, duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
