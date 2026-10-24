package healthcheckhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_Run_ticksUntilContextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// first cycle fails, which must not stop the loop; the second cycle succeeds
	listed := make(chan int, 64)
	calls := 0
	mockDB.EXPECT().
		ProviderList(gomock.Any(), "", healthCheckPageSize, map[provider.Field]any{}).
		DoAndReturn(func(_ context.Context, _ string, _ uint64, _ map[provider.Field]any) ([]*provider.Provider, error) {
			calls++
			listed <- calls
			if calls == 1 {
				return nil, errors.New("list failed")
			}
			return []*provider.Provider{}, nil
		}).
		MinTimes(2)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}

	done := make(chan struct{})
	go func() {
		h.Run(ctx, 50*time.Millisecond)
		close(done)
	}()

	for want := 1; want <= 2; want++ {
		select {
		case got := <-listed:
			if got != want {
				t.Errorf("Wrong match. expect: cycle %d, got: %d", want, got)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect: cycle %d started, got: timeout", want)
		}
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: Run returned after cancel, got: timeout")
	}
}

func Test_Run_cancelledBeforeFirstTick(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectations: a cancelled context must stop the loop before any cycle starts
	h := &healthCheckHandler{
		db:         dbhandler.NewMockDBHandler(mc),
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		locker:     NewMocklocker(mc),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		h.Run(ctx, time.Hour)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: Run returned, got: timeout")
	}
}

func Test_runOnce_unlockFails_cycleStillSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	mockMutex := NewMockredsyncMutex(mc)
	mockMutex.EXPECT().TryLockContext(ctx).Return(nil)
	mockMutex.EXPECT().ExtendContext(gomock.Any()).Return(true, nil)
	mockMutex.EXPECT().UnlockContext(gomock.Any()).Return(false, errors.New("unlock failed"))

	mockLocker := NewMocklocker(mc)
	mockLocker.EXPECT().NewMutex(healthCheckLockName, gomock.Any()).Return(mockMutex)

	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return([]*provider.Provider{}, nil)

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: mockLocker}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_runOnce_fullPageWithoutCreateTime_stops(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	page := make([]*provider.Provider, healthCheckPageSize)
	for i := range page {
		page[i] = &provider.Provider{
			ID:       [16]byte{byte(i + 1)},
			Hostname: fmt.Sprintf("sip%d.example.invalid", i),
		}
	}

	// the last provider has no create time, so no next-page token can be built: exactly one page is read
	mockDB.EXPECT().
		ProviderList(ctx, "", healthCheckPageSize, map[provider.Field]any{}).
		Return(page, nil).
		Times(1)
	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, gomock.Any()).
		Return(&requesthandler.KamailioProviderHealthResult{Status: "healthy", ResultCode: "200"}, nil).
		Times(int(healthCheckPageSize))
	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, gomock.Any(), "healthy", gomock.Any()).
		Return(nil).
		Times(int(healthCheckPageSize))

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	if err := h.runOnce(ctx); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_checkProvider_updateFails_doesNotPanic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	p := &provider.Provider{ID: [16]byte{9}, Hostname: "sip.example.invalid"}

	mockReq.EXPECT().
		KamailioV1ProviderHealthCheck(ctx, "sip.example.invalid").
		Return(&requesthandler.KamailioProviderHealthResult{Status: "unhealthy", ResultCode: "503"}, nil)
	mockDB.EXPECT().
		ProviderUpdateHealthStatus(ctx, p.ID, "unhealthy", gomock.Any()).
		Return(errors.New("update failed"))

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq, locker: newAlwaysSucceedsLocker(mc)}
	h.checkProvider(ctx, p)
}
