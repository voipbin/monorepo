package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
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

// delSerialize deletes cached serialized info.
func (h *handler) delSerialize(ctx context.Context, key string) error {
	_, err := h.Cache.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

// NumberGetByNumber returns number call info
func (h *handler) NumberGetByNumber(ctx context.Context, num string) (*number.Number, error) {
	key := fmt.Sprintf("number-number:%s", num)

	var res number.Number
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberSetByNumber sets the number info into the cache.
func (h *handler) NumberSetByNumber(ctx context.Context, numb *number.Number) error {
	key := fmt.Sprintf("number-number:%s", numb.Number)

	if err := h.setSerialize(ctx, key, numb); err != nil {
		return err
	}

	return nil
}

// NumberSetByNumber deletes the number info by the number from the cache.
func (h *handler) NumberDelByNumber(ctx context.Context, num string) error {
	key := fmt.Sprintf("number-number:%s", num)

	if err := h.delSerialize(ctx, key); err != nil {
		return err
	}

	return nil
}

// NumberGet returns number call info
func (h *handler) NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	key := fmt.Sprintf("number:%s", id)

	var res number.Number
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberSet sets the number info into the cache.
func (h *handler) NumberSet(ctx context.Context, numb *number.Number) error {
	key := fmt.Sprintf("number:%s", numb.ID)
	if err := h.setSerialize(ctx, key, numb); err != nil {
		return err
	}

	keyNumber := fmt.Sprintf("number-number:%s", numb.Number)
	if err := h.setSerialize(ctx, keyNumber, numb); err != nil {
		return err
	}

	return nil
}

// NumberDel deletes the number info from the cache.
func (h *handler) NumberDel(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("number:%s", id)

	if err := h.delSerialize(ctx, key); err != nil {
		return err
	}

	return nil
}
