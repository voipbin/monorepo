package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/models/recording"
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

	AsteriskAddressInternalGet(ctx context.Context, id string) (string, error)

	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeSet(ctx context.Context, bridge *bridge.Bridge) error

	CallAppAMDGet(ctx context.Context, channelID string) (*callapplication.AMD, error)
	CallAppAMDSet(ctx context.Context, channelID string, app *callapplication.AMD) error

	ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error)
	ExternalMediaGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*externalmedia.ExternalMedia, error)
	ExternalMediaSet(ctx context.Context, data *externalmedia.ExternalMedia) error
	ExternalMediaDelete(ctx context.Context, externalMediaID uuid.UUID) error

	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallSet(ctx context.Context, call *call.Call) error

	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelSet(ctx context.Context, channel *channel.Channel) error

	ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	ConfbridgeSet(ctx context.Context, data *confbridge.Confbridge) error

	GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)
	GroupcallSet(ctx context.Context, data *groupcall.Groupcall) error

	RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	RecordingSet(ctx context.Context, record *recording.Recording) error

	KamailioMetadataGet(ctx context.Context, sipCallID string) (map[string]string, error)

	// OutboundConfigGet returns a cached OutboundConfig for customerID.
	// Returns (nil, nil) when key exists but is a negative-cache sentinel (no DB row).
	// Returns (nil, redis.Nil) when key is absent (cache miss).
	OutboundConfigGet(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error)

	// OutboundConfigSet caches the config for customerID with a 30-minute TTL.
	OutboundConfigSet(ctx context.Context, customerID uuid.UUID, c *outboundconfig.OutboundConfig) error

	// OutboundConfigSetNotFound caches a not-found sentinel for customerID (1-minute TTL).
	// Prevents DB hammering for customers with no config row.
	OutboundConfigSetNotFound(ctx context.Context, customerID uuid.UUID) error

	// OutboundConfigDelete removes the cached config for customerID.
	OutboundConfigDelete(ctx context.Context, customerID uuid.UUID) error

	// RateLimitIncrement atomically increments the counter at key (INCR) and
	// unconditionally attempts to set a TTL on it via EXPIRE...NX (only takes effect
	// if the key has no TTL yet). Returns the counter's new value after the increment.
	// Calling ExpireNX unconditionally on every request (instead of only when
	// count==1) closes a permanent-lockout gap: if the process crashes between INCR
	// and EXPIRE (e.g. pod restart during a rolling deploy), the key would otherwise
	// keep incrementing forever with no TTL. Requires Redis 7.0+ (ExpireNX). VOIP-1259.
	RateLimitIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error)
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
