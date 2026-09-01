package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
)

// scanBatchSize is the COUNT hint handed to Redis SCAN. The key family is bounded by the number
// of live Asterisk instances (single digits today), so one batch normally covers everything and
// the cursor loop exits after a single round trip.
const scanBatchSize = 100

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache *redis.Client
}

// CacheHandler is sentinel-manager's read-only view of the Redis namespace that
// voip-asterisk-proxy owns. Nothing in this service ever writes to it.
type CacheHandler interface {
	Connect() error

	// AsteriskAddressInternalScan returns every `asterisk.<id>.address-internal` key currently
	// present, with its value and REMAINING ttl. The ttl is what makes the reverse (IP ->
	// asterisk-id) direction usable at all -- see models/asteriskaddress.
	AsteriskAddressInternalScan(ctx context.Context) ([]*asteriskaddress.AsteriskAddress, error)
}

// NewHandler creates a CacheHandler. The redis client is created eagerly; Connect verifies
// reachability.
func NewHandler(addr string, password string, db int) CacheHandler {
	return &handler{
		Addr:     addr,
		Password: password,
		DB:       db,

		Cache: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}
