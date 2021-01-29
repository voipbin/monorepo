package cachehandler

//go:generate mockgen -destination ./mock_cachehandler_cachehandler.go -package cachehandler -source ./main.go CacheHandler

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/number"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
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

	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeSet(ctx context.Context, bridge *bridge.Bridge) error

	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallSet(ctx context.Context, call *call.Call) error

	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelSet(ctx context.Context, channel *channel.Channel) error

	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceSet(ctx context.Context, conference *conference.Conference) error

	NumberGetByNumber(ctx context.Context, num string) (*number.Number, error)
	NumberSetByNumber(ctx context.Context, numb *number.Number) error

	RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	RecordingSet(ctx context.Context, record *recording.Recording) error
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
