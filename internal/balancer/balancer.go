package balancer

import (
	"errors"
	"net/http"
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
