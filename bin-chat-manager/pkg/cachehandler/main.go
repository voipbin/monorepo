package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/models/messagechatroom"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache *redis.Client
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	ChatSet(ctx context.Context, data *chat.Chat) error
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)

	ChatroomSet(ctx context.Context, af *chatroom.Chatroom) error
	ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error)

	MessagechatSet(ctx context.Context, t *messagechat.Messagechat) error
	MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error)

	MessagechatroomSet(ctx context.Context, t *messagechatroom.Messagechatroom) error
	MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error)
}

// NewHandler creates DBHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
	}

	return h
}

// Connect connects to the cache server
func (h *handler) Connect() error {
	_, err := h.Cache.Ping(context.Background()).Result()
	if err != nil {
		return err
	}

	return nil
}
