package customerhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"go.uber.org/mock/gomock"
)

func TestNewCustomerHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewCustomerHandler(mockReq, mockDB, mockCache, mockNotify)
	if h == nil {
		t.Error("NewCustomerHandler returned nil")
	}
}
