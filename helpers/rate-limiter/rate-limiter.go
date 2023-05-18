package rate_limiter

import (
	"context"
	"errors"
	"time"

	goLock "github.com/viney-shih/go-lock"
)

var (
	errRateLimit = errors.New("rate limit exceeded")
)

type RateLimiter struct {
	maxRPS int32
	curRPS int32

	mu goLock.Mutex
}

func New(ctx context.Context, maxRPS int32) *RateLimiter {

	rt := &RateLimiter{
		maxRPS: maxRPS,
		curRPS: 0,
		mu:     goLock.NewCASMutex(),
	}

	go rt.tickFunc(ctx)

	return rt
}

func (rt *RateLimiter) reset() {
	rt.curRPS = 0
}

func (rt *RateLimiter) tickFunc(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rt.reset()
		}
	}
}

func (rt *RateLimiter) Acquire(ctx context.Context) error {
	if !rt.mu.TryLockWithContext(ctx) {
		return ctx.Err()
	}
	defer rt.mu.Unlock()

	if rt.curRPS >= rt.maxRPS {
		return errRateLimit
	} else {
		rt.curRPS++
		return nil
	}
}
