package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
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

	AccountSet(ctx context.Context, a *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)

	ConversationSet(ctx context.Context, cv *conversation.Conversation) error
	ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)

	MediaSet(ctx context.Context, m *media.Media) error
	MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error)

	MessageSet(ctx context.Context, cv *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
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
