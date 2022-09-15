package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
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

// ChatSet sets the chat info into the cache.
func (h *handler) ChatSet(ctx context.Context, data *chat.Chat) error {
	key := fmt.Sprintf("chat:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// ChatGet returns cached chat info
func (h *handler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	key := fmt.Sprintf("chat:%s", id)

	var res chat.Chat
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatroomSet sets the chatroom info into the cache
func (h *handler) ChatroomSet(ctx context.Context, af *chatroom.Chatroom) error {
	key := fmt.Sprintf("chatroom:%s", af.ID)

	if err := h.setSerialize(ctx, key, af); err != nil {
		return err
	}

	return nil
}

// ChatroomGet returns cached chatroom info
func (h *handler) ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	key := fmt.Sprintf("chatroom:%s", id)

	var res chatroom.Chatroom
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessagechatSet sets the messagechat info into the cache
func (h *handler) MessagechatSet(ctx context.Context, t *messagechat.Messagechat) error {
	key := fmt.Sprintf("messagechat:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// MessagechatGet returns cached messagechat info
func (h *handler) MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	key := fmt.Sprintf("messagechat:%s", id)

	var res messagechat.Messagechat
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessagechatroomSet sets the messagechatroom info into the cache
func (h *handler) MessagechatroomSet(ctx context.Context, t *messagechatroom.Messagechatroom) error {
	key := fmt.Sprintf("messagechatroom:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// MessagechatroomGet returns cached messagechatroom info
func (h *handler) MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	key := fmt.Sprintf("messagechatroom:%s", id)

	var res messagechatroom.Messagechatroom
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
