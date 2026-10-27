package accounthandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
)

func Test_NewAccountHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockPaddle := paddlehandler.NewMockPaddleHandler(mc)

	res := NewAccountHandler(mockReq, mockDB, mockNotify, mockPaddle)
	h, ok := res.(*accountHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *accountHandler, got: %T", res)
	}

	if h.utilHandler == nil || h.reqHandler != mockReq || h.db != mockDB || h.notifyHandler != mockNotify || h.paddleHandler != mockPaddle {
		t.Errorf("Wrong match. every dependency must be wired. got: %+v", h)
	}
}
