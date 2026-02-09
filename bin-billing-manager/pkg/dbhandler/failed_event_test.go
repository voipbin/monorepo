package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

func Test_FailedEventCreate(t *testing.T) {

	type test struct {
		name string

		failedEvent *failedevent.FailedEvent

		responseCurTime *time.Time
		expectRes       *failedevent.FailedEvent
	}

	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	nextRetry := time.Date(2023, 6, 7, 3, 23, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "have all fields",

			failedEvent: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f0000001-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"id":"test-event-id"}`,
				ErrorMessage:   "processing failed",
				RetryCount:     0,
				MaxRetries:     5,
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
			},

			responseCurTime: &tmCreate,
			expectRes: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f0000001-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"id":"test-event-id"}`,
				ErrorMessage:   "processing failed",
				RetryCount:     0,
				MaxRetries:     5,
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
				TMCreate:       &tmCreate,
				TMUpdate:       nil,
			},
		},
		{
			name: "minimal fields",

			failedEvent: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f0000002-0001-0001-0001-000000000001"),
				EventType:      "message_created",
				EventPublisher: "message-manager",
				EventData:      `{}`,
				ErrorMessage:   "err",
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
			},

			responseCurTime: &tmCreate,
			expectRes: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f0000002-0001-0001-0001-000000000001"),
				EventType:      "message_created",
				EventPublisher: "message-manager",
				EventData:      `{}`,
				ErrorMessage:   "err",
				RetryCount:     0,
				MaxRetries:     0,
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
				TMCreate:       &tmCreate,
				TMUpdate:       nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.FailedEventCreate(ctx, tt.failedEvent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify by listing pending events with a future time
			futureTime := tt.responseCurTime.Add(24 * time.Hour)
			res, err := h.FailedEventListPendingRetry(ctx, futureTime)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// find our event in the results
			var found *failedevent.FailedEvent
			for _, fe := range res {
				if fe.ID == tt.failedEvent.ID {
					found = fe
					break
				}
			}

			if found == nil {
				t.Fatalf("Created failed event not found in list results")
			}

			if !reflect.DeepEqual(tt.expectRes, found) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, found)
			}

			// cleanup
			if err := h.FailedEventDelete(ctx, tt.failedEvent.ID); err != nil {
				t.Errorf("Could not clean up failed event: %v", err)
			}
		})
	}
}

func Test_FailedEventListPendingRetry(t *testing.T) {

	type test struct {
		name string

		failedEvents []*failedevent.FailedEvent

		queryTime time.Time

		responseCurTime *time.Time
		expectCount     int
	}

	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	pastRetry := time.Date(2023, 6, 7, 3, 20, 0, 0, time.UTC)
	futureRetry := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	queryTime := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "returns only pending events with past next_retry_at",

			failedEvents: []*failedevent.FailedEvent{
				{
					// should be returned: pending + past retry time
					ID:             uuid.FromStringOrNil("f1000001-0002-0001-0001-000000000001"),
					EventType:      "call_progressing",
					EventPublisher: "call-manager",
					EventData:      `{"id":"1"}`,
					ErrorMessage:   "err",
					RetryCount:     0,
					MaxRetries:     5,
					NextRetryAt:    &pastRetry,
					Status:         failedevent.StatusPending,
				},
				{
					// should be returned: retrying + past retry time
					ID:             uuid.FromStringOrNil("f1000002-0002-0001-0001-000000000001"),
					EventType:      "call_hangup",
					EventPublisher: "call-manager",
					EventData:      `{"id":"2"}`,
					ErrorMessage:   "err",
					RetryCount:     1,
					MaxRetries:     5,
					NextRetryAt:    &pastRetry,
					Status:         failedevent.StatusRetrying,
				},
				{
					// should NOT be returned: exhausted status
					ID:             uuid.FromStringOrNil("f1000003-0002-0001-0001-000000000001"),
					EventType:      "message_created",
					EventPublisher: "message-manager",
					EventData:      `{"id":"3"}`,
					ErrorMessage:   "err",
					RetryCount:     5,
					MaxRetries:     5,
					NextRetryAt:    &pastRetry,
					Status:         failedevent.StatusExhausted,
				},
				{
					// should NOT be returned: pending but future retry time
					ID:             uuid.FromStringOrNil("f1000004-0002-0001-0001-000000000001"),
					EventType:      "number_created",
					EventPublisher: "number-manager",
					EventData:      `{"id":"4"}`,
					ErrorMessage:   "err",
					RetryCount:     0,
					MaxRetries:     5,
					NextRetryAt:    &futureRetry,
					Status:         failedevent.StatusPending,
				},
			},

			queryTime: queryTime,

			responseCurTime: &tmCreate,
			expectCount:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			// create all test events
			for range tt.failedEvents {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			}
			for _, fe := range tt.failedEvents {
				if err := h.FailedEventCreate(ctx, fe); err != nil {
					t.Fatalf("Could not create failed event: %v", err)
				}
			}

			// query
			res, err := h.FailedEventListPendingRetry(ctx, tt.queryTime)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// count only the events we created (filter by our test IDs)
			count := 0
			for _, fe := range res {
				for _, created := range tt.failedEvents {
					if fe.ID == created.ID {
						count++
						break
					}
				}
			}

			if count != tt.expectCount {
				t.Errorf("Wrong match. expect count: %d, got: %d", tt.expectCount, count)
			}

			// cleanup
			for _, fe := range tt.failedEvents {
				_ = h.FailedEventDelete(ctx, fe.ID)
			}
		})
	}
}

func Test_FailedEventUpdate(t *testing.T) {

	type test struct {
		name string

		failedEvent *failedevent.FailedEvent
		fields      map[failedevent.Field]any

		responseCurTime *time.Time
		expectStatus    failedevent.Status
		expectRetry     int
	}

	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	nextRetry := time.Date(2023, 6, 7, 3, 20, 0, 0, time.UTC)

	tests := []test{
		{
			name: "update status and retry count",

			failedEvent: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f2000001-0003-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"id":"1"}`,
				ErrorMessage:   "err",
				RetryCount:     0,
				MaxRetries:     5,
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
			},

			fields: map[failedevent.Field]any{
				failedevent.FieldStatus:     failedevent.StatusRetrying,
				failedevent.FieldRetryCount: 1,
			},

			responseCurTime: &tmCreate,
			expectStatus:    failedevent.StatusRetrying,
			expectRetry:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			// create
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.FailedEventCreate(ctx, tt.failedEvent); err != nil {
				t.Fatalf("Could not create failed event: %v", err)
			}

			// update
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.FailedEventUpdate(ctx, tt.failedEvent.ID, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify by listing
			futureTime := tt.responseCurTime.Add(24 * time.Hour)
			res, err := h.FailedEventListPendingRetry(ctx, futureTime)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var found *failedevent.FailedEvent
			for _, fe := range res {
				if fe.ID == tt.failedEvent.ID {
					found = fe
					break
				}
			}

			if found == nil {
				t.Fatalf("Updated failed event not found in list results")
			}

			if found.Status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %s, got: %s", tt.expectStatus, found.Status)
			}

			if found.RetryCount != tt.expectRetry {
				t.Errorf("Wrong retry count. expect: %d, got: %d", tt.expectRetry, found.RetryCount)
			}

			if found.TMUpdate == nil {
				t.Errorf("TMUpdate should be set after update")
			}

			// cleanup
			_ = h.FailedEventDelete(ctx, tt.failedEvent.ID)
		})
	}
}

func Test_FailedEventDelete(t *testing.T) {

	type test struct {
		name string

		failedEvent *failedevent.FailedEvent

		responseCurTime *time.Time
	}

	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	nextRetry := time.Date(2023, 6, 7, 3, 20, 0, 0, time.UTC)

	tests := []test{
		{
			name: "delete removes event",

			failedEvent: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("f3000001-0004-0001-0001-000000000001"),
				EventType:      "call_hangup",
				EventPublisher: "call-manager",
				EventData:      `{"id":"1"}`,
				ErrorMessage:   "err",
				RetryCount:     0,
				MaxRetries:     5,
				NextRetryAt:    &nextRetry,
				Status:         failedevent.StatusPending,
			},

			responseCurTime: &tmCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			// create
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.FailedEventCreate(ctx, tt.failedEvent); err != nil {
				t.Fatalf("Could not create failed event: %v", err)
			}

			// delete
			if err := h.FailedEventDelete(ctx, tt.failedEvent.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify it's gone
			futureTime := tt.responseCurTime.Add(24 * time.Hour)
			res, err := h.FailedEventListPendingRetry(ctx, futureTime)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, fe := range res {
				if fe.ID == tt.failedEvent.ID {
					t.Errorf("Deleted failed event should not appear in list results")
				}
			}
		})
	}
}

func Test_FailedEventUpdate_to_exhausted_hides_from_list(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	pastRetry := time.Date(2023, 6, 7, 3, 20, 0, 0, time.UTC)

	fe := &failedevent.FailedEvent{
		ID:             uuid.FromStringOrNil("f4000001-0005-0001-0001-000000000001"),
		EventType:      "call_progressing",
		EventPublisher: "call-manager",
		EventData:      `{"id":"1"}`,
		ErrorMessage:   "err",
		RetryCount:     4,
		MaxRetries:     5,
		NextRetryAt:    &pastRetry,
		Status:         failedevent.StatusRetrying,
	}

	// create
	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	if err := h.FailedEventCreate(ctx, fe); err != nil {
		t.Fatalf("Could not create failed event: %v", err)
	}

	// mark as exhausted
	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	if err := h.FailedEventUpdate(ctx, fe.ID, map[failedevent.Field]any{
		failedevent.FieldStatus: failedevent.StatusExhausted,
	}); err != nil {
		t.Fatalf("Could not update failed event: %v", err)
	}

	// verify it no longer appears in pending retry list
	futureTime := tmCreate.Add(24 * time.Hour)
	res, err := h.FailedEventListPendingRetry(ctx, futureTime)
	if err != nil {
		t.Fatalf("Could not list: %v", err)
	}

	for _, r := range res {
		if r.ID == fe.ID {
			t.Errorf("Exhausted event should not appear in pending retry list")
		}
	}

	// cleanup
	_ = h.FailedEventDelete(ctx, fe.ID)
}
