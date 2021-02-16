package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
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
	return h.setSerializeWithExpiration(ctx, key, data, time.Hour*24)
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerializeWithExpiration(ctx context.Context, key string, data interface{}, expiration time.Duration) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, expiration).Err(); err != nil {
		return err
	}
	return nil
}

// delKey deletes the given key from the cache.
func (h *handler) delKey(ctx context.Context, key string) error {
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

// AstEndpointGet returns cached AstEndpoint info
func (h *handler) AstEndpointGet(ctx context.Context, id string) (*models.AstEndpoint, error) {
	key := fmt.Sprintf("ast_endpoint:%s", id)

	var res models.AstEndpoint
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstEndpointSet sets the models.AstEndpoint info into the cache.
func (h *handler) AstEndpointSet(ctx context.Context, e *models.AstEndpoint) error {
	key := fmt.Sprintf("ast_endpoint:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstEndpointDel deletes the models.AstEndpoint info from the cache.
func (h *handler) AstEndpointDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_endpoint:%s", id)

	return h.delKey(ctx, key)
}

// AstAuthGet returns cached AstAuth info
func (h *handler) AstAuthGet(ctx context.Context, id string) (*models.AstAuth, error) {
	key := fmt.Sprintf("ast_auth:%s", id)

	var res models.AstAuth
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstAuthSet sets the models.AstAuth info into the cache.
func (h *handler) AstAuthSet(ctx context.Context, e *models.AstAuth) error {
	key := fmt.Sprintf("ast_auth:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstAuthDel deletes the models.AstAuth info from the cache.
func (h *handler) AstAuthDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_auth:%s", id)

	return h.delKey(ctx, key)
}

// AstAORGet returns cached AstAOR info
func (h *handler) AstAORGet(ctx context.Context, id string) (*models.AstAOR, error) {
	key := fmt.Sprintf("ast_aor:%s", id)

	var res models.AstAOR
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AstAORSet sets the models.AstAOR info into the cache.
func (h *handler) AstAORSet(ctx context.Context, e *models.AstAOR) error {
	key := fmt.Sprintf("ast_aor:%s", *e.ID)

	if err := h.setSerializeWithExpiration(ctx, key, e, time.Minute*3); err != nil {
		return err
	}

	return nil
}

// AstAORDel deletes the models.AstAOR info from the cache.
func (h *handler) AstAORDel(ctx context.Context, id string) error {
	key := fmt.Sprintf("ast_aor:%s", id)

	return h.delKey(ctx, key)
}

// DomainGet returns cached Domain info
func (h *handler) DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error) {
	key := fmt.Sprintf("domain:%s", id)

	var res models.Domain
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// DomainSet sets the domain info into the cache.
func (h *handler) DomainSet(ctx context.Context, e *models.Domain) error {
	key := fmt.Sprintf("domain:%s", e.ID)

	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	return nil
}

// DomainDel deletes the domain info from the cache.
func (h *handler) DomainDel(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("domain:%s", id)

	return h.delKey(ctx, key)
}

// ExtensionGet returns cached Domain info
func (h *handler) ExtensionGet(ctx context.Context, id uuid.UUID) (*models.Extension, error) {
	key := fmt.Sprintf("extension:%s", id)

	var res models.Extension
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ExtensionSet sets the extension info into the cache.
func (h *handler) ExtensionSet(ctx context.Context, e *models.Extension) error {
	key := fmt.Sprintf("extension:%s", e.ID)

	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	return nil
}

// ExtensionDel deletes the domain info from the cache.
func (h *handler) ExtensionDel(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("extension:%s", id)

	return h.delKey(ctx, key)
}

// AstContactsGet returns cached contacts info of the given endpoint
func (h *handler) AstContactsGet(ctx context.Context, endpoint string) ([]*models.AstContact, error) {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	var tmp []models.AstContact
	if err := h.getSerialize(ctx, key, &tmp); err != nil {
		return nil, err
	}

	var res []*models.AstContact
	for _, c := range tmp {
		res = append(res, &c)
	}

	return res, nil
}

// AstContactsSet sets the contacts info into the cache.
func (h *handler) AstContactsSet(ctx context.Context, endpoint string, contacts []*models.AstContact) error {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	if err := h.setSerialize(ctx, key, contacts); err != nil {
		return err
	}

	return nil
}

// AstContactsDel deletes the contacts info from the cache.
func (h *handler) AstContactsDel(ctx context.Context, endpoint string) error {
	key := fmt.Sprintf("ast_contacts:%s", endpoint)

	return h.delKey(ctx, key)
}
