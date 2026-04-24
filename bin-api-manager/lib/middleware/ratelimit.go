package middleware

import (
	"net/http"
	"sync"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimitStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
}

func newRateLimitStore(r rate.Limit, burst int) *rateLimitStore {
	s := &rateLimitStore{
		limiters: make(map[string]*ipLimiter),
		rate:     r,
		burst:    burst,
	}

	go s.cleanup()

	return s
}

func (s *rateLimitStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, exists := s.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(s.rate, s.burst)
		s.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (s *rateLimitStore) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		for ip, v := range s.limiters {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(s.limiters, ip)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimit returns a Gin middleware that limits requests per IP address.
// r is the rate in requests per second, burst is the maximum burst size.
func RateLimit(r float64, burst int) gin.HandlerFunc {
	store := newRateLimitStore(rate.Limit(r), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			// Inline envelope construction — lib/middleware cannot import
			// the server package (would create an import cycle), so we
			// build the same shape as server.abortWithError here.
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"status":     string(cerrors.StatusResourceExhausted),
					"reason":     "RATE_LIMIT_EXCEEDED",
					"domain":     string(commonoutline.ServiceNameAPIManager),
					"message":    "Too many requests. Please try again later.",
					"request_id": RequestIDFromContext(c),
				},
			})
			return
		}

		c.Next()
	}
}
