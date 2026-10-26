package outplanhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_NewOutplanHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewOutplanHandler(mockDB, mockReq, mockNotify)

	h, ok := res.(*outplanHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *outplanHandler, got: %T", res)
	}
	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Errorf("Wrong match. the given dependencies must be kept. handler: %+v", h)
	}
	if h.util == nil {
		t.Errorf("Wrong match. expect: util handler is set, got: nil")
	}
}
