package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides IP-based rate limiting for API endpoints.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	rps     rate.Limit
	burst   int
}

// NewRateLimiter creates a rate limiter with the given requests per second and burst size.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*client),
		rps:     rate.Limit(rps),
		burst:   burst,
	}

	// Clean up stale entries every minute
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) getClient(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if c, exists := rl.clients[ip]; exists {
		c.lastSeen = time.Now()
		return c.limiter
	}

	limiter := rate.NewLimiter(rl.rps, rl.burst)
	rl.clients[ip] = &client{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > 3*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getClient(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
