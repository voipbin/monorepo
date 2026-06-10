package activeflowhandler

//go:generate mockgen -package activeflowhandler -destination ./mock_activeflowhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/singleflight"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// default cache TTLs (design 5.3 / 5.4). These are package-level vars so they
// can be overridden in tests; production wiring uses NewActiveflowHandler.
var (
	// defaultTLive is the TTL for a positive entry (cache lifetime safety net).
	defaultTLive = 24 * time.Hour
	// defaultTNeg is the TTL for a negative entry (no webhook / deleted).
	defaultTNeg = 10 * time.Minute
	// defaultTTransient is the very short TTL used for a transient NotFound.
	defaultTTransient = 5 * time.Second
)

// Destination is a resolved, deliverable per-activeflow webhook destination.
type Destination struct {
	URI    string
	Method webhook.MethodType
}

// ActiveflowHandler is the interface for the per-activeflow webhook resolver.
type ActiveflowHandler interface {
	// Get resolves the per-activeflow webhook destination for the given
	// activeflowID. It returns a non-nil Destination only when a positive,
	// deliverable destination is configured; it returns (nil, nil) when there is
	// no webhook (negative) or when delivery must be skipped (rpc error, etc.).
	Get(ctx context.Context, activeflowID uuid.UUID) (*Destination, error)
}

// activeflowHandler is the concrete resolver.
type activeflowHandler struct {
	cache      cachehandler.CacheHandler
	reqHandler requesthandler.RequestHandler

	sfGroup singleflight.Group

	tLive      time.Duration
	tNeg       time.Duration
	tTransient time.Duration
}

var (
	metricsNamespace = "webhook_manager"

	promActiveflowResolveTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_resolve_total",
			Help:      "Total number of per-activeflow webhook fallback resolutions by result.",
		},
		[]string{"result"},
	)

	promActiveflowCacheTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_cache_total",
			Help:      "Total number of per-activeflow webhook cache lookups by result.",
		},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(
		promActiveflowResolveTotal,
		promActiveflowCacheTotal,
	)
}

// NewActiveflowHandler returns a new ActiveflowHandler.
func NewActiveflowHandler(cache cachehandler.CacheHandler, reqHandler requesthandler.RequestHandler) ActiveflowHandler {
	return &activeflowHandler{
		cache:      cache,
		reqHandler: reqHandler,

		tLive:      defaultTLive,
		tNeg:       defaultTNeg,
		tTransient: defaultTTransient,
	}
}
