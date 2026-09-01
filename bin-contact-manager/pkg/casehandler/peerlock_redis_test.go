package casehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/mock/gomock"
)

// testCounterValue reads the current value of a prometheus.Counter
// directly via its Write method, avoiding a new dependency on
// prometheus/client_golang/prometheus/testutil (not otherwise used
// anywhere in this monorepo) just for this one assertion.
func testCounterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("could not read counter value: %v", err)
	}
	return m.GetCounter().GetValue()
}

// Test_tryAcquireRedisLock_nilLocker_failsOpen verifies main.go's
// documented nil-locker contract: a caseHandler with redisLocker == nil
// (as produced by every caseHandler{...} struct-literal test in this
// package, and by contact-control's NewCaseHandler(..., nil) call) skips
// the Redis lock entirely and returns a no-op release with no error.
func Test_tryAcquireRedisLock_nilLocker_failsOpen(t *testing.T) {
	h := &caseHandler{}

	release, err := h.tryAcquireRedisLock(context.Background(), "customer1|tel|+15551234567|call")
	if err != nil {
		t.Fatalf("tryAcquireRedisLock() error = %v, want nil (fail-open on nil locker)", err)
	}
	if release == nil {
		t.Fatal("expected a non-nil no-op release func")
	}
	release() // must not panic
}

// Test_tryAcquireRedisLock_lockAcquired_releaseCallsUnlock verifies the
// success path: LockContext succeeds, and the returned release func calls
// UnlockContext exactly once.
func Test_tryAcquireRedisLock_lockAcquired_releaseCallsUnlock(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	key := "customer1|tel|+15551234567|call"
	mockLocker.EXPECT().
		NewMutex(redisPeerLockName(key), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(nil)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Return(true, nil)

	h := &caseHandler{redisLocker: mockLocker}

	release, err := h.tryAcquireRedisLock(context.Background(), key)
	if err != nil {
		t.Fatalf("tryAcquireRedisLock() error = %v, want nil", err)
	}
	if release == nil {
		t.Fatal("expected a non-nil release func")
	}
	release()
}

// Test_tryAcquireRedisLock_lockAcquired_releaseLogsUnlockFailure verifies
// that a failed UnlockContext is swallowed (logged, not propagated) --
// release funcs must never panic or return an error the caller must
// handle, mirroring the in-process release's unconditional semantics.
func Test_tryAcquireRedisLock_lockAcquired_releaseLogsUnlockFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	key := "customer1|tel|+15551234567|call"
	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(nil)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Return(false, errors.New("connection reset"))

	h := &caseHandler{redisLocker: mockLocker}

	release, err := h.tryAcquireRedisLock(context.Background(), key)
	if err != nil {
		t.Fatalf("tryAcquireRedisLock() error = %v, want nil", err)
	}
	release() // must not panic despite UnlockContext failing
}

// Test_tryAcquireRedisLock_errTaken_failsClosed verifies that ErrTaken
// (another replica currently holds the peer tuple's lock, budget still
// live) is returned as-is -- a fail-closed outcome distinct from the
// RedisError fail-open branch.
func Test_tryAcquireRedisLock_errTaken_failsClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(&redsync.ErrTaken{Nodes: []int{0}})

	h := &caseHandler{redisLocker: mockLocker}

	release, err := h.tryAcquireRedisLock(context.Background(), "customer1|tel|+15551234567|call")
	if err == nil {
		t.Fatal("expected a non-nil error (fail-closed on ErrTaken)")
	}
	if !errors.As(err, new(*redsync.ErrTaken)) {
		t.Errorf("expected the returned error to be (or wrap) *redsync.ErrTaken, got: %v (%T)", err, err)
	}
	if release != nil {
		t.Error("expected a nil release func on failure")
	}
}

// Test_tryAcquireRedisLock_redisError_failsOpen verifies that a
// *redsync.RedisError (Redis connectivity failure, budget still live) is
// treated as fail-open: a no-op release and nil error, so GetOrCreate
// proceeds on the in-process lock alone.
func Test_tryAcquireRedisLock_redisError_failsOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(&redsync.RedisError{Node: 0, Err: errors.New("connection refused")})

	h := &caseHandler{redisLocker: mockLocker}

	release, err := h.tryAcquireRedisLock(context.Background(), "customer1|tel|+15551234567|call")
	if err != nil {
		t.Fatalf("tryAcquireRedisLock() error = %v, want nil (fail-open on RedisError)", err)
	}
	if release == nil {
		t.Fatal("expected a non-nil no-op release func")
	}
	release() // must not panic (no real mutex behind it)
}

// Test_tryAcquireRedisLock_budgetExhausted_failsClosedEvenIfWrappedAsRedisError
// is the regression test for the 2nd-round design review's core fix:
// when the CALLER's ctx already carries a deadline shorter than
// redisPeerLockBudget, budgetCtx inherits and can expire mid-LockContext.
// redsync v4.15.0 can, in that exact timing window, surface the resulting
// failure wrapped as a *redsync.RedisError (actOnPoolsAsync wraps an
// in-flight SetNX cancellation this way) rather than a plain context
// error. This test forces that ambiguous shape (an outer ctx that is
// already expired by the time LockContext returns a RedisError) and
// asserts the budgetCtx.Err() check still wins: the call MUST fail
// closed (a plain, non-RedisError-typed wrapped error), not fall through
// to the fail-open branch above -- verifying the deterministic ordering
// mandated by peerlock.go's tryAcquireRedisLock comment.
func Test_tryAcquireRedisLock_budgetExhausted_failsClosedEvenIfWrappedAsRedisError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		// Block until the caller-supplied ctx (and therefore the
		// derived budgetCtx, which inherits the earlier of the two
		// deadlines) has actually expired, then return the ambiguous
		// RedisError shape redsync can produce in this exact window.
		<-ctx.Done()
		return &redsync.RedisError{Node: 0, Err: ctx.Err()}
	})

	h := &caseHandler{redisLocker: mockLocker}

	// A 5ms deadline is far shorter than redisPeerLockBudget (1s), so
	// budgetCtx (context.WithTimeout(ctx, redisPeerLockBudget)) inherits
	// and expires at this 5ms mark instead.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	release, err := h.tryAcquireRedisLock(ctx, "customer1|tel|+15551234567|call")
	if err == nil {
		t.Fatal("expected a non-nil error (fail-closed on budget exhaustion)")
	}
	if release != nil {
		t.Error("expected a nil release func on failure")
	}
	// Must NOT take the fail-open branch: promRedisPeerLockFailOpenTotal
	// only increments there, and errors.As would otherwise have matched
	// here too (the error IS a *redsync.RedisError) if the ordering bug
	// this test guards against were reintroduced. The strongest available
	// behavioral signal is the non-nil error itself (fail-open always
	// returns err == nil), asserted above.
}

// Test_acquirePeerLock_redisLockFails_releasesInProcessLockAndIncrementsCounter
// verifies acquirePeerLock's integration of the two lock layers: when the
// Redis layer fails (ErrTaken here), the already-acquired in-process lock
// must be released (so the channel isn't leaked) and the wrapped error
// propagated.
func Test_acquirePeerLock_redisLockFails_releasesInProcessLockAndIncrementsCounter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	key := "customer1|tel|+15551234567|call"
	mockLocker.EXPECT().NewMutex(redisPeerLockName(key), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(&redsync.ErrTaken{Nodes: []int{0}})

	h := &caseHandler{redisLocker: mockLocker, peerLocks: make(map[string]chan struct{})}

	before := testCounterValue(t, promRedisPeerLockFailTotal)

	release, err := h.acquirePeerLock(context.Background(), key)
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if release != nil {
		t.Error("expected a nil release func on failure")
	}

	after := testCounterValue(t, promRedisPeerLockFailTotal)
	if after != before+1 {
		t.Errorf("expected promRedisPeerLockFailTotal to increment by exactly 1, got before=%v after=%v", before, after)
	}

	// The in-process channel must have been released back to empty
	// (capacity 1, 0 currently held) -- confirm a subsequent acquire on
	// the SAME key does not block.
	h.peerLocksMu.RLock()
	ch := h.peerLocks[key]
	h.peerLocksMu.RUnlock()
	select {
	case ch <- struct{}{}:
		<-ch
	default:
		t.Error("expected the in-process peer lock channel to have been released after the Redis lock failure")
	}
}
