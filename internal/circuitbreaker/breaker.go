package circuitbreaker

import (
	"sync"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

type State string

const (
	StateClosed State = "closed"
	StateOpen   State = "open"
)

type backendState struct {
	state               State
	consecutiveFailures int
	lastFailureTime     time.Time
}

type CircuitBreaker struct {
	config   config.CircuitBreakerConfig
	mu       sync.RWMutex
	backends map[string]*backendState
}

func New(cfg config.CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:   cfg,
		backends: make(map[string]*backendState),
	}
}

func (cb *CircuitBreaker) CanAttempt(backendURL string) bool {
	if !cb.config.Enabled {
		return true
	}

	cb.mu.RLock()
	state, exists := cb.backends[backendURL]
	cb.mu.RUnlock()

	if !exists {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if state.state == StateOpen {
		if time.Since(state.lastFailureTime) >= cb.config.Timeout {
			state.state = StateClosed
			state.consecutiveFailures = 0
			return true
		}
		return false
	}

	return true
}

func (cb *CircuitBreaker) RecordSuccess(backendURL string) {
	if !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, exists := cb.backends[backendURL]
	if !exists {
		return
	}

	state.consecutiveFailures = 0
	state.state = StateClosed
}

func (cb *CircuitBreaker) RecordFailure(backendURL string) {
	if !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, exists := cb.backends[backendURL]
	if !exists {
		state = &backendState{
			state:               StateClosed,
			consecutiveFailures: 0,
		}
		cb.backends[backendURL] = state
	}

	state.consecutiveFailures++
	state.lastFailureTime = time.Now()

	if state.consecutiveFailures >= cb.config.FailureThreshold {
		state.state = StateOpen
	}
}

func (cb *CircuitBreaker) GetState(backendURL string) State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, exists := cb.backends[backendURL]
	if !exists {
		return StateClosed
	}

	return state.state
}

func (cb *CircuitBreaker) Reset(backendURL string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, exists := cb.backends[backendURL]
	if !exists {
		return
	}

	state.state = StateClosed
	state.consecutiveFailures = 0
}
