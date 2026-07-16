package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
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

// WidgetGet returns cached widget info
func (h *handler) WidgetGet(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	key := fmt.Sprintf("webchat:widget:%s", id)

	var res widget.Widget
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// WidgetSet sets the widget info into the cache.
func (h *handler) WidgetSet(ctx context.Context, u *widget.Widget) error {
	key := fmt.Sprintf("webchat:widget:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// SessionGet returns cached session info
func (h *handler) SessionGet(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	key := fmt.Sprintf("webchat:session:%s", id)

	var res session.Session
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// SessionSet sets the session info into the cache.
func (h *handler) SessionSet(ctx context.Context, u *session.Session) error {
	key := fmt.Sprintf("webchat:session:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// MessageGet returns cached message info
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	key := fmt.Sprintf("webchat:message:%s", id)

	var res message.Message
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageSet sets the message info into the cache.
func (h *handler) MessageSet(ctx context.Context, u *message.Message) error {
	key := fmt.Sprintf("webchat:message:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
