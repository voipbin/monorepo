package middleware

import (
	"math"
	"strconv"
	"sync"
	"time"

	"monorepo/bin-api-manager/lib/apierror"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var (
	metricsNamespace = "api_manager"

	promRateLimitAllowedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "rate_limit_allowed_total",
			Help:      "Total number of requests allowed by the rate limiter, by tier",
		},
		[]string{"tier"},
	)

	promRateLimitRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "rate_limit_rejected_total",
			Help:      "Total number of requests rejected by the rate limiter, by tier",
		},
		[]string{"tier"},
	)
)

func init() {
	prometheus.MustRegister(
		promRateLimitAllowedTotal,
		promRateLimitRejectedTotal,
	)
}

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

// RateLimit returns a Gin middleware that limits requests per IP address,
// tracked separately per tier for observability (metrics below).
//
// r is the rate in requests per second, burst is the maximum burst size.
//
// Disable semantics: if r is not a positive number (covers <= 0, -Inf, and
// NaN) or burst <= 0, the tier is treated as disabled -- no per-IP store or
// cleanup goroutine is created, and the returned middleware unconditionally
// calls c.Next(). This makes "set to 0 to disable" a safe rollback lever:
// the naive alternative of feeding r=0 straight into rate.NewLimiter would
// deny every request instead of allowing them, turning an intended
// incident-time disable into a total outage for the tier.
//
// +Inf is normalized to rate.Inf (the library's actual "unlimited" sentinel,
// golang.org/x/time/rate.Inf = Limit(math.MaxFloat64), matched by an exact
// == comparison internally) rather than passed through as IEEE +Inf, which
// does not equal that sentinel and would fall into unspecified floating-point
// behavior in the limiter's internal token accounting instead of the
// library's intended fast path.
func RateLimit(tier string, r float64, burst int) gin.HandlerFunc {
	allowed := promRateLimitAllowedTotal.WithLabelValues(tier)
	rejected := promRateLimitRejectedTotal.WithLabelValues(tier)

	if !(r > 0) || burst <= 0 {
		logrus.WithFields(logrus.Fields{
			"func":  "RateLimit",
			"tier":  tier,
			"rate":  r,
			"burst": burst,
		}).Warnf("Rate limiting is disabled for this tier.")

		return func(c *gin.Context) {
			c.Next()
		}
	}

	if math.IsInf(r, 1) {
		r = float64(rate.Inf)
	}

	store := newRateLimitStore(rate.Limit(r), burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			rejected.Inc()

			// Compute Retry-After without consuming additional budget.
			// Reserve() books a token immediately (as if the request had
			// been allowed); Cancel() undoes that booking right away. If
			// Cancel() were omitted here, every rejected request would
			// leave the bucket with one fewer token than it should have,
			// so a burst of rejections would starve subsequent legitimate
			// requests (VOIP-1302 §4-8).
			reservation := limiter.Reserve()
			delay := reservation.Delay()
			reservation.Cancel()
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))

			// Build the canonical external envelope. The internal Domain
			// field is omitted by lib/apierror -- see envelope.go.
			e := cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.")
			e.Details = []map[string]any{{"limited_by": "ip"}}
			c.AbortWithStatusJSON(
				cerrors.HTTPStatusFor(e.Status),
				apierror.EnvelopeFor(e, RequestIDFromContext(c)),
			)
			return
		}

		allowed.Inc()
		c.Next()
	}
}
