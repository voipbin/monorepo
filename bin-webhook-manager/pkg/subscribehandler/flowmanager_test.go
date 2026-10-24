package subscribehandler

import (
	"errors"
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

func Test_processEventFMActiveflowCreatedUpdated_invalidJSON(t *testing.T) {
	tests := []struct {
		name string

		eventType string
		run       func(h *subscribeHandler, m *sock.Event) error
	}{
		{
			name:      "created/updated handler rejects malformed data",
			eventType: fmactiveflow.EventTypeActiveflowCreated,
			run:       func(h *subscribeHandler, m *sock.Event) error { return h.processEventFMActiveflowCreatedUpdated(m) },
		},
		{
			name:      "deleted handler rejects malformed data",
			eventType: fmactiveflow.EventTypeActiveflowDeleted,
			run:       func(h *subscribeHandler, m *sock.Event) error { return h.processEventFMActiveflowDeleted(m) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the cache mock has no expectation, so any cache call fails the test.
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
				cacheHandler:   mockCache,
			}

			event := &sock.Event{
				Publisher: publisherFlowManager,
				Type:      tt.eventType,
				DataType:  "application/json",
				Data:      []byte(`{"id": not-json`),
			}

			if err := tt.run(h, event); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_processEventFMActiveflow_cacheErrors(t *testing.T) {
	errCache := errors.New("cache write failed")

	tests := []struct {
		name string

		data      string
		eventType string
		run       func(h *subscribeHandler, m *sock.Event) error

		expectPositive *mwactiveflow.Webhook
		expectTm       time.Time
		expectTMDelete *time.Time
		expectTTL      time.Duration
	}{
		{
			name:           "positive write failure is propagated",
			data:           `{"id":"d1111111-1111-1111-1111-111111111111","webhook_uri":"https://example.com/hook","webhook_method":"POST","tm_create":"2026-06-10T00:00:00Z"}`,
			eventType:      fmactiveflow.EventTypeActiveflowCreated,
			run:            func(h *subscribeHandler, m *sock.Event) error { return h.processEventFMActiveflowCreatedUpdated(m) },
			expectPositive: &mwactiveflow.Webhook{URI: "https://example.com/hook", Method: webhook.MethodTypePOST, Tm: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)},
		},
		{
			name:      "negative write failure is propagated",
			data:      `{"id":"d1111111-1111-1111-1111-111111111111","tm_create":"2026-06-10T00:00:00Z"}`,
			eventType: fmactiveflow.EventTypeActiveflowUpdated,
			run:       func(h *subscribeHandler, m *sock.Event) error { return h.processEventFMActiveflowCreatedUpdated(m) },
			expectTm:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
			expectTTL: 10 * time.Minute,
		},
		{
			name:      "delete tombstone write failure is propagated",
			data:      `{"id":"d1111111-1111-1111-1111-111111111111","tm_create":"2026-06-10T00:00:00Z","tm_delete":"2026-06-10T01:00:00Z"}`,
			eventType: fmactiveflow.EventTypeActiveflowDeleted,
			run:       func(h *subscribeHandler, m *sock.Event) error { return h.processEventFMActiveflowDeleted(m) },
			expectTm:  time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC),
			expectTMDelete: func() *time.Time {
				tm := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)
				return &tm
			}(),
			expectTTL: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
				cacheHandler:   mockCache,
			}

			activeflowID := uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111")
			if activeflowID == uuid.Nil {
				t.Fatalf("invalid test uuid")
			}

			if tt.expectPositive != nil {
				mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), activeflowID, tt.expectPositive, 24*time.Hour).Return(errCache).Times(1)
			} else {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), activeflowID, tt.expectTm, tt.expectTMDelete, tt.expectTTL).Return(errCache).Times(1)
			}

			event := &sock.Event{
				Publisher: publisherFlowManager,
				Type:      tt.eventType,
				DataType:  "application/json",
				Data:      []byte(tt.data),
			}

			if err := tt.run(h, event); !errors.Is(err, errCache) {
				t.Errorf("Wrong match. expect: %v, got: %v", errCache, err)
			}
		})
	}
}

func Test_activeflowEventTimestamp(t *testing.T) {
	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 6, 10, 5, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		message *fmactiveflow.WebhookMessage

		// expect nil means the current time (checked as a range).
		expect *time.Time
	}{
		{
			name:    "delete wins over update and create",
			message: &fmactiveflow.WebhookMessage{TMCreate: &tmCreate, TMUpdate: &tmUpdate, TMDelete: &tmDelete},
			expect:  &tmDelete,
		},
		{
			name:    "update wins over create",
			message: &fmactiveflow.WebhookMessage{TMCreate: &tmCreate, TMUpdate: &tmUpdate},
			expect:  &tmUpdate,
		},
		{
			name:    "create is used alone",
			message: &fmactiveflow.WebhookMessage{TMCreate: &tmCreate},
			expect:  &tmCreate,
		},
		{
			name:    "current time when no timestamp exists",
			message: &fmactiveflow.WebhookMessage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			got := activeflowEventTimestamp(tt.message)
			after := time.Now()

			if tt.expect != nil {
				if !got.Equal(*tt.expect) {
					t.Errorf("Wrong match. expect: %v, got: %v", *tt.expect, got)
				}
				return
			}
			if got.Before(before) || got.After(after) {
				t.Errorf("Wrong match. expect a time between %v and %v, got: %v", before, after, got)
			}
		})
	}
}
