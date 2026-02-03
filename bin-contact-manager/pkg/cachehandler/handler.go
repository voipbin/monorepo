package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	// Cache key prefix for contacts
	contactKeyPrefix = "contact:"

	// Default cache TTL
	cacheTTL = time.Hour * 24
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

	if err := h.Cache.Set(ctx, key, tmp, cacheTTL).Err(); err != nil {
		return err
	}
	return nil
}

// delete removes a key from the cache.
func (h *handler) delete(ctx context.Context, key string) error {
	return h.Cache.Del(ctx, key).Err()
}

// ContactGet returns cached contact info
func (h *handler) ContactGet(ctx context.Context, id uuid.UUID) (*contact.Contact, error) {
	key := fmt.Sprintf("%s%s", contactKeyPrefix, id)

	var res contact.Contact
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ContactSet sets the contact info into the cache.
func (h *handler) ContactSet(ctx context.Context, c *contact.Contact) error {
	key := fmt.Sprintf("%s%s", contactKeyPrefix, c.ID)

	if err := h.setSerialize(ctx, key, c); err != nil {
		return err
	}

	return nil
}

// ContactDelete removes the contact from the cache.
func (h *handler) ContactDelete(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s%s", contactKeyPrefix, id)
	return h.delete(ctx, key)
}
