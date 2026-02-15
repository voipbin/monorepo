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
const signupSessionKeyPrefix = "signup_session:"
const signupAttemptsKeyPrefix = "signup_attempts:"
const verifyLockKeyPrefix = "verify_lock:"

// SignupSession stores the headless signup session data in Redis.
type SignupSession struct {
	CustomerID  uuid.UUID `json:"customer_id"`
	OTPCode     string    `json:"otp_code"`
	VerifyToken string    `json:"verify_token"`
}

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

// SignupSessionSet stores a signup session in Redis with a TTL.
func (h *handler) SignupSessionSet(ctx context.Context, tempToken string, session *SignupSession, ttl time.Duration) error {
	key := signupSessionKeyPrefix + tempToken
	tmp, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err := h.Cache.Set(ctx, key, tmp, ttl).Err(); err != nil {
		return err
	}
	return nil
}

// SignupSessionGet retrieves a signup session from Redis.
func (h *handler) SignupSessionGet(ctx context.Context, tempToken string) (*SignupSession, error) {
	key := signupSessionKeyPrefix + tempToken
	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("signup session not found or expired")
		}
		return nil, err
	}

	var session SignupSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// SignupSessionDelete removes a signup session from Redis.
func (h *handler) SignupSessionDelete(ctx context.Context, tempToken string) error {
	key := signupSessionKeyPrefix + tempToken
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

// SignupAttemptIncrement increments the signup attempt counter for the given tempToken.
// On the first increment (count == 1), the TTL is set. Returns the current count.
func (h *handler) SignupAttemptIncrement(ctx context.Context, tempToken string, ttl time.Duration) (int64, error) {
	key := signupAttemptsKeyPrefix + tempToken
	count, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		if err := h.Cache.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, err
		}
	}
	return count, nil
}

// SignupAttemptDelete removes the signup attempt counter from Redis.
func (h *handler) SignupAttemptDelete(ctx context.Context, tempToken string) error {
	key := signupAttemptsKeyPrefix + tempToken
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
