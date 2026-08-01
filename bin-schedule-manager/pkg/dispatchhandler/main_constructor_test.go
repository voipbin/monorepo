package dispatchhandler

import (
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/pkg/cachehandler"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

func TestNewDispatchHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewDispatchHandler(mockDB, mockCache, mockReq, mockNotify, 10, 10)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func TestNewDispatchHandler_defaults(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	// zero-valued configuration falls back to the design §11 defaults
	h := NewDispatchHandler(mockDB, mockCache, mockReq, mockNotify, 0, 0)

	impl, ok := h.(*dispatchHandler)
	if !ok {
		t.Errorf("Expected *dispatchHandler, got %T", h)
		return
	}
	if impl.tickInterval != time.Duration(defaultTickIntervalSec)*time.Second {
		t.Errorf("Wrong match. expect: %v, got: %v", time.Duration(defaultTickIntervalSec)*time.Second, impl.tickInterval)
	}
	if cap(impl.sem) != defaultConcurrency {
		t.Errorf("Wrong match. expect: %v, got: %v", defaultConcurrency, cap(impl.sem))
	}
}
