package dispatchhandler

//go:generate mockgen -package dispatchhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

const (
	// scheduleLockTTL is the redis dispatch-lock expiry (design §5.3). The
	// lock guards the claim critical section only, never the RPC.
	scheduleLockTTL = 30 * time.Second

	// defaultRetryBackoff is the linear backoff between in-run retry
	// attempts (design §5.5).
	defaultRetryBackoff = 5 * time.Second

	// resultMaxBytes bounds the stored RPC response body (design §4.2).
	resultMaxBytes = 60000

	// scanLimit bounds the rows fetched per tick per query.
	scanLimit = 1000

	// defaultTickIntervalSec/defaultConcurrency guard against a zero-valued
	// configuration (design §11 defaults).
	defaultTickIntervalSec = 10
	defaultConcurrency     = 10
)

// DispatchHandler interface
type DispatchHandler interface {
	Run(ctx context.Context)
	ExecuteManual(ctx context.Context, scheduleID uuid.UUID) (*execution.Execution, error)
}

type dispatchHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
	notifyHandler notifyhandler.NotifyHandler

	tickInterval time.Duration
	retryBackoff time.Duration // injectable in tests

	// sem bounds the number of in-flight dispatches per replica (design §5.2).
	sem chan struct{}

	// muOverlap guards lastOverlapSkip: schedule id -> the tm_scheduled slot
	// whose overlap skip was already counted, so skipped_overlap increments
	// once per skipped slot, not once per tick (design §5.7).
	muOverlap       sync.Mutex
	lastOverlapSkip map[uuid.UUID]time.Time
}

// NewDispatchHandler returns DispatchHandler interface
func NewDispatchHandler(
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	tickIntervalSec int,
	concurrency int,
) DispatchHandler {
	if tickIntervalSec <= 0 {
		tickIntervalSec = defaultTickIntervalSec
	}
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	return &dispatchHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		cache:         cache,
		notifyHandler: notifyHandler,

		tickInterval: time.Duration(tickIntervalSec) * time.Second,
		retryBackoff: defaultRetryBackoff,

		sem:             make(chan struct{}, concurrency),
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}
}
