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
