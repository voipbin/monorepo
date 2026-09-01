package healthcheckhandler

//go:generate mockgen -package healthcheckhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// HealthCheckHandler manages periodic SIP OPTIONS health checks for providers.
type HealthCheckHandler interface {
	Run(ctx context.Context, interval time.Duration)
}

// locker is the minimal Redsync surface runOnce depends on. *redsync.Redsync
// cannot satisfy this directly (NewMutex returns the concrete *redsync.Mutex,
// not an interface), so redsyncLocker below adapts it. Abstracting this way
// lets tests substitute a mock without a live Redis instance, mirroring how
// bin-flow-manager/pkg/activeflowhandler mocks ActiveflowGetWithLock one
// layer above the lock library itself.
type locker interface {
	NewMutex(name string, options ...redsync.Option) redsyncMutex
}

// redsyncMutex is the subset of *redsync.Mutex's method set runOnce uses.
type redsyncMutex interface {
	TryLockContext(ctx context.Context) error
	ExtendContext(ctx context.Context) (bool, error)
	UnlockContext(ctx context.Context) (bool, error)
}

// redsyncLocker adapts *redsync.Redsync to the locker interface.
type redsyncLocker struct {
	rs *redsync.Redsync
}

// NewRedsyncLocker wraps a *redsync.Redsync so it satisfies locker.
func NewRedsyncLocker(rs *redsync.Redsync) locker { //nolint:revive // unexported return type is intentional; locker is only consumed within this package
	return &redsyncLocker{rs: rs}
}

func (r *redsyncLocker) NewMutex(name string, options ...redsync.Option) redsyncMutex {
	return r.rs.NewMutex(name, options...)
}

type healthCheckHandler struct {
	db         dbhandler.DBHandler
	reqHandler requesthandler.RequestHandler
	locker     locker
}

// NewHealthCheckHandler creates a HealthCheckHandler. l guards runOnce so
// only one replica probes providers per cycle - build it via
// NewRedsyncLocker(redsync.New(pool)).
func NewHealthCheckHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler, l locker) HealthCheckHandler { //nolint:revive // unexported param type is intentional; locker is only constructed within this package via NewRedsyncLocker
	return &healthCheckHandler{
		db:         db,
		reqHandler: reqHandler,
		locker:     l,
	}
}
