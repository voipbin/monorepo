package subscribehandler

import (
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

func Test_processEventFMActiveflowCreatedUpdated(t *testing.T) {

	tests := []struct {
		name string

		event *sock.Event

		expectID uuid.UUID
		// expectPositive true means ActiveflowWebhookSet expected, otherwise ActiveflowWebhookSetNegative.
		expectPositive bool
	}{
		{
			name: "uri set caches positive",

			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"a1111111-1111-1111-1111-111111111111","webhook_uri":"af.test.com","webhook_method":"POST","tm_create":"2026-06-10T00:00:00Z"}`),
			},

			expectID:       uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111"),
			expectPositive: true,
		},
		{
			name: "empty uri caches negative",

			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowUpdated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"a2222222-2222-2222-2222-222222222222","webhook_uri":"","tm_create":"2026-06-10T00:00:00Z"}`),
			},

			expectID:       uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222"),
			expectPositive: false,
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
				mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), tt.expectID, gomock.Any(), gomock.Any()).Return(nil)
			} else {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), tt.expectID, gomock.Any(), nil, gomock.Any()).Return(nil)
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

	event := &sock.Event{
		Publisher: publisherFlowManager,
		Type:      fmactiveflow.EventTypeActiveflowDeleted,
		DataType:  "application/json",
		Data:      []byte(`{"id":"b1111111-1111-1111-1111-111111111111","webhook_uri":"af.test.com","tm_create":"2026-06-10T00:00:00Z","tm_delete":"2026-06-10T01:00:00Z"}`),
	}

	// delete writes a negative tombstone carrying the delete timestamp.
	mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), activeflowID, tmDelete, &tmDelete, gomock.Any()).Return(nil)

	if err := h.processEventFMActiveflowDeleted(event); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
