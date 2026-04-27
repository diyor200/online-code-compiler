package rate_limiter

import (
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type RateLimiter interface {
	GetLimiter(ip string) *rate.Limiter
	CleanUp()
}

type rateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(r rate.Limit, burst int) RateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

func (r *rateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, ok := r.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(r.rate, r.burst)
		r.limiters[ip] = limiter
	}

	return limiter
}
func (r *rateLimiter) CleanUp() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			r.mu.Lock()
			r.limiters = make(map[string]*rate.Limiter)
			r.mu.Unlock()
		}
	}()
}
