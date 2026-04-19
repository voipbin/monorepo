package healthcheckhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
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

	h := &healthCheckHandler{db: mockDB, reqHandler: mockReq}
	// Should not panic or propagate error
	h.checkProvider(ctx, p)
}
