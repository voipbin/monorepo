package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
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
