package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/chatbot"
	"monorepo/bin-ai-manager/models/chatbotcall"
	"monorepo/bin-ai-manager/models/message"
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

	ChatbotGet(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error)
	ChatbotSet(ctx context.Context, data *chatbot.Chatbot) error

	ChatbotcallGet(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallSet(ctx context.Context, data *chatbotcall.Chatbotcall) error

	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageSet(ctx context.Context, data *message.Message) error
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
