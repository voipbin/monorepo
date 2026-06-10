package activeflowhandler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

func Test_Get_cacheHit(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID

		responseEntry *mwactiveflow.Webhook

		expectDest *Destination
	}{
		{
			name: "cache hit positive returns destination without fallback",

			activeflowID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),

			responseEntry: &mwactiveflow.Webhook{
				URI:    "af.test.com",
				Method: webhook.MethodTypePOST,
				Tm:     time.Now(),
			},

			expectDest: &Destination{URI: "af.test.com", Method: webhook.MethodTypePOST},
		},
		{
			name: "cache hit negative returns nil without fallback",

			activeflowID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),

			responseEntry: &mwactiveflow.Webhook{
				Deleted: true,
				Tm:      time.Now(),
			},

			expectDest: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := NewActiveflowHandler(mockCache, mockReq)

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowWebhookGet(ctx, tt.activeflowID).Return(tt.responseEntry, true, nil)
			// no fallback expected: FlowV1ActiveflowGet must NOT be called.
			mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), gomock.Any()).Times(0)

			dest, err := h.Get(ctx, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if (dest == nil) != (tt.expectDest == nil) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectDest, dest)
			}
			if dest != nil && (dest.URI != tt.expectDest.URI || dest.Method != tt.expectDest.Method) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectDest, dest)
			}
		})
	}
}

func Test_Get_missFallback(t *testing.T) {

	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow
		responseErr        error

		// expectations
		expectSetPositive bool
		expectSetNegative bool
		expectDest        *Destination
	}{
		{
			name: "miss then fallback with uri set caches positive and returns destination",

			activeflowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
				WebhookURI:    "af.test.com",
				WebhookMethod: fmactiveflow.WebhookMethodPost,
				TMCreate:      &tmCreate,
			},

			expectSetPositive: true,
			expectDest:        &Destination{URI: "af.test.com", Method: webhook.MethodTypePOST},
		},
		{
			name: "miss then fallback with empty uri caches negative and returns nil",

			activeflowID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				},
				WebhookURI: "",
				TMCreate:   &tmCreate,
			},

			expectSetNegative: true,
			expectDest:        nil,
		},
		{
			name: "miss then fallback with tm_delete caches negative (never positive) and returns nil",

			activeflowID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				},
				WebhookURI: "af.test.com",
				TMCreate:   &tmCreate,
				TMDelete:   &tmDelete,
			},

			expectSetNegative: true,
			expectDest:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := NewActiveflowHandler(mockCache, mockReq)

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowWebhookGet(ctx, tt.activeflowID).Return(nil, false, nil)
			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, tt.responseErr)

			if tt.expectSetPositive {
				mockCache.EXPECT().ActiveflowWebhookSet(ctx, tt.activeflowID, gomock.Any(), gomock.Any()).Return(nil)
			}
			if tt.expectSetNegative {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(ctx, tt.activeflowID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			}

			dest, err := h.Get(ctx, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if (dest == nil) != (tt.expectDest == nil) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectDest, dest)
			}
			if dest != nil && (dest.URI != tt.expectDest.URI || dest.Method != tt.expectDest.Method) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectDest, dest)
			}
		})
	}
}

func Test_Get_missFallback_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	activeflowID := uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := NewActiveflowHandler(mockCache, mockReq)

	ctx := context.Background()

	mockCache.EXPECT().ActiveflowWebhookGet(ctx, activeflowID).Return(nil, false, nil)
	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(nil, errors.Wrap(requesthandler.ErrNotFound, "wrapped"))
	// NotFound is treated as transient: a short negative is written.
	mockCache.EXPECT().ActiveflowWebhookSetNegative(ctx, activeflowID, gomock.Any(), nil, gomock.Any()).Return(nil)

	dest, err := h.Get(ctx, activeflowID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if dest != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", dest)
	}
}

func Test_Get_missFallback_rpcError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	activeflowID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := NewActiveflowHandler(mockCache, mockReq)

	ctx := context.Background()

	mockCache.EXPECT().ActiveflowWebhookGet(ctx, activeflowID).Return(nil, false, nil)
	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(nil, fmt.Errorf("rpc transport error"))
	// generic rpc error: no cache set at all.
	mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	dest, err := h.Get(ctx, activeflowID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if dest != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", dest)
	}
}

// Test_Get_singleflight asserts that N concurrent Get() calls for the same id
// on a cache miss coalesce into a single FlowV1ActiveflowGet fallback (design
// 10). The fallback mock blocks until all goroutines have entered Get so the
// singleflight window is actually exercised, and a counter verifies the RPC is
// invoked exactly once.
func Test_Get_singleflight(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	const n = 20

	activeflowID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")
	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := NewActiveflowHandler(mockCache, mockReq)

	ctx := context.Background()

	// every lookup is a miss so all callers funnel into the fallback.
	mockCache.EXPECT().ActiveflowWebhookGet(gomock.Any(), activeflowID).Return(nil, false, nil).AnyTimes()

	var callCount int32
	var release sync.WaitGroup
	release.Add(1)
	// FlowV1ActiveflowGet must be called exactly once. It blocks on the release
	// gate so the concurrent callers overlap inside the singleflight window.
	mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), activeflowID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*fmactiveflow.Activeflow, error) {
			atomic.AddInt32(&callCount, 1)
			release.Wait()
			return &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: activeflowID,
				},
				WebhookURI:    "af.test.com",
				WebhookMethod: fmactiveflow.WebhookMethodPost,
				TMCreate:      &tmCreate,
			}, nil
		},
	).Times(1)
	// the single fallback caches a positive entry.
	mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), activeflowID, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	var start sync.WaitGroup
	start.Add(n)

	var done sync.WaitGroup
	done.Add(n)

	results := make([]*Destination, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer done.Done()
			start.Done()
			dest, err := h.Get(ctx, activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			results[idx] = dest
		}(i)
	}

	// wait for all goroutines to be in flight, then let the fallback return.
	start.Wait()
	time.Sleep(50 * time.Millisecond)
	release.Done()

	done.Wait()

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("Wrong match. expect FlowV1ActiveflowGet called once, got: %d", got)
	}
	for i, dest := range results {
		if dest == nil || dest.URI != "af.test.com" {
			t.Errorf("Wrong match. idx: %d, expect: af.test.com, got: %v", i, dest)
		}
	}
}
