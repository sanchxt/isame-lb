package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNewRetrier(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	r := New(cfg)
	if r == nil {
		t.Fatal("Expected retrier to be non-nil")
	}
}

func TestRetrierSuccess(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0

	err := r.Do(func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetrierFailureThenSuccess(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0

	err := r.Do(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetrierMaxAttemptsExceeded(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0
	expectedErr := errors.New("persistent failure")

	err := r.Do(func() error {
		attempts++
		return expectedErr
	})

	if err == nil {
		t.Error("Expected error after max attempts")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRetrierExponentialBackoff(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    4,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0
	attemptTimes := []time.Time{}

	err := r.Do(func() error {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())
		if attempts < 4 {
			return errors.New("retry")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}

	if len(attemptTimes) >= 2 {
		delay1 := attemptTimes[1].Sub(attemptTimes[0])
		if delay1 < 30*time.Millisecond || delay1 > 150*time.Millisecond {
			t.Logf("First delay: %v (expected ~50ms)", delay1)
		}
	}

	if len(attemptTimes) >= 3 {
		delay2 := attemptTimes[2].Sub(attemptTimes[1])
		if delay2 < 50*time.Millisecond || delay2 > 250*time.Millisecond {
			t.Logf("Second delay: %v (expected ~100ms)", delay2)
		}
	}
}

func TestRetrierDisabled(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        false,
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0

	err := r.Do(func() error {
		attempts++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error when retries disabled")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt when disabled, got %d", attempts)
	}
}

func TestRetrierMaxBackoff(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
	}

	r := New(cfg)
	attempts := 0
	attemptTimes := []time.Time{}

	err := r.Do(func() error {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())
		if attempts < 5 {
			return errors.New("retry")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(attemptTimes) >= 4 {
		delay3 := attemptTimes[3].Sub(attemptTimes[2])
		if delay3 > 300*time.Millisecond {
			t.Errorf("Backoff exceeded max: %v", delay3)
		}
	}
}

func TestRetrierShouldRetry(t *testing.T) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	r := New(cfg)

	retryableErrors := []error{
		errors.New("connection refused"),
		errors.New("timeout"),
		errors.New("temporary failure"),
	}

	for _, err := range retryableErrors {
		if !r.ShouldRetry(err) {
			t.Errorf("Expected error %v to be retryable", err)
		}
	}
}
