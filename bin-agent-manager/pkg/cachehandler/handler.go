package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
)

const passwordResetKeyPrefix = "password_reset:"

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

// AgentGet retrieves the cached agent information associated with the given ID.
// It uses the ID as the cache key.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// id (uuid.UUID): The ID of the agent for which to retrieve the information.
//
// Returns:
// (*agent.Agent, error): A pointer to the retrieved agent information and any error encountered during the operation.
// If the agent information is not found in the cache, nil is returned for the agent and an error is returned.
func (h *handler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	key := fmt.Sprintf("agent:agent:%d", id)

	var res agent.Agent
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentSet sets the agent info into the cache.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// u (agent.Agent): The agent information to be set in the cache.
//
// Returns:
// error: An error if the operation fails, nil otherwise.
//
// Note: The agent information is stored in the cache using the format "agent:agent:<id>".
// The agent ID is used as the unique identifier for each agent.
func (h *handler) AgentSet(ctx context.Context, u *agent.Agent) error {
	key := fmt.Sprintf("agent:agent:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// PasswordResetTokenSet stores a password reset token in Redis with a TTL.
func (h *handler) PasswordResetTokenSet(ctx context.Context, token string, agentID uuid.UUID, ttl time.Duration) error {
	key := passwordResetKeyPrefix + token
	if err := h.Cache.Set(ctx, key, agentID.String(), ttl).Err(); err != nil {
		return err
	}
	return nil
}

// PasswordResetTokenGet retrieves the agent ID associated with a password reset token.
func (h *handler) PasswordResetTokenGet(ctx context.Context, token string) (uuid.UUID, error) {
	key := passwordResetKeyPrefix + token
	val, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.FromString(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse agent id from token: %v", err)
	}
	return id, nil
}

// PasswordResetTokenDelete removes a password reset token from Redis.
func (h *handler) PasswordResetTokenDelete(ctx context.Context, token string) error {
	key := passwordResetKeyPrefix + token
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}
