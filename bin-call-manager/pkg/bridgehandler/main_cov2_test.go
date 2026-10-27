package bridgehandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_NewBridgeHandler_cov2(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	res := NewBridgeHandler(mockReq, mockNotify, mockDB)

	h, ok := res.(*bridgeHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. utilHandler must be set")
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB {
		t.Errorf("Wrong match. handlers are not wired: %v", h)
	}
}
