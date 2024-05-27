package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
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

// delSerialize deletes the cached serialized data.
func (h *handler) delSerialize(ctx context.Context, key string) error {
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

// FileSet sets the file info into the cache.
func (h *handler) FileSet(ctx context.Context, f *file.File) error {
	key := fmt.Sprintf("storage:file:%s", f.ID)

	if err := h.setSerialize(ctx, key, f); err != nil {
		return err
	}

	return nil
}

// FileGet returns cached file info
func (h *handler) FileGet(ctx context.Context, id uuid.UUID) (*file.File, error) {
	key := fmt.Sprintf("storage:file:%s", id)

	var res file.File
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FileDel delete the cached file info
func (h *handler) FileDel(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("storage:file:%s", id)

	if err := h.delSerialize(ctx, key); err != nil {
		return err
	}

	return nil
}

// AccountSet sets the account info into the cache.
func (h *handler) AccountSet(ctx context.Context, f *account.Account) error {
	key := fmt.Sprintf("storage:account:%s", f.ID)

	if err := h.setSerialize(ctx, key, f); err != nil {
		return err
	}

	return nil
}

// AccountGet returns cached account info
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	key := fmt.Sprintf("storage:account:%s", id)

	var res account.Account
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
