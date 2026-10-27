package channelhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_NewChannelHandler_cov2_fields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewChannelHandler(mockReq, mockNotify, mockDB, mockCache)
	h, ok := res.(*channelHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Expected util handler to be set")
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB || h.cache != mockCache {
		t.Errorf("Wrong dependency wiring")
	}
}

// cov2RunGetFail runs the given operation against a channel that can not be found.
// the parent deadline is shorter than defaultExistTimeout, so Get gives up quickly.
// It verifies the operation fails with the expected message and sends no request.
func cov2RunGetFail(t *testing.T, id string, run func(h *channelHandler, ctx context.Context) error, expectMsg string) {
	h, m := newCovHandler(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

	err := run(h, ctx)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), expectMsg) {
		t.Errorf("Wrong match. expect message contains: %s, got: %v", expectMsg, err)
	}

	checkNoGetWithTimeoutGoroutine(t)
}

const (
	cov2ChannelID  = "c2c00000-0000-4000-8000-000000000001"
	cov2AsteriskID = "00:11:22:33:44:55"
)
