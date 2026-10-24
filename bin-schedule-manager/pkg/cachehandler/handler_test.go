package cachehandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-schedule-manager/models/schedule"
)

func newTestHandler(t *testing.T) (CacheHandler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h := NewHandler(mr.Addr(), "", 0)

	return h, mr
}

func newTestSchedule(id string, next *time.Time) *schedule.Schedule {
	return &schedule.Schedule{
		Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil(id)},
		Name:         "cache-test",
		Cron:         "0 1 * * *",
		TargetQueue:  "bin-manager.test.request",
		TargetURI:    "/v1/test",
		TargetMethod: "POST",
		TimeoutMS:    1500,
		RetryMax:     2,
		Enabled:      true,
		TMNextRun:    next,
	}
}

func Test_ScheduleSet_ScheduleGet(t *testing.T) {
	next := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		next *time.Time
	}{
		{name: "with next run", next: &next},
		{name: "without next run", next: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)
			ctx := context.Background()

			s := newTestSchedule("5f9c1b2a-6f13-11f0-c001-0242ac110002", tt.next)

			if err := h.ScheduleSet(ctx, s); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			key := "schedule:5f9c1b2a-6f13-11f0-c001-0242ac110002"
			if !mr.Exists(key) {
				t.Errorf("Wrong match. expect: key %s, got: none", key)
			}
			if ttl := mr.TTL(key); ttl != 24*time.Hour {
				t.Errorf("Wrong match. expect: 24h, got: %v", ttl)
			}

			res, err := h.ScheduleGet(ctx, s.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID != s.ID || res.Name != s.Name || res.Cron != s.Cron || res.TargetQueue != s.TargetQueue ||
				res.TargetURI != s.TargetURI || res.TargetMethod != s.TargetMethod ||
				res.TimeoutMS != s.TimeoutMS || res.RetryMax != s.RetryMax || res.Enabled != s.Enabled {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", s, res)
			}

			switch {
			case tt.next == nil && res.TMNextRun != nil:
				t.Errorf("Wrong match. expect: nil next run, got: %v", res.TMNextRun)
			case tt.next != nil && (res.TMNextRun == nil || !res.TMNextRun.Equal(*tt.next)):
				t.Errorf("Wrong match. expect: %v, got: %v", tt.next, res.TMNextRun)
			}
		})
	}
}

func Test_ScheduleGet_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002")

	t.Run("miss", func(t *testing.T) {
		h, _ := newTestHandler(t)

		res, err := h.ScheduleGet(context.Background(), id)
		if !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})

	t.Run("corrupt json", func(t *testing.T) {
		h, mr := newTestHandler(t)

		if err := mr.Set("schedule:"+id.String(), "{not json"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res, err := h.ScheduleGet(context.Background(), id)
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("Wrong match. expect: json syntax error, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})
}

func Test_ScheduleDelete(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	s := newTestSchedule("5f9c1b2a-6f13-11f0-c001-0242ac110002", nil)
	other := newTestSchedule("5f9c1b2a-6f13-11f0-c001-0242ac110003", nil)
	for _, item := range []*schedule.Schedule{s, other} {
		if err := h.ScheduleSet(ctx, item); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	if err := h.ScheduleDelete(ctx, s.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists("schedule:" + s.ID.String()) {
		t.Errorf("Wrong match. expect: key deleted, got: still exists")
	}
	if !mr.Exists("schedule:" + other.ID.String()) {
		t.Errorf("Wrong match. expect: other key kept, got: deleted")
	}

	// deleting a missing key is not an error
	if err := h.ScheduleDelete(ctx, s.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Connect(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server closed, got: nil")
	}
}

func Test_server_down_errors(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	s := newTestSchedule("5f9c1b2a-6f13-11f0-c001-0242ac110002", nil)
	mr.Close()

	if err := h.ScheduleSet(ctx, s); err == nil {
		t.Errorf("Wrong match. expect: set error, got: nil")
	}
	if res, err := h.ScheduleGet(ctx, s.ID); err == nil || errors.Is(err, redis.Nil) || res != nil {
		t.Errorf("Wrong match. expect: connection error and nil result, got: %v, %v", res, err)
	}
	if err := h.ScheduleDelete(ctx, s.ID); err == nil {
		t.Errorf("Wrong match. expect: delete error, got: nil")
	}

	unlock, err := h.LockSchedule(ctx, s.ID, time.Minute)
	if err == nil || errors.Is(err, ErrLockBusy) || unlock != nil {
		t.Errorf("Wrong match. expect: non-busy error and nil unlock, got: %v, %v", unlock != nil, err)
	}
}

func Test_LockSchedule(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002")
	otherID := uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110003")
	lockKey := "schedule:lock:" + id.String()
	ttl := 30 * time.Second

	unlock, err := h.LockSchedule(ctx, id, ttl)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !mr.Exists(lockKey) {
		t.Errorf("Wrong match. expect: key %s, got: none", lockKey)
	}
	if got := mr.TTL(lockKey); got != ttl {
		t.Errorf("Wrong match. expect: %v, got: %v", ttl, got)
	}

	// a second caller on the same schedule is told the lock is busy
	if _, errBusy := h.LockSchedule(ctx, id, ttl); !errors.Is(errBusy, ErrLockBusy) {
		t.Errorf("Wrong match. expect: ErrLockBusy, got: %v", errBusy)
	}

	// another schedule is independent
	unlockOther, errOther := h.LockSchedule(ctx, otherID, ttl)
	if errOther != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errOther)
	} else {
		unlockOther()
	}

	// the release frees the lock for the next caller
	unlock()
	if mr.Exists(lockKey) {
		t.Errorf("Wrong match. expect: key released, got: still exists")
	}
	unlockAgain, errAgain := h.LockSchedule(ctx, id, ttl)
	if errAgain != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errAgain)
	}

	// the lock expires on its own, and releasing an expired lock is harmless
	mr.FastForward(ttl + time.Second)
	if mr.Exists(lockKey) {
		t.Errorf("Wrong match. expect: key expired, got: still exists")
	}
	unlockAfterExpiry, errExpired := h.LockSchedule(ctx, id, ttl)
	if errExpired != nil {
		t.Fatalf("Wrong match. expect: ok after expiry, got: %v", errExpired)
	}
	unlockAgain()
	if !mr.Exists(lockKey) {
		t.Errorf("Wrong match. expect: stale release keeps the new holder's lock, got: released")
	}
	unlockAfterExpiry()
}

func Test_LockSchedule_unlock_ignores_canceled_context(t *testing.T) {
	h, mr := newTestHandler(t)

	id := uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002")
	ctx, cancel := context.WithCancel(context.Background())

	unlock, err := h.LockSchedule(ctx, id, time.Minute)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	cancel()
	unlock()

	if mr.Exists("schedule:lock:" + id.String()) {
		t.Errorf("Wrong match. expect: lock released despite the canceled context, got: still exists")
	}
}

func Test_LockSchedule_canceled_context(t *testing.T) {
	h, mr := newTestHandler(t)

	id := uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	unlock, err := h.LockSchedule(ctx, id, time.Minute)
	if err == nil || errors.Is(err, ErrLockBusy) || unlock != nil {
		t.Errorf("Wrong match. expect: non-busy error and nil unlock, got: %v, %v", unlock != nil, err)
	}
	if mr.Exists("schedule:lock:" + id.String()) {
		t.Errorf("Wrong match. expect: no lock, got: key exists")
	}
}
