package agenthandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_NewAgentHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res, ok := NewAgentHandler(mockReq, mockDB, mockNotify, mockCache).(*agentHandler)
	if !ok {
		t.Fatalf("Wrong type. expect: *agentHandler")
	}

	if res.reqHandler != mockReq {
		t.Errorf("Wrong reqHandler wiring.")
	}
	if res.db != mockDB {
		t.Errorf("Wrong db wiring.")
	}
	if res.notifyHandler != mockNotify {
		t.Errorf("Wrong notifyHandler wiring.")
	}
	if res.cache != mockCache {
		t.Errorf("Wrong cache wiring.")
	}
	if res.utilHandler == nil {
		t.Errorf("Wrong utilHandler wiring. expect: not nil")
	}
}
