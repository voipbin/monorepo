package activeflowhandler

import (
	"context"
	"fmt"
	"reflect"
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

func Test_Get_cacheReadError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := NewActiveflowHandler(mockCache, mockReq)

	ctx := context.Background()
	activeflowID := uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001")
	if activeflowID == uuid.Nil {
		t.Fatalf("invalid test uuid")
	}
	tmUpdate := time.Date(2026, 6, 10, 2, 0, 0, 0, time.UTC)

	// a cache read error is treated as a miss and the RPC fallback resolves the destination.
	mockCache.EXPECT().ActiveflowWebhookGet(ctx, activeflowID).Return(nil, false, errors.New("cache down"))
	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, activeflowID).Return(&fmactiveflow.Activeflow{
		Identity:      commonidentity.Identity{ID: activeflowID},
		WebhookURI:    "af.test.com",
		WebhookMethod: fmactiveflow.WebhookMethodPost,
		TMUpdate:      &tmUpdate,
	}, nil).Times(1)
	mockCache.EXPECT().ActiveflowWebhookSet(ctx, activeflowID, &mwactiveflow.Webhook{
		URI:    "af.test.com",
		Method: webhook.MethodTypePOST,
		Tm:     tmUpdate,
	}, 24*time.Hour).Return(nil).Times(1)
	mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	dest, err := h.Get(ctx, activeflowID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if dest == nil || dest.URI != "af.test.com" || dest.Method != webhook.MethodTypePOST {
		t.Errorf("Wrong match. expect: af.test.com/POST, got: %v", dest)
	}
}

func Test_Get_cacheWriteErrorsIgnored(t *testing.T) {
	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow
		responseErr        error

		expectSetPositive *mwactiveflow.Webhook

		expectSetNegative bool
		// expectNegTm nil means the current time (not found row only, checked as a range).
		expectNegTm       *time.Time
		expectNegTMDelete *time.Time
		expectNegTTL      time.Duration

		expectDest *Destination
	}{
		{
			name:         "not found: transient negative write failure is ignored",
			activeflowID: uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000001"),

			responseErr: requesthandler.ErrNotFound,

			expectSetNegative: true,
			expectNegTTL:      5 * time.Second,
		},
		{
			name:         "soft-deleted: negative write failure is ignored",
			activeflowID: uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000002"),

			responseActiveflow: &fmactiveflow.Activeflow{
				WebhookURI: "af.test.com",
				TMCreate:   &tmCreate,
				TMDelete:   &tmDelete,
			},

			expectSetNegative: true,
			expectNegTm:       &tmCreate,
			expectNegTMDelete: &tmDelete,
			expectNegTTL:      10 * time.Minute,
		},
		{
			name:         "empty uri: negative write failure is ignored",
			activeflowID: uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000003"),

			responseActiveflow: &fmactiveflow.Activeflow{
				TMCreate: &tmCreate,
			},

			expectSetNegative: true,
			expectNegTm:       &tmCreate,
			expectNegTTL:      10 * time.Minute,
		},
		{
			name:         "positive: write failure is ignored and the destination is still returned",
			activeflowID: uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000004"),

			responseActiveflow: &fmactiveflow.Activeflow{
				WebhookURI:    "af.test.com",
				WebhookMethod: fmactiveflow.WebhookMethodPost,
				TMCreate:      &tmCreate,
			},

			expectSetPositive: &mwactiveflow.Webhook{URI: "af.test.com", Method: webhook.MethodTypePOST, Tm: tmCreate},
			expectDest:        &Destination{URI: "af.test.com", Method: webhook.MethodTypePOST},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if tt.activeflowID == uuid.Nil {
				t.Fatalf("invalid test uuid")
			}

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := NewActiveflowHandler(mockCache, mockReq)

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowWebhookGet(ctx, tt.activeflowID).Return(nil, false, nil)
			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, tt.responseErr).Times(1)

			before := time.Now()
			if tt.expectSetPositive != nil {
				mockCache.EXPECT().ActiveflowWebhookSet(ctx, tt.activeflowID, tt.expectSetPositive, 24*time.Hour).Return(errors.New("write failed")).Times(1)
			}
			if tt.expectSetNegative {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(ctx, tt.activeflowID, gomock.Any(), tt.expectNegTMDelete, tt.expectNegTTL).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, tm time.Time, _ *time.Time, _ time.Duration) error {
						if tt.expectNegTm != nil {
							if !tm.Equal(*tt.expectNegTm) {
								t.Errorf("Wrong match. expect tm: %v, got: %v", *tt.expectNegTm, tm)
							}
						} else if tm.Before(before) || tm.After(time.Now()) {
							t.Errorf("Wrong match. expect the current time, got: %v", tm)
						}
						return errors.New("write failed")
					}).Times(1)
			}

			dest, err := h.Get(ctx, tt.activeflowID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(dest, tt.expectDest) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectDest, dest)
			}
		})
	}
}

func Test_Get_fallbackTimestamp(t *testing.T) {
	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 6, 10, 5, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		activeflowID uuid.UUID

		tmCreate *time.Time
		tmUpdate *time.Time

		// expectTm is the exact cached timestamp; nil means "the current time" (checked as a range).
		expectTm *time.Time
	}{
		{
			name:         "update wins over create",
			activeflowID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001"),
			tmCreate:     &tmCreate,
			tmUpdate:     &tmUpdate,
			expectTm:     &tmUpdate,
		},
		{
			name:         "create is used when there is no update",
			activeflowID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000002"),
			tmCreate:     &tmCreate,
			expectTm:     &tmCreate,
		},
		{
			name:         "current time is used when neither exists",
			activeflowID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000003"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if tt.activeflowID == uuid.Nil {
				t.Fatalf("invalid test uuid")
			}

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := NewActiveflowHandler(mockCache, mockReq)

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowWebhookGet(ctx, tt.activeflowID).Return(nil, false, nil)
			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(&fmactiveflow.Activeflow{
				WebhookURI:    "af.test.com",
				WebhookMethod: fmactiveflow.WebhookMethodPost,
				TMCreate:      tt.tmCreate,
				TMUpdate:      tt.tmUpdate,
			}, nil)

			var gotTm time.Time
			before := time.Now()
			mockCache.EXPECT().ActiveflowWebhookSet(ctx, tt.activeflowID, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, w *mwactiveflow.Webhook, _ time.Duration) error {
					gotTm = w.Tm
					return nil
				}).Times(1)

			if _, err := h.Get(ctx, tt.activeflowID); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			after := time.Now()

			if tt.expectTm != nil {
				if !gotTm.Equal(*tt.expectTm) {
					t.Errorf("Wrong match. expect: %v, got: %v", *tt.expectTm, gotTm)
				}
				return
			}
			if gotTm.Before(before) || gotTm.After(after) {
				t.Errorf("Wrong match. expect a time between %v and %v, got: %v", before, after, gotTm)
			}
		})
	}
}
