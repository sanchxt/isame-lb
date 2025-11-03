package retry

import (
	"math"
	"math/rand"
	"time"

	"github.com/sanchxt/isame-lb/internal/config"
)

type Retrier struct {
	config config.RetryConfig
	rand   *rand.Rand
}

func New(cfg config.RetryConfig) *Retrier {
	return &Retrier{
		config: cfg,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *Retrier) Do(fn func() error) error {
	var lastErr error

	maxAttempts := r.config.MaxAttempts
	if !r.config.Enabled {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < maxAttempts && r.ShouldRetry(err) {
			backoff := r.calculateBackoff(attempt)
			time.Sleep(backoff)
		}
	}

	return lastErr
}

func (r *Retrier) ShouldRetry(err error) bool {
	return err != nil
}

func (r *Retrier) calculateBackoff(attempt int) time.Duration {
	backoff := float64(r.config.InitialBackoff) * math.Pow(2, float64(attempt-1))

	if backoff > float64(r.config.MaxBackoff) {
		backoff = float64(r.config.MaxBackoff)
	}

	jitter := r.rand.Float64()*0.5 + 0.75 // 0.75 to 1.25
	backoff = backoff * jitter

	return time.Duration(backoff)
}

func (r *Retrier) CalculateBackoff(attempt int) time.Duration {
	return r.calculateBackoff(attempt)
}
