package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"monorepo/bin-scheduler-manager/models/schedule"
)

// ErrLockBusy is returned by LockSchedule when another replica already holds
// the schedule's dispatch lock. Callers skip the slot silently (design §5.3).
var ErrLockBusy = fmt.Errorf("schedule lock busy")

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache  *redis.Client
	Locker *redsync.Redsync
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	ScheduleGet(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error)
	ScheduleSet(ctx context.Context, s *schedule.Schedule) error
	ScheduleDelete(ctx context.Context, id uuid.UUID) error

	LockSchedule(ctx context.Context, scheduleID uuid.UUID, ttl time.Duration) (func(), error)
}

// NewHandler creates CacheHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// redis lock
	pool := goredis.NewPool(cache)
	locker := redsync.New(pool)

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
		Locker:   locker,
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

// LockSchedule acquires the schedule's dispatch lock with try-once semantics
// (redsync.WithTries(1), design §5.3). On contention it returns ErrLockBusy.
// The returned func releases the lock; unlock errors are ignored (debug-logged)
// because correctness never depends on the lock — the DB CAS does the real work.
func (h *handler) LockSchedule(ctx context.Context, scheduleID uuid.UUID, ttl time.Duration) (func(), error) {
	mutex := h.Locker.NewMutex(scheduleLockKey(scheduleID), redsync.WithExpiry(ttl), redsync.WithTries(1))

	if errLock := mutex.TryLockContext(ctx); errLock != nil {
		if isLockBusyError(errLock) {
			return nil, ErrLockBusy
		}
		return nil, fmt.Errorf("could not acquire schedule lock. LockSchedule. err: %v", errLock)
	}

	unlock := func() {
		if _, errUnlock := mutex.UnlockContext(context.Background()); errUnlock != nil {
			logrus.Debugf("Could not unlock the schedule lock. scheduleID: %s, err: %v", scheduleID, errUnlock)
		}
	}

	return unlock, nil
}

// isLockBusyError reports whether the given redsync error means the lock is
// held by someone else (as opposed to e.g. a redis connectivity failure).
func isLockBusyError(err error) bool {
	if errors.Is(err, redsync.ErrFailed) {
		return true
	}

	var errTaken *redsync.ErrTaken
	if errors.As(err, &errTaken) {
		return true
	}

	var errNodeTaken *redsync.ErrNodeTaken
	return errors.As(err, &errNodeTaken)
}
