package trunkhandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_NewTrunkHandler_wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewTrunkHandler(mockReq, mockDB, mockNotify)

	h, ok := res.(*trunkHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *trunkHandler, got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. the util handler must be created")
	}
	if h.reqHandler != mockReq {
		t.Errorf("Wrong match. the request handler is not the injected one")
	}
	if h.db != mockDB {
		t.Errorf("Wrong match. the database handler is not the injected one")
	}
	if h.notifyHandler != mockNotify {
		t.Errorf("Wrong match. the notify handler is not the injected one")
	}
}
