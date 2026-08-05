package middleware

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/ratelimithandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Tier names for the Redis customer-level rate limiter (VOIP-1302 §4-3,
// §4-10). These are used verbatim as the middle segment of the Redis key
// (ratelimit:{tier}:{customer_id}), as the Prometheus "tier" label
// (reusing api_manager_rate_limit_{allowed,rejected}_total), and in config
// field/env var names.
const (
	TierV1Customer         = "v1_customer"
	TierV1CustomerDirect   = "v1_customer_direct"
	TierV1CustomerDelegate = "v1_customer_delegate"
)

// CustomerRateLimitConfig holds the per-tier GCRA parameters and the Redis
// call timeout for CustomerRateLimit. See VOIP-1302 design doc §4-9 for the
// initial values and rationale.
type CustomerRateLimitConfig struct {
	CustomerRPS   float64 // TypeAgent and TypeAccesskey share this bucket (tier v1_customer).
	CustomerBurst int

	DirectRPS   float64 // TypeDirect (tier v1_customer_direct).
	DirectBurst int

	DelegateRPS   float64 // TypeDelegate (tier v1_customer_delegate).
	DelegateBurst int

	RedisTimeout time.Duration // Budget for the Redis round trip; on timeout/error the request fails open.
}

// customerLogSampler rate-limits the WARN-level logs emitted for skipped or
// rejected customer-tier requests to at most once per minute per
// customer_id, so a sustained abuse burst does not spam the log pipeline.
// It reuses the idle-eviction pattern from ratelimitStore.cleanup() in
// ratelimit.go: entries idle for 10 minutes are evicted every 5 minutes.
// Per VOIP-1302 §4-11, this is per-pod, so a fleet of N pods can log up to
// N times per customer per minute -- aggregation downstream is expected to
// collapse that.
type customerLogSampler struct {
	mu         sync.Mutex
	lastLogged map[uuid.UUID]time.Time
}

func newCustomerLogSampler() *customerLogSampler {
	s := &customerLogSampler{
		lastLogged: make(map[uuid.UUID]time.Time),
	}

	go s.cleanup()

	return s
}

// shouldLog reports whether the caller should emit a log line for id now,
// and if so, records the current time as the last-logged time.
func (s *customerLogSampler) shouldLog(id uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	last, exists := s.lastLogged[id]
	if exists && now.Sub(last) < time.Minute {
		return false
	}

	s.lastLogged[id] = now
	return true
}

func (s *customerLogSampler) cleanup() {
	for {
		time.Sleep(5 * time.Minute)

		s.mu.Lock()
		for id, t := range s.lastLogged {
			if time.Since(t) > 10*time.Minute {
				delete(s.lastLogged, id)
			}
		}
		s.mu.Unlock()
	}
}

// tierForIdentity maps an auth identity to its rate limit tier and
// configured (rps, burst). It returns tier == "" for identity types that do
// not carry a customer-scoped tier (there are currently none -- all four
// auth.Type values map to a tier -- but this keeps the switch exhaustive
// and safe against a future identity type being added without an explicit
// tier decision).
func tierForIdentity(identity *auth.AuthIdentity, cfg CustomerRateLimitConfig) (tier string, rps float64, burst int) {
	switch identity.Type {
	case auth.TypeAgent, auth.TypeAccesskey:
		// Agent and accesskey identities share a bucket keyed only by
		// customer_id (no identity_type segment) -- see VOIP-1302 §4-3:
		// they are the same API client resource authenticated two
		// different ways, and splitting them would let a customer double
		// their effective quota by using both credentials at once.
		return TierV1Customer, cfg.CustomerRPS, cfg.CustomerBurst
	case auth.TypeDirect:
		return TierV1CustomerDirect, cfg.DirectRPS, cfg.DirectBurst
	case auth.TypeDelegate:
		return TierV1CustomerDelegate, cfg.DelegateRPS, cfg.DelegateBurst
	default:
		return "", 0, 0
	}
}

// CustomerRateLimit returns a Gin middleware enforcing a Redis-backed
// global (cross-pod) rate limit keyed by customer_id, per VOIP-1302. It
// must run after Authenticate() (which populates "auth_identity" in the gin
// context).
//
// Behavior:
//   - If no auth identity is present in context, the request proceeds
//     unaffected (this middleware only enforces customer-scoped limits; IP
//     tiers cover the unauthenticated surface).
//   - If the identity's CustomerID is uuid.Nil, the request proceeds (nil
//     guard, §4-4) and a sampled WARN log is emitted -- this should not
//     happen in practice since every identity constructor now populates
//     CustomerID from an upstream value, but this middleware is the final
//     safety net.
//   - If the resolved tier's rps is not positive or burst <= 0, the tier is
//     disabled (unlimited pass-through) -- same "set to 0 to disable"
//     convention as the IP-based RateLimit middleware.
//   - On a Redis error or timeout, the request fails open (proceeds) --
//     this limiter is an abuse-protection layer, not a billing/entitlement
//     enforcement mechanism (§4-7).
//   - On rejection, a 429 RESOURCE_EXHAUSTED envelope is returned with a
//     Retry-After header and Details: [{"limited_by": "customer"}].
func CustomerRateLimit(limiter ratelimithandler.RateLimiter, cfg CustomerRateLimitConfig) gin.HandlerFunc {
	sampler := newCustomerLogSampler()

	return func(c *gin.Context) {
		v, exists := c.Get("auth_identity")
		if !exists {
			c.Next()
			return
		}

		identity, ok := v.(*auth.AuthIdentity)
		if !ok || identity == nil {
			c.Next()
			return
		}

		if identity.CustomerID == uuid.Nil {
			if sampler.shouldLog(uuid.Nil) {
				logrus.WithFields(logrus.Fields{
					"func": "CustomerRateLimit",
					"type": identity.Type,
				}).Warn("Skipping customer rate limit: identity has nil CustomerID.")
			}
			c.Next()
			return
		}

		tier, rps, burst := tierForIdentity(identity, cfg)
		if tier == "" {
			c.Next()
			return
		}

		if !(rps > 0) || burst <= 0 {
			// Tier disabled -- safe rollback lever, see ratelimit.go's
			// RateLimit doc comment for the same convention.
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", tier, identity.CustomerID.String())

		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.RedisTimeout)
		defer cancel()

		limit := ratelimithandler.Limit{
			Rate:   int(math.Round(rps)),
			Burst:  burst,
			Period: time.Second,
		}

		result, err := limiter.Allow(ctx, key, limit)
		if err != nil {
			// Fail-open: abuse protection only, never blocks on Redis
			// unavailability (§4-7).
			if sampler.shouldLog(identity.CustomerID) {
				logrus.WithFields(logrus.Fields{
					"func":        "CustomerRateLimit",
					"tier":        tier,
					"customer_id": identity.CustomerID,
				}).Warnf("Customer rate limiter Redis call failed; failing open. err: %v", err)
			}
			c.Next()
			return
		}

		allowed := promRateLimitAllowedTotal.WithLabelValues(tier)
		rejected := promRateLimitRejectedTotal.WithLabelValues(tier)

		if result.Allowed < 1 {
			rejected.Inc()

			if sampler.shouldLog(identity.CustomerID) {
				logrus.WithFields(logrus.Fields{
					"func":        "CustomerRateLimit",
					"tier":        tier,
					"customer_id": identity.CustomerID,
					"remaining":   result.Remaining,
					"retry_after": result.RetryAfter,
				}).Warn("Customer rate limit exceeded.")
			}

			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(result.RetryAfter.Seconds()))))

			e := cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.")
			e.Details = []map[string]any{{"limited_by": "customer"}}
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
