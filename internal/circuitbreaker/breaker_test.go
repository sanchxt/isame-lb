package circuitbreaker

import (
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewCircuitBreaker(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	}

	cb := New(cfg)
	if cb == nil {
		t.Fatal("Expected circuit breaker to be non-nil")
	}

	if !cb.CanAttempt("http://test.com") {
		t.Error("New backend should be in closed state")
	}
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend := "http://test.com"

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	if !cb.CanAttempt(backend) {
		t.Error("Circuit should still be closed before threshold")
	}

	cb.RecordSuccess(backend)

	for i := 0; i < 3; i++ {
		cb.RecordFailure(backend)
	}

	if cb.CanAttempt(backend) {
		t.Error("Circuit should be open after threshold failures")
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend := "http://test.com"

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	if !cb.CanAttempt(backend) {
		t.Error("Circuit should be closed before threshold")
	}

	cb.RecordFailure(backend)

	if cb.CanAttempt(backend) {
		t.Error("Circuit should be open after threshold failures")
	}
}

func TestCircuitBreakerClosesAfterTimeout(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := New(cfg)
	backend := "http://test.com"

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	if cb.CanAttempt(backend) {
		t.Error("Circuit should be open after threshold failures")
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.CanAttempt(backend) {
		t.Error("Circuit should be closed after timeout")
	}
}

func TestCircuitBreakerMultipleBackends(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend1 := "http://backend1.com"
	backend2 := "http://backend2.com"

	cb.RecordFailure(backend1)
	cb.RecordFailure(backend1)

	if cb.CanAttempt(backend1) {
		t.Error("Circuit for backend1 should be open")
	}

	if !cb.CanAttempt(backend2) {
		t.Error("Circuit for backend2 should be closed")
	}
}

func TestCircuitBreakerDisabled(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          false,
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend := "http://test.com"

	for i := 0; i < 10; i++ {
		cb.RecordFailure(backend)
	}

	if !cb.CanAttempt(backend) {
		t.Error("Circuit breaker should allow all attempts when disabled")
	}
}

func TestCircuitBreakerGetState(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 2,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend := "http://test.com"

	if state := cb.GetState(backend); state != StateClosed {
		t.Errorf("Expected state %s, got %s", StateClosed, state)
	}

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	if state := cb.GetState(backend); state != StateOpen {
		t.Errorf("Expected state %s, got %s", StateOpen, state)
	}
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 3,
		Timeout:          1 * time.Second,
	}

	cb := New(cfg)
	backend := "http://test.com"

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	cb.RecordSuccess(backend)

	cb.RecordFailure(backend)
	cb.RecordFailure(backend)

	if !cb.CanAttempt(backend) {
		t.Error("Circuit should be closed after success reset failure count")
	}

	cb.RecordFailure(backend)

	if cb.CanAttempt(backend) {
		t.Error("Circuit should be open after threshold consecutive failures")
	}
}
