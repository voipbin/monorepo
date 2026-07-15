package messagehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/internal/config"
)

// rateLimitWindowMinute and rateLimitWindowHour are the fixed-window TTLs for
// the outbound SMS rate limit counters. VOIP-1259.
const (
	rateLimitWindowMinute = time.Minute
	rateLimitWindowHour   = time.Hour
)

// validateCustomerMessageRate returns true if the customer has not exceeded
// the outbound SMS rate limit (per-minute and per-hour). VOIP-1259.
//
// Implementation: Redis-backed fixed-window counters keyed
// "message-manager:ratelimit:sms:<customerID>:minute" and "...:hour". Each
// request unconditionally performs INCR + ExpireNX (NOT gated on count==1 —
// see design doc VOIP-1259 §6-A for why a count==1 guard leaves a
// permanent-lockout gap if the process crashes between INCR and EXPIRE).
// Either window breaching its cap fails closed (false). Any Redis error also
// fails closed (false), consistent with the fail-closed convention used
// throughout this codebase (e.g. balance checks).
func (h *messageHandler) validateCustomerMessageRate(ctx context.Context, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "validateCustomerMessageRate",
		"customer_id": customerID,
	})

	cfg := config.Get()

	minuteKey := fmt.Sprintf("message-manager:ratelimit:sms:%s:minute", customerID)
	minuteCount, err := h.cache.RateLimitIncrement(ctx, minuteKey, rateLimitWindowMinute)
	if err != nil {
		log.Errorf("Could not increment the minute rate limit counter, failing closed. err: %v", err)
		promOutboundRateLimitedTotal.WithLabelValues("sms", "rejected").Inc()
		return false
	}

	hourKey := fmt.Sprintf("message-manager:ratelimit:sms:%s:hour", customerID)
	hourCount, err := h.cache.RateLimitIncrement(ctx, hourKey, rateLimitWindowHour)
	if err != nil {
		log.Errorf("Could not increment the hour rate limit counter, failing closed. err: %v", err)
		promOutboundRateLimitedTotal.WithLabelValues("sms", "rejected").Inc()
		return false
	}

	if minuteCount > int64(cfg.MessageOutboundRateLimitPerMinute) {
		log.Infof("Customer exceeded the per-minute outbound SMS rate limit. count: %d, limit: %d", minuteCount, cfg.MessageOutboundRateLimitPerMinute)
		promOutboundRateLimitedTotal.WithLabelValues("sms", "rejected").Inc()
		return false
	}

	if hourCount > int64(cfg.MessageOutboundRateLimitPerHour) {
		log.Infof("Customer exceeded the per-hour outbound SMS rate limit. count: %d, limit: %d", hourCount, cfg.MessageOutboundRateLimitPerHour)
		promOutboundRateLimitedTotal.WithLabelValues("sms", "rejected").Inc()
		return false
	}

	return true
}
