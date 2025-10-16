package balancer

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/sanchxt/isame-lb/internal/config"
)

var (
	ErrNoHealthyBackends = errors.New("no healthy backends available")
	ErrInvalidAlgorithm  = errors.New("invalid load balancing algorithm")
)

type LoadBalancer interface {
	SelectBackend(request *http.Request, backends []config.Backend, healthStatus map[string]bool) (*config.Backend, error)

	Algorithm() string
}

func NewLoadBalancer(algorithm string) (LoadBalancer, error) {
	switch algorithm {
	case "round_robin", "":
		return NewRoundRobin(), nil
	case "weighted_round_robin":
		return NewWeightedRoundRobin(), nil
	case "least_connections":
		return NewLeastConnections(), nil
	default:
		return nil, ErrInvalidAlgorithm
	}
}

type RoundRobin struct {
	counter uint64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (rr *RoundRobin) SelectBackend(request *http.Request, backends []config.Backend, healthStatus map[string]bool) (*config.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	var healthyBackends []config.Backend
	for _, backend := range backends {
		if healthy, exists := healthStatus[backend.URL]; !exists || healthy {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	next := atomic.AddUint64(&rr.counter, 1)
	index := (next - 1) % uint64(len(healthyBackends))

	return &healthyBackends[index], nil
}

func (rr *RoundRobin) Algorithm() string {
	return "round_robin"
}

type WeightedRoundRobin struct {
	mu      sync.Mutex
	weights map[string]int
}

func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{
		weights: make(map[string]int),
	}
}

func (wrr *WeightedRoundRobin) SelectBackend(request *http.Request, backends []config.Backend, healthStatus map[string]bool) (*config.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	var healthyBackends []config.Backend
	for _, backend := range backends {
		if healthy, exists := healthStatus[backend.URL]; !exists || healthy {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	for _, backend := range healthyBackends {
		if _, exists := wrr.weights[backend.URL]; !exists {
			wrr.weights[backend.URL] = 0
		}
	}

	totalWeight := 0
	for _, backend := range healthyBackends {
		totalWeight += backend.Weight
		wrr.weights[backend.URL] += backend.Weight
	}

	var selected *config.Backend
	maxWeight := -1
	for i := range healthyBackends {
		backend := &healthyBackends[i]
		if wrr.weights[backend.URL] > maxWeight {
			maxWeight = wrr.weights[backend.URL]
			selected = backend
		}
	}

	if selected == nil {
		return nil, ErrNoHealthyBackends
	}

	wrr.weights[selected.URL] -= totalWeight

	return selected, nil
}

func (wrr *WeightedRoundRobin) Algorithm() string {
	return "weighted_round_robin"
}

type LeastConnections struct {
	mu          sync.RWMutex
	connections map[string]int64
}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{
		connections: make(map[string]int64),
	}
}

func (lc *LeastConnections) SelectBackend(request *http.Request, backends []config.Backend, healthStatus map[string]bool) (*config.Backend, error) {
	if len(backends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	lc.mu.RLock()
	defer lc.mu.RUnlock()

	var healthyBackends []config.Backend
	for _, backend := range backends {
		if healthy, exists := healthStatus[backend.URL]; !exists || healthy {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil, ErrNoHealthyBackends
	}

	var selected *config.Backend
	minConnections := int64(-1)

	for i := range healthyBackends {
		backend := &healthyBackends[i]
		connections := lc.connections[backend.URL]

		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	if selected == nil {
		return nil, ErrNoHealthyBackends
	}

	return selected, nil
}

func (lc *LeastConnections) IncrementConnections(backendURL string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.connections[backendURL]++
}

func (lc *LeastConnections) DecrementConnections(backendURL string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.connections[backendURL] > 0 {
		lc.connections[backendURL]--
	}
}

func (lc *LeastConnections) GetConnections(backendURL string) int64 {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.connections[backendURL]
}

func (lc *LeastConnections) Algorithm() string {
	return "least_connections"
}
