package middleware

import (
	"sync"
	"time"

	"monorepo/bin-api-manager/lib/apierror"
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
			// Build the canonical external envelope. The internal Domain
			// field is omitted by lib/apierror — see envelope.go.
			e := cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.")
			c.AbortWithStatusJSON(
				cerrors.HTTPStatusFor(e.Status),
				apierror.EnvelopeFor(e, RequestIDFromContext(c)),
			)
			return
		}

		c.Next()
	}
}
