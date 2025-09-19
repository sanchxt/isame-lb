package health

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

type Status struct {
	Healthy              bool
	LastCheck            time.Time
	ConsecutiveSuccesses int
	ConsecutiveFailures  int
	mu                   sync.RWMutex
}

type Checker struct {
	config      config.HealthConfig
	statuses    map[string]*Status
	statusMutex sync.RWMutex
	client      *http.Client
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewChecker(cfg config.HealthConfig) *Checker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Checker{
		config:   cfg,
		statuses: make(map[string]*Status),
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (hc *Checker) Start(upstreams []config.Upstream) {
	if !hc.config.Enabled {
		log.Println("Health checker disabled")
		return
	}

	hc.statusMutex.Lock()
	for _, upstream := range upstreams {
		for _, backend := range upstream.Backends {
			if _, exists := hc.statuses[backend.URL]; !exists {
				hc.statuses[backend.URL] = &Status{
					Healthy:   true,
					LastCheck: time.Now(),
				}
			}
		}
	}
	hc.statusMutex.Unlock()

	for _, upstream := range upstreams {
		for _, backend := range upstream.Backends {
			hc.wg.Add(1)
			go hc.checkBackend(backend.URL)
		}
	}

	log.Printf("Health checker started with %d backends", len(hc.statuses))
}

func (hc *Checker) Stop() {
	log.Println("Stopping health checker...")
	hc.cancel()
	hc.wg.Wait()
	log.Println("Health checker stopped")
}

func (hc *Checker) IsHealthy(backendURL string) bool {
	hc.statusMutex.RLock()
	defer hc.statusMutex.RUnlock()

	status, exists := hc.statuses[backendURL]
	if !exists {
		return true
	}

	status.mu.RLock()
	defer status.mu.RUnlock()
	return status.Healthy
}

func (hc *Checker) GetStatus(backendURL string) *Status {
	hc.statusMutex.RLock()
	defer hc.statusMutex.RUnlock()

	status, exists := hc.statuses[backendURL]
	if !exists {
		return &Status{Healthy: true, LastCheck: time.Time{}}
	}

	status.mu.RLock()
	defer status.mu.RUnlock()
	return &Status{
		Healthy:              status.Healthy,
		LastCheck:            status.LastCheck,
		ConsecutiveSuccesses: status.ConsecutiveSuccesses,
		ConsecutiveFailures:  status.ConsecutiveFailures,
	}
}

func (hc *Checker) GetAllStatuses() map[string]bool {
	hc.statusMutex.RLock()
	defer hc.statusMutex.RUnlock()

	result := make(map[string]bool, len(hc.statuses))
	for url, status := range hc.statuses {
		status.mu.RLock()
		result[url] = status.Healthy
		status.mu.RUnlock()
	}

	return result
}

func (hc *Checker) checkBackend(backendURL string) {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	log.Printf("Starting health checks for %s", backendURL)

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthCheck(backendURL)
		}
	}
}

func (hc *Checker) performHealthCheck(backendURL string) {
	ctx, cancel := context.WithTimeout(hc.ctx, hc.config.Timeout)
	defer cancel()

	healthURL := backendURL + hc.config.Path

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		hc.updateBackendStatus(backendURL, false)
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		hc.updateBackendStatus(backendURL, false)
		return
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	hc.updateBackendStatus(backendURL, healthy)
}

func (hc *Checker) updateBackendStatus(backendURL string, healthy bool) {
	hc.statusMutex.RLock()
	status, exists := hc.statuses[backendURL]
	hc.statusMutex.RUnlock()

	if !exists {
		return
	}

	status.mu.Lock()
	defer status.mu.Unlock()

	status.LastCheck = time.Now()
	previouslyHealthy := status.Healthy

	if healthy {
		status.ConsecutiveSuccesses++
		status.ConsecutiveFailures = 0

		if !status.Healthy && status.ConsecutiveSuccesses >= hc.config.HealthyThreshold {
			status.Healthy = true
			log.Printf("Backend %s marked as HEALTHY (%d consecutive successes)",
				backendURL, status.ConsecutiveSuccesses)
		}
	} else {
		status.ConsecutiveFailures++
		status.ConsecutiveSuccesses = 0

		if status.Healthy && status.ConsecutiveFailures >= hc.config.UnhealthyThreshold {
			status.Healthy = false
			log.Printf("Backend %s marked as UNHEALTHY (%d consecutive failures)",
				backendURL, status.ConsecutiveFailures)
		}
	}

	if previouslyHealthy != status.Healthy {
		if status.Healthy {
			log.Printf("✓ Backend %s recovered", backendURL)
		} else {
			log.Printf("✗ Backend %s failed", backendURL)
		}
	}
}
