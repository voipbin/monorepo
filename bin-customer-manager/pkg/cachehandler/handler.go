package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
)

const emailVerifyKeyPrefix = "email_verify:"
const verifyLockKeyPrefix = "verify_lock:"
const resendCooldownKeyPrefix = "email_verify_resend_cd:"
const resendCountKeyPrefix = "email_verify_resend_n:"

// getSerialize returns cached serialized info.
func (h *handler) getSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(tmp), &data); err != nil {
		return err
	}
	return nil
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, time.Hour*24).Err(); err != nil {
		return err
	}
	return nil
}

// CustomerGet returns cached customer info
func (h *handler) CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	key := fmt.Sprintf("customer:%s", id)

	var res customer.Customer
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CustomerSet sets the customer info into the cache.
func (h *handler) CustomerSet(ctx context.Context, c *customer.Customer) error {
	key := fmt.Sprintf("customer:%s", c.ID)

	if err := h.setSerialize(ctx, key, c); err != nil {
		return err
	}

	return nil
}

// AccesskeyGet returns cached accesskey info
func (h *handler) AccesskeyGet(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	key := fmt.Sprintf("customer_accesskey:%s", id)

	var res accesskey.Accesskey
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AccesskeySet sets the accesskey info into the cache.
func (h *handler) AccesskeySet(ctx context.Context, a *accesskey.Accesskey) error {
	key := fmt.Sprintf("customer_accesskey:%s", a.ID)

	if err := h.setSerialize(ctx, key, a); err != nil {
		return err
	}

	return nil
}

// EmailVerifyTokenSet stores an email verification token in Redis with a TTL.
func (h *handler) EmailVerifyTokenSet(ctx context.Context, token string, customerID uuid.UUID, ttl time.Duration) error {
	key := emailVerifyKeyPrefix + token
	if err := h.Cache.Set(ctx, key, customerID.String(), ttl).Err(); err != nil {
		return err
	}
	return nil
}

// EmailVerifyTokenGet retrieves the customer ID associated with an email verification token.
func (h *handler) EmailVerifyTokenGet(ctx context.Context, token string) (uuid.UUID, error) {
	key := emailVerifyKeyPrefix + token
	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, fmt.Errorf("token not found or expired")
		}
		return uuid.Nil, err
	}

	id, err := uuid.FromString(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse customer id from token: %v", err)
	}
	return id, nil
}

// EmailVerifyTokenDelete removes an email verification token from Redis.
func (h *handler) EmailVerifyTokenDelete(ctx context.Context, token string) error {
	key := emailVerifyKeyPrefix + token
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

// VerifyLockAcquire attempts to acquire a distributed lock for customer verification.
// Returns true if the lock was acquired, false if another process holds it.
func (h *handler) VerifyLockAcquire(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (bool, error) {
	key := verifyLockKeyPrefix + customerID.String()
	ok, err := h.Cache.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// VerifyLockRelease releases the distributed verification lock.
func (h *handler) VerifyLockRelease(ctx context.Context, customerID uuid.UUID) error {
	key := verifyLockKeyPrefix + customerID.String()
	return h.Cache.Del(ctx, key).Err()
}

// ResendCooldownAcquire returns true when a verification resend may be sent for
// this customer right now, and arms the cooldown. It returns false while a
// previous send is still inside the cooldown window.
func (h *handler) ResendCooldownAcquire(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (bool, error) {
	key := resendCooldownKeyPrefix + customerID.String()

	ok, err := h.Cache.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}

	return ok, nil
}

// ResendCountIncr increments this customer's rolling resend counter and returns
// the new value.
//
// The TTL repair below is not defensive noise. INCR creates the key without an
// expiry, so if the follow-up EXPIRE fails or the process dies between the two
// commands, the counter would live forever and permanently bar that customer
// from the one recovery path this endpoint exists to provide. Re-arming a
// missing TTL on every call makes that failure self-healing.
func (h *handler) ResendCountIncr(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (int64, error) {
	key := resendCountKeyPrefix + customerID.String()

	n, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if n == 1 {
		if errExpire := h.Cache.Expire(ctx, key, ttl).Err(); errExpire != nil {
			return n, errExpire
		}
		return n, nil
	}

	remain, err := h.Cache.TTL(ctx, key).Result()
	if err != nil {
		return n, err
	}
	// go-redis returns a negative duration when the key has no expiry (-1) or is
	// already gone (-2).
	if remain < 0 {
		if errExpire := h.Cache.Expire(ctx, key, ttl).Err(); errExpire != nil {
			return n, errExpire
		}
	}

	return n, nil
}

// ResendCountDecr gives back one unit of this customer's rolling resend budget.
//
// It exists so an email-manager outage does not burn the daily cap without a
// single delivered mail, which would lock the customer out of the exact recovery
// path this endpoint provides. A single atomic DECR is enough: the key already
// carries the TTL armed by ResendCountIncr, and DECR does not clear it.
func (h *handler) ResendCountDecr(ctx context.Context, customerID uuid.UUID) error {
	key := resendCountKeyPrefix + customerID.String()

	return h.Cache.Decr(ctx, key).Err()
}
