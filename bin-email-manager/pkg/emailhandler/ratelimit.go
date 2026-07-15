package emailhandler

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-email-manager/internal/config"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// rateLimitWindowMinute and rateLimitWindowHour are the fixed-window TTLs for
// the outbound email rate limit counters. VOIP-1259.
const (
	rateLimitWindowMinute = time.Minute
	rateLimitWindowHour   = time.Hour
)

// validateCustomerEmailRate returns true if the customer has not exceeded the
// outbound email rate limit (per-minute and per-hour). VOIP-1259.
//
// Implementation: Redis-backed fixed-window counters keyed
// "email-manager:ratelimit:email:<customerID>:minute" and "...:hour". Each
// request unconditionally performs INCR + ExpireNX (NOT gated on count==1 —
// see design doc VOIP-1259 §6-A for why a count==1 guard leaves a
// permanent-lockout gap if the process crashes between INCR and EXPIRE).
// Either window breaching its cap fails closed (false). Any Redis error also
// fails closed (false), consistent with the existing balance-check's
// fail-closed convention.
func (h *emailHandler) validateCustomerEmailRate(ctx context.Context, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "validateCustomerEmailRate",
		"customer_id": customerID,
	})

	cfg := config.Get()

	minuteKey := fmt.Sprintf("email-manager:ratelimit:email:%s:minute", customerID)
	minuteCount, err := h.cache.RateLimitIncrement(ctx, minuteKey, rateLimitWindowMinute)
	if err != nil {
		log.Errorf("Could not increment the minute rate limit counter, failing closed. err: %v", err)
		promOutboundRateLimitedTotal.WithLabelValues("email", "rejected").Inc()
		return false
	}

	hourKey := fmt.Sprintf("email-manager:ratelimit:email:%s:hour", customerID)
	hourCount, err := h.cache.RateLimitIncrement(ctx, hourKey, rateLimitWindowHour)
	if err != nil {
		log.Errorf("Could not increment the hour rate limit counter, failing closed. err: %v", err)
		promOutboundRateLimitedTotal.WithLabelValues("email", "rejected").Inc()
		return false
	}

	if minuteCount > int64(cfg.EmailOutboundRateLimitPerMinute) {
		log.Infof("Customer exceeded the per-minute outbound email rate limit. count: %d, limit: %d", minuteCount, cfg.EmailOutboundRateLimitPerMinute)
		promOutboundRateLimitedTotal.WithLabelValues("email", "rejected").Inc()
		return false
	}

	if hourCount > int64(cfg.EmailOutboundRateLimitPerHour) {
		log.Infof("Customer exceeded the per-hour outbound email rate limit. count: %d, limit: %d", hourCount, cfg.EmailOutboundRateLimitPerHour)
		promOutboundRateLimitedTotal.WithLabelValues("email", "rejected").Inc()
		return false
	}

	return true
}
