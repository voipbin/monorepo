package externalmediahandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_NewExternalMediaHandler_cov2(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)
	mockBridge := bridgehandler.NewMockBridgeHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewExternalMediaHandler(mockReq, mockNotify, mockDB, mockChannel, mockBridge, mockCache, 8088)

	h, ok := res.(*externalMediaHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. utilHandler must be set")
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB || h.channelHandler != mockChannel || h.bridgeHandler != mockBridge || h.cache != mockCache {
		t.Errorf("Wrong match. handlers are not wired: %v", h)
	}
	if h.asteriskWSPort != 8088 {
		t.Errorf("Wrong match. expect: 8088, got: %d", h.asteriskWSPort)
	}
}
