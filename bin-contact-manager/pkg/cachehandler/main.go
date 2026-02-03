package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache *redis.Client
}

// CacheHandler interface for contact caching operations
type CacheHandler interface {
	Connect() error

	// Contact cache operations
	ContactGet(ctx context.Context, id uuid.UUID) (*contact.Contact, error)
	ContactSet(ctx context.Context, c *contact.Contact) error
	ContactDelete(ctx context.Context, id uuid.UUID) error
}

// NewHandler creates CacheHandler
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
