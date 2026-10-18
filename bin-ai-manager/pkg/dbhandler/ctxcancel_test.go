package dbhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// Test_ContextCancel_QueryAborted verifies that the request context reaches the
// database driver: a cancelled context aborts the query (read, write and list
// paths) instead of the query running to completion.
func Test_ContextCancel_QueryAborted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"read via private get (aicall)", func() error { _, err := h.aicallGetFromDB(ctx, id); return err }},
		{"read via private get (summary)", func() error { _, err := h.summaryGetFromDB(ctx, id); return err }},
		{"write (team delete)", func() error { return h.TeamDelete(ctx, id) }},
		{"list (summary)", func() error { _, err := h.SummaryList(ctx, 10, "", nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
			mockUtil.EXPECT().TimeGetCurTime().Return("").AnyTimes()
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}
		})
	}
}
