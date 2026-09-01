package healthcheckhandler

import (
	"context"
	"testing"

	"github.com/go-redsync/redsync/v4"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// Test_NewRedsyncLocker_adaptsRealRedsync exercises the redsyncLocker
// adapter against a genuine *redsync.Redsync (no live Redis needed - built
// with zero backing pools) to cover the production wiring path
// (NewRedsyncLocker / redsyncLocker.NewMutex) that every other test in this
// package bypasses via Mocklocker.
func Test_NewRedsyncLocker_adaptsRealRedsync(t *testing.T) {
	rs := redsync.New() // zero pools - never actually talks to Redis
	l := NewRedsyncLocker(rs)

	mutex := l.NewMutex(healthCheckLockName, redsync.WithTries(1))
	if mutex == nil {
		t.Fatal("Expected a non-nil mutex from NewMutex")
	}

	// Quorum (len(pools)/2+1 = 1) can never be reached with zero backing
	// pools, so this always fails - confirming the adapter genuinely
	// round-trips into a working *redsync.Mutex rather than a stub.
	if err := mutex.TryLockContext(context.Background()); err == nil {
		t.Error("Expected an error acquiring a lock with zero backing pools, got nil")
	}
}

// Test_runOnce_lockGenericError_failsOpen exercises acquireLock's final
// catch-all branch (an error that is neither *redsync.ErrTaken nor
// *redsync.RedisError - here, redsync.ErrFailed from a zero-pool quorum
// miss) using a real *redsync.Redsync built via NewHealthCheckHandler /
// NewRedsyncLocker, the same construction path cmd/route-manager/main.go
// uses in production.
func Test_runOnce_lockGenericError_failsOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{}, nil)

	rs := redsync.New() // zero pools - TryLockContext fails with the ErrFailed sentinel, not ErrTaken/RedisError
	h := NewHealthCheckHandler(mockDB, mockReq, NewRedsyncLocker(rs))

	hh, ok := h.(*healthCheckHandler)
	if !ok {
		t.Fatalf("Expected NewHealthCheckHandler to return *healthCheckHandler, got %T", h)
	}
	if err := hh.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error (fail-open on an unrecognized lock error), got: %v", err)
	}
}
