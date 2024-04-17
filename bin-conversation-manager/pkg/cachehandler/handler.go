package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
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

// AccountSet sets the account info into the cache.
func (h *handler) AccountSet(ctx context.Context, a *account.Account) error {
	key := fmt.Sprintf("conversation.account:%s", a.ID)

	if err := h.setSerialize(ctx, key, a); err != nil {
		return err
	}

	return nil
}

// AccountGet returns cached account info
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	key := fmt.Sprintf("conversation.account:%s", id)

	var res account.Account
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationSet sets the conversation info into the cache.
func (h *handler) ConversationSet(ctx context.Context, cv *conversation.Conversation) error {
	key := fmt.Sprintf("conversation.conversation:%s", cv.ID)

	if err := h.setSerialize(ctx, key, cv); err != nil {
		return err
	}

	return nil
}

// ConversationGet returns cached conversation info
func (h *handler) ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {
	key := fmt.Sprintf("conversation.conversation:%s", id)

	var res conversation.Conversation
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageSet sets the conversation message info into the cache.
func (h *handler) MessageSet(ctx context.Context, cv *message.Message) error {
	key := fmt.Sprintf("conversation.message:%s", cv.ID)

	if err := h.setSerialize(ctx, key, cv); err != nil {
		return err
	}

	return nil
}

// MessageGet returns cached message info
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	key := fmt.Sprintf("conversation.message:%s", id)

	var res message.Message
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MediaSet sets the conversation message info into the cache.
func (h *handler) MediaSet(ctx context.Context, m *media.Media) error {
	key := fmt.Sprintf("conversation.media:%s", m.ID)

	if err := h.setSerialize(ctx, key, m); err != nil {
		return err
	}

	return nil
}

// MediaGet returns cached message info
func (h *handler) MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error) {
	key := fmt.Sprintf("conversation.media:%s", id)

	var res media.Media
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
