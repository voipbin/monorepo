package cachehandler

//go:generate mockgen -destination ./mock_cachehandler_cachehandler.go -package cachehandler gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler CacheHandler

import (
	"context"

	"github.com/go-redis/redis/v8"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
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

	AsteriskAddressInternerGet(ctx context.Context, id string) (string, error)

	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelSet(ctx context.Context, channel *channel.Channel) error

	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeSet(ctx context.Context, bridge *bridge.Bridge) error
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
