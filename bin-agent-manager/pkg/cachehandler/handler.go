package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/address"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
)

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

// ResourceGet retrieves the cached resource information associated with the given ID.
// It uses the ID as the cache key.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// id (uuid.UUID): The ID of the resource for which to retrieve the information.
//
// Returns:
// (*resource.Resource, error): A pointer to the retrieved resource information and any error encountered during the operation.
// If the resource information is not found in the cache, nil is returned for the resource and an error is returned.
func (h *handler) ResourceGet(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	key := fmt.Sprintf("agent:resource:%d", id)

	var res resource.Resource
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ResourceSet sets the resource info into the cache.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// u (resource.Resource): The resource information to be set in the cache.
//
// Returns:
// error: An error if the operation fails, nil otherwise.
//
// Note: The resource information is stored in the cache using the format "agent:resource:<id>".
// The resource ID is used as the unique identifier for each resource.
func (h *handler) ResourceSet(ctx context.Context, u *resource.Resource) error {
	key := fmt.Sprintf("agent:resource:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// AddressGetByCommonAddress retrieves the cached address information associated with the given common address.
// It uses the common address type and target as the cache key.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// addr (commonaddress.Address): The common address for which to retrieve the address information.
//
// Returns:
// (*address.Address, error): A pointer to the retrieved address information and any error encountered during the operation.
// If the address information is not found in the cache, nil is returned for the address and an error is returned.

func (h *handler) AddressGetByCommonAddress(ctx context.Context, u commonaddress.Address) (*address.Address, error) {
	key := fmt.Sprintf("agent:address:%s-%s", u.Type, u.Target)

	var res address.Address
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AddressSet sets the address info into the cache.
// It uses the common address type and target as the cache key.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// u (address.Address): The address information to be set in the cache.
//
// Returns:
// error: An error if the operation fails, nil otherwise.
//
// Note: The address information is stored in the cache using the format "agent:address:<type>-<target>".
func (h *handler) AddressSet(ctx context.Context, u *address.Address) error {
	key := fmt.Sprintf("agent:address:%s-%s", u.Address.Type, u.Address.Target)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// AddressDel deletes the address info from the cache.
// It uses the common address type and target as the cache key.
//
// Parameters:
// ctx (context.Context): The context for the operation.
// u (address.Address): The address information to be deleted from the cache.
//
// Returns:
// error: An error if the operation fails, nil otherwise.
//
// Note: The address information is deleted from the cache using the format "agent:address:<type>-<target>".
func (h *handler) AddressDel(ctx context.Context, u *address.Address) error {
	key := fmt.Sprintf("agent:address:%s-%s", u.Address.Type, u.Address.Target)

	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}
