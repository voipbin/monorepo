package outdialhandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

func Test_NewOutdialHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h, ok := NewOutdialHandler(mockDB, mockReq, mockNotify).(*outdialHandler)
	if !ok {
		t.Fatalf("expected an *outdialHandler")
	}

	// each dependency is checked on its own field, and they are distinct mock instances
	if h.db != dbhandler.DBHandler(mockDB) {
		t.Errorf("the db handler was not stored")
	}
	if h.reqHandler != requesthandler.RequestHandler(mockReq) {
		t.Errorf("the request handler was not stored")
	}
	if h.notifyHandler != notifyhandler.NotifyHandler(mockNotify) {
		t.Errorf("the notify handler was not stored")
	}
}
