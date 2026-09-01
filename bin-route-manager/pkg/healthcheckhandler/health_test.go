package healthcheckhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// newAlwaysSucceedsLocker returns a Mocklocker whose NewMutex yields a mutex
// that always succeeds on TryLock/Extend/Unlock. Used by tests that exercise
// runOnce's provider-list/probe behavior and are not themselves about lock
// semantics - mirrors uncontested single-replica behavior, where the lock is
// always immediately acquirable.
func newAlwaysSucceedsLocker(mc *gomock.Controller) *Mocklocker {
	mockMutex := NewMockredsyncMutex(mc)
	mockMutex.EXPECT().TryLockContext(gomock.Any()).Return(nil).AnyTimes()
	mockMutex.EXPECT().ExtendContext(gomock.Any()).Return(true, nil).AnyTimes()
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Return(true, nil).AnyTimes()

	mockLocker := NewMocklocker(mc)
	mockLocker.EXPECT().NewMutex(healthCheckLockName, gomock.Any()).Return(mockMutex).AnyTimes()
	return mockLocker
}

func Test_runOnce_noProviders(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	// ProviderList returns empty list → no health checks performed
	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{}, nil)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func Test_runOnce_oneProvider_healthy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	now := time.Now()
	p := &provider.Provider{
		ID:       [16]byte{1},
		Hostname: "sip.telnyx.com",
		TMCreate: &now,
	}

	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{p}, nil)

	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, "sip.telnyx.com").
		Return(&requesthandler.KamailioProviderHealthResult{Status: "healthy", ResultCode: "200"}, nil)

	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, p.ID, "healthy", gomock.Any()).
		Return(nil)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func Test_runOnce_oneProvider_unhealthy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	now := time.Now()
	p := &provider.Provider{
		ID:       [16]byte{2},
		Hostname: "sip.dead.example.com",
		TMCreate: &now,
	}

	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{p}, nil)

	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, "sip.dead.example.com").
		Return(&requesthandler.KamailioProviderHealthResult{Status: "unhealthy", ResultCode: "timeout"}, nil)

	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, p.ID, "unhealthy", gomock.Any()).
		Return(nil)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func Test_runOnce_providerListError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return(nil, fmt.Errorf("db error"))

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err == nil {
		t.Error("Expected error from ProviderList failure, got nil")
	}
}

func Test_runOnce_multiPage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	// Build a full page (100 providers) + a partial second page (1 provider)
	t1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	page1 := make([]*provider.Provider, healthCheckPageSize)
	for i := range page1 {
		page1[i] = &provider.Provider{
			ID:       [16]byte{byte(i + 1)},
			Hostname: fmt.Sprintf("sip%d.example.com", i),
			TMCreate: &t1,
		}
	}
	page2 := []*provider.Provider{
		{ID: [16]byte{200}, Hostname: "sip200.example.com", TMCreate: &t2},
	}

	nextToken := t1.UTC().Format(timeLayout)

	gomock.InOrder(
		mockDB.EXPECT().
			ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
			Return(page1, nil),
		mockDB.EXPECT().
			ProviderList(ctx, nextToken, healthCheckPageSize, map[provider.Field]any{}).
			Return(page2, nil),
	)

	// Expect health check + update for all 101 providers
	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, gomock.Any()).
		Return(&requesthandler.KamailioProviderHealthResult{Status: "healthy", ResultCode: "200"}, nil).
		Times(int(healthCheckPageSize) + 1)

	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, gomock.Any(), "healthy", gomock.Any()).
		Return(nil).
		Times(int(healthCheckPageSize) + 1)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func Test_checkProvider_rpcError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	p := &provider.Provider{ID: [16]byte{1}, Hostname: "sip.example.com"}

	// RPC error → no update call
	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, "sip.example.com").
		Return(nil, fmt.Errorf("rpc timeout"))

	// checkProvider does not touch the lock, so this test does not need
	// (and does not construct) a locker.
	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
	// Should not panic or propagate error
	h.checkProvider(ctx, p)
}

func Test_runOnce_lockPeerHeld_skipsCycle(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockMutex := NewMockredsyncMutex(mc)
	mockMutex.EXPECT().TryLockContext(ctx).Return(&redsync.ErrTaken{Nodes: []int{0}})
	// A skipped cycle must never touch the DB, never probe, and never
	// attempt to unlock a lock it never acquired.
	mockMutex.EXPECT().ExtendContext(gomock.Any()).Times(0)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Times(0)

	mockLocker := NewMocklocker(mc)
	mockLocker.EXPECT().NewMutex(healthCheckLockName, gomock.Any()).Return(mockMutex)

	mockDB.EXPECT().ProviderList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockReq.EXPECT().KamailioV1ProviderHealthCheck(gomock.Any(), gomock.Any()).Times(0)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: mockLocker}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error (skip is not a failure), got: %v", err)
	}
}

func Test_runOnce_lockRedisError_failsOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	now := time.Now()
	p := &provider.Provider{ID: [16]byte{3}, Hostname: "sip.example.com", TMCreate: &now}

	mockMutex := NewMockredsyncMutex(mc)
	mockMutex.EXPECT().TryLockContext(ctx).Return(&redsync.RedisError{Node: 0, Err: errors.New("connection refused")})
	// Never acquired, so must not extend or unlock.
	mockMutex.EXPECT().ExtendContext(gomock.Any()).Times(0)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Times(0)

	mockLocker := NewMocklocker(mc)
	mockLocker.EXPECT().NewMutex(healthCheckLockName, gomock.Any()).Return(mockMutex)

	// Fail-open: the cycle still runs unlocked.
	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{p}, nil)
	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, "sip.example.com").
		Return(&requesthandler.KamailioProviderHealthResult{Status: "healthy", ResultCode: "200"}, nil)
	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, p.ID, "healthy", gomock.Any()).
		Return(nil)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: mockLocker}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error (fail-open still completes the cycle), got: %v", err)
	}
}

func Test_runOnce_extendFails_continuesCycle(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	t1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	page1 := make([]*provider.Provider, healthCheckPageSize)
	for i := range page1 {
		page1[i] = &provider.Provider{
			ID:       [16]byte{byte(i + 1)},
			Hostname: fmt.Sprintf("sip%d.example.com", i),
			TMCreate: &t1,
		}
	}
	page2 := []*provider.Provider{
		{ID: [16]byte{200}, Hostname: "sip200.example.com", TMCreate: &t2},
	}
	nextToken := t1.UTC().Format(timeLayout)

	mockMutex := NewMockredsyncMutex(mc)
	mockMutex.EXPECT().TryLockContext(ctx).Return(nil)
	// The lock was genuinely acquired, so the deferred unlock still runs
	// even though the mid-cycle extend below fails. Exactly 2 calls (one
	// per page) - not AnyTimes() - so this test fails if a future
	// refactor accidentally drops the per-page Extend call.
	mockMutex.EXPECT().ExtendContext(gomock.Any()).Return(false, redsync.ErrExtendFailed).Times(2)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Return(true, nil)

	mockLocker := NewMocklocker(mc)
	mockLocker.EXPECT().NewMutex(healthCheckLockName, gomock.Any()).Return(mockMutex)

	gomock.InOrder(
		mockDB.EXPECT().
			ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
			Return(page1, nil),
		mockDB.EXPECT().
			ProviderList(ctx, nextToken, healthCheckPageSize, map[provider.Field]any{}).
			Return(page2, nil),
	)
	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, gomock.Any()).
		Return(&requesthandler.KamailioProviderHealthResult{Status: "healthy", ResultCode: "200"}, nil).
		Times(int(healthCheckPageSize) + 1)
	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, gomock.Any(), "healthy", gomock.Any()).
		Return(nil).
		Times(int(healthCheckPageSize) + 1)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: mockLocker}
	// An Extend failure mid-cycle must not abort the cycle - both pages
	// (all 101 providers) are still checked.
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Expected nil error (extend failure fails open, cycle continues), got: %v", err)
	}
}
