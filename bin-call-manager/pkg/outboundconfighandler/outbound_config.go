package outboundconfighandler

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// Delete soft-deletes the OutboundConfig identified by id and invalidates its cache entry.
func (h *outboundConfigHandler) Delete(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	// Fetch first so we have the customer_id for cache invalidation.
	c, err := h.db.OutboundConfigGetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get outbound_config for delete: %w", err)
	}

	if err := h.db.OutboundConfigDelete(ctx, id); err != nil {
		return nil, fmt.Errorf("could not delete outbound_config: %w", err)
	}

	if c != nil {
		_ = h.cacheHandler.OutboundConfigDelete(ctx, c.CustomerID)
	}

	return c, nil
}

// GetByCustomerID returns the OutboundConfig for the given customerID.
// It uses a cache-aside pattern with a negative-cache sentinel for missing rows.
func (h *outboundConfigHandler) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("customer_id", customerID)

	// Cache lookup
	cached, err := h.cacheHandler.OutboundConfigGet(ctx, customerID)
	if err == nil {
		// cache hit (real row or negative sentinel)
		return cached, nil
	}
	if err != redis.Nil {
		log.Warnf("Cache error getting outbound_config, falling through to DB. err: %v", err)
	}

	// DB lookup
	c, err := h.db.OutboundConfigGetByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("db error getting outbound_config: %w", err)
	}
	if c == nil {
		_ = h.cacheHandler.OutboundConfigSetNotFound(ctx, customerID)
		return nil, nil
	}
	_ = h.cacheHandler.OutboundConfigSet(ctx, customerID, c)
	return c, nil
}

// GetByID returns the OutboundConfig with the given id directly from DB.
func (h *outboundConfigHandler) GetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	return h.db.OutboundConfigGetByID(ctx, id)
}

// List returns a page of OutboundConfigs for the given customerID.
func (h *outboundConfigHandler) List(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error) {
	return h.db.OutboundConfigList(ctx, customerID, pageSize, pageToken)
}

// Create validates the request, creates a new OutboundConfig, and writes it through the cache.
func (h *outboundConfigHandler) Create(ctx context.Context, customerID uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
	if err := h.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	c := &outboundconfig.OutboundConfig{
		ID:         h.utilHandler.UUIDCreate(),
		CustomerID: customerID,
	}
	applyUpdateRequest(c, req)

	if err := h.db.OutboundConfigCreate(ctx, c); err != nil {
		return nil, fmt.Errorf("could not create outbound_config: %w", err)
	}
	_ = h.cacheHandler.OutboundConfigSet(ctx, customerID, c)
	return c, nil
}

// Update validates the request, updates the OutboundConfig in DB, and invalidates the cache.
func (h *outboundConfigHandler) Update(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
	if err := h.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	c, err := h.db.OutboundConfigUpdate(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("could not update outbound_config: %w", err)
	}
	if c != nil {
		_ = h.cacheHandler.OutboundConfigDelete(ctx, c.CustomerID)
	}
	return c, nil
}

// applyUpdateRequest applies non-nil fields from req onto c.
func applyUpdateRequest(c *outboundconfig.OutboundConfig, req *outboundconfig.UpdateRequest) {
	if req == nil {
		return
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Detail != nil {
		c.Detail = *req.Detail
	}
	if req.DestinationWhitelist != nil {
		c.DestinationWhitelist = *req.DestinationWhitelist
	}
	if req.Codecs != nil {
		c.Codecs = *req.Codecs
	}
}
