package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// provisioningTokenKeyPrefix is the Redis key prefix for extension
// provisioning tokens (VOIP-1391). Value: extension UUID string.
const provisioningTokenKeyPrefix = "api-manager.provisioning_token."

// getSerialize returns cached serialized info.
//nolint: unused // reserved
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
//nolint: unused // reserved
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

// ProvisioningTokenSet stores an extension provisioning token in Redis with a TTL.
func (h *handler) ProvisioningTokenSet(ctx context.Context, token string, extensionID uuid.UUID, ttl time.Duration) error {
	key := provisioningTokenKeyPrefix + token
	if err := h.Cache.Set(ctx, key, extensionID.String(), ttl).Err(); err != nil {
		return err
	}
	return nil
}

// ProvisioningTokenGet retrieves the extension ID associated with a provisioning token.
func (h *handler) ProvisioningTokenGet(ctx context.Context, token string) (uuid.UUID, error) {
	key := provisioningTokenKeyPrefix + token
	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.FromString(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse extension id from token: %v", err)
	}
	return id, nil
}
