package subscribehandler

import (
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// Test_processEventFMActiveflowCreatedUpdated asserts that lifecycle
// created/updated events pre-populate the cache from the event payload
// (Option A): a positive entry when webhook_uri is present, a negative entry
// when it is empty (design 5.2 / 5.6).
func Test_processEventFMActiveflowCreatedUpdated(t *testing.T) {

	tests := []struct {
		name string

		event *sock.Event

		expectActiveflowID uuid.UUID
		expectTm           time.Time
		expectPositive     bool
		expectURI          string
		expectMethod       webhook.MethodType
	}{
		{
			name: "created event with webhook_uri sets a positive entry",

			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"a1111111-1111-1111-1111-111111111111","webhook_uri":"https://example.com/hook","webhook_method":"POST","tm_create":"2026-06-10T00:00:00Z"}`),
			},

			expectActiveflowID: uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
			expectTm:           time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
			expectPositive:     true,
			expectURI:          "https://example.com/hook",
			expectMethod:       webhook.MethodTypePOST,
		},
		{
			name: "updated event without webhook_uri sets a negative entry",

			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowUpdated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"a2222222-2222-2222-2222-222222222222","tm_create":"2026-06-10T00:00:00Z","tm_update":"2026-06-10T00:30:00Z"}`),
			},

			expectActiveflowID: uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			expectTm:           time.Date(2026, 6, 10, 0, 30, 0, 0, time.UTC),
			expectPositive:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &subscribeHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				cacheHandler:   mockCache,
			}

			if tt.expectPositive {
				expectEntry := &mwactiveflow.Webhook{
					URI:    tt.expectURI,
					Method: tt.expectMethod,
					Tm:     tt.expectTm,
				}
				mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), tt.expectActiveflowID, expectEntry, subTLive).Return(nil)
			} else {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), tt.expectActiveflowID, tt.expectTm, nil, subTNeg).Return(nil)
			}

			if err := h.processEventFMActiveflowCreatedUpdated(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventFMActiveflowDeleted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	tmDelete := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)
	activeflowID := uuid.FromStringOrNil("b1111111-1111-1111-1111-111111111111")

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		accountHandler: mockAccount,
		cacheHandler:   mockCache,
	}

	// the deleted event carries id + tm_delete.
	event := &sock.Event{
		Publisher: publisherFlowManager,
		Type:      fmactiveflow.EventTypeActiveflowDeleted,
		DataType:  "application/json",
		Data:      []byte(`{"id":"b1111111-1111-1111-1111-111111111111","tm_create":"2026-06-10T00:00:00Z","tm_delete":"2026-06-10T01:00:00Z"}`),
	}

	// delete writes a negative tombstone carrying the delete timestamp.
	mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), activeflowID, tmDelete, &tmDelete, gomock.Any()).Return(nil)

	if err := h.processEventFMActiveflowDeleted(event); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
