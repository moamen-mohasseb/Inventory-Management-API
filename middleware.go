package main

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// clientBucket is the token bucket that belongs to a single client (IP address).
type clientBucket struct {
	tokens   float64   // tokens currently available (a request costs 1 token)
	lastSeen time.Time // last time this bucket was refilled / used
}

// RateLimiter implements the Token Bucket algorithm with one bucket per client.
//   - rate  : how many tokens are added per second (sustained requests/second)
//   - burst : the bucket capacity (how many requests may arrive at once)
type RateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientBucket
	rate        float64
	burst       float64
	idleTTL     time.Duration
	lastCleanup time.Time
}

// NewRateLimiter creates a limiter allowing ratePerSecond requests per second
// with a burst capacity of burst requests.
func NewRateLimiter(ratePerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		clients:     make(map[string]*clientBucket),
		rate:        ratePerSecond,
		burst:       float64(burst),
		idleTTL:     10 * time.Minute,
		lastCleanup: time.Now(),
	}
}

// Allow reports whether the client identified by key may make a request now.
// When it may not, it also returns how long the client should wait.
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	return rl.allowAt(key, time.Now())
}

// allowAt is Allow with an explicit clock, which keeps the logic easy to test.
func (rl *RateLimiter) allowAt(key string, now time.Time) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanup(now)

	b, exists := rl.clients[key]
	if !exists {
		// A new client starts with a full bucket.
		b = &clientBucket{tokens: rl.burst, lastSeen: now}
		rl.clients[key] = b
	} else {
		// Refill according to the time that has passed, never above the capacity.
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens = math.Min(rl.burst, b.tokens+elapsed*rl.rate)
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	wait := time.Duration((1 - b.tokens) / rl.rate * float64(time.Second))
	return false, wait
}

// cleanup removes buckets of clients that have been idle for a long time so the
// map does not grow forever. It must be called with rl.mu held.
func (rl *RateLimiter) cleanup(now time.Time) {
	if now.Sub(rl.lastCleanup) < time.Minute {
		return
	}
	for key, b := range rl.clients {
		if now.Sub(b.lastSeen) > rl.idleTTL {
			delete(rl.clients, key)
		}
	}
	rl.lastCleanup = now
}

// RateLimitMiddleware returns a Gin middleware that rejects requests with
// 429 Too Many Requests when the client has used up its tokens.
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, wait := rl.Allow(c.ClientIP())
		if !allowed {
			seconds := int(math.Ceil(wait.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
				Error: "rate limit exceeded, please retry later",
			})
			return
		}
		c.Next()
	}
}
