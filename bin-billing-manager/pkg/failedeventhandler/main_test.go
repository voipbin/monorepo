package failedeventhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_Save(t *testing.T) {

	type test struct {
		name string

		event         *sock.Event
		processingErr error

		responseUUID    uuid.UUID
		responseDBError error

		expectError bool
	}

	tests := []test{
		{
			name: "success",

			event: &sock.Event{
				Type:      "call_progressing",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"test-id"}`),
			},
			processingErr: fmt.Errorf("processing failed"),

			responseUUID:    uuid.FromStringOrNil("aa000001-0001-0001-0001-000000000001"),
			responseDBError: nil,

			expectError: false,
		},
		{
			name: "db create error",

			event: &sock.Event{
				Type:      "call_hangup",
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      json.RawMessage(`{"id":"test-id"}`),
			},
			processingErr: fmt.Errorf("processing failed"),

			responseUUID:    uuid.FromStringOrNil("aa000002-0001-0001-0001-000000000001"),
			responseDBError: fmt.Errorf("db connection error"),

			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &failedEventHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				eventProcessor: nil, // not used in Save
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().FailedEventCreate(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, fe *failedevent.FailedEvent) error {
					// verify the struct was populated correctly
					if fe.ID != tt.responseUUID {
						t.Errorf("Wrong ID. expect: %s, got: %s", tt.responseUUID, fe.ID)
					}
					if fe.EventType != tt.event.Type {
						t.Errorf("Wrong EventType. expect: %s, got: %s", tt.event.Type, fe.EventType)
					}
					if fe.EventPublisher != tt.event.Publisher {
						t.Errorf("Wrong EventPublisher. expect: %s, got: %s", tt.event.Publisher, fe.EventPublisher)
					}
					if fe.ErrorMessage != tt.processingErr.Error() {
						t.Errorf("Wrong ErrorMessage. expect: %s, got: %s", tt.processingErr.Error(), fe.ErrorMessage)
					}
					if fe.RetryCount != 0 {
						t.Errorf("Wrong RetryCount. expect: 0, got: %d", fe.RetryCount)
					}
					if fe.MaxRetries != maxRetries {
						t.Errorf("Wrong MaxRetries. expect: %d, got: %d", maxRetries, fe.MaxRetries)
					}
					if fe.Status != failedevent.StatusPending {
						t.Errorf("Wrong Status. expect: %s, got: %s", failedevent.StatusPending, fe.Status)
					}
					if fe.NextRetryAt == nil {
						t.Errorf("NextRetryAt should not be nil")
					}
					// verify event data is valid JSON of the original event
					var restored sock.Event
					if err := json.Unmarshal([]byte(fe.EventData), &restored); err != nil {
						t.Errorf("EventData is not valid JSON: %v", err)
					}
					return tt.responseDBError
				},
			)

			err := h.Save(ctx, tt.event, tt.processingErr)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_RetryPending_success(t *testing.T) {

	tests := []struct {
		name string

		events []*failedevent.FailedEvent
	}{
		{
			name: "successful retry deletes event",
			events: []*failedevent.FailedEvent{
				{
					ID:             uuid.FromStringOrNil("bb000001-0001-0001-0001-000000000001"),
					EventType:      "call_progressing",
					EventPublisher: "call-manager",
					EventData:      `{"type":"call_progressing","publisher":"call-manager","data_type":"application/json","data":{"id":"test"}}`,
					ErrorMessage:   "processing failed",
					RetryCount:     0,
					MaxRetries:     5,
					Status:         failedevent.StatusPending,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			processorCalled := false
			processor := func(m *sock.Event) error {
				processorCalled = true
				return nil // success
			}

			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: processor,
			}
			ctx := context.Background()

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return(tt.events, nil)
			mockDB.EXPECT().FailedEventDelete(ctx, tt.events[0].ID).Return(nil)

			err := h.RetryPending(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !processorCalled {
				t.Errorf("Event processor was not called")
			}
		})
	}
}

func Test_RetryPending_failure_increments_retry(t *testing.T) {

	tests := []struct {
		name string

		event *failedevent.FailedEvent

		expectRetryCount int
		expectStatus     failedevent.Status
	}{
		{
			name: "first retry failure",
			event: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("bb000002-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"type":"call_progressing","publisher":"call-manager","data_type":"application/json"}`,
				ErrorMessage:   "err",
				RetryCount:     0,
				MaxRetries:     5,
				Status:         failedevent.StatusPending,
			},
			expectRetryCount: 1,
			expectStatus:     failedevent.StatusRetrying,
		},
		{
			name: "third retry failure",
			event: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("bb000003-0001-0001-0001-000000000001"),
				EventType:      "call_hangup",
				EventPublisher: "call-manager",
				EventData:      `{"type":"call_hangup","publisher":"call-manager","data_type":"application/json"}`,
				ErrorMessage:   "err",
				RetryCount:     2,
				MaxRetries:     5,
				Status:         failedevent.StatusRetrying,
			},
			expectRetryCount: 3,
			expectStatus:     failedevent.StatusRetrying,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			processor := func(m *sock.Event) error {
				return fmt.Errorf("still failing")
			}

			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: processor,
			}
			ctx := context.Background()

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return([]*failedevent.FailedEvent{tt.event}, nil)
			mockDB.EXPECT().FailedEventUpdate(ctx, tt.event.ID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[failedevent.Field]any) error {
					if fields[failedevent.FieldRetryCount] != tt.expectRetryCount {
						t.Errorf("Wrong retry count. expect: %d, got: %v", tt.expectRetryCount, fields[failedevent.FieldRetryCount])
					}
					if fields[failedevent.FieldStatus] != tt.expectStatus {
						t.Errorf("Wrong status. expect: %s, got: %v", tt.expectStatus, fields[failedevent.FieldStatus])
					}
					if fields[failedevent.FieldNextRetryAt] == nil {
						t.Errorf("NextRetryAt should be set")
					}
					return nil
				},
			)

			err := h.RetryPending(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_RetryPending_exhausted(t *testing.T) {

	tests := []struct {
		name  string
		event *failedevent.FailedEvent
	}{
		{
			name: "max retries reached marks exhausted",
			event: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("bb000004-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"type":"call_progressing","publisher":"call-manager","data_type":"application/json"}`,
				ErrorMessage:   "err",
				RetryCount:     4,
				MaxRetries:     5,
				Status:         failedevent.StatusRetrying,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			processor := func(m *sock.Event) error {
				return fmt.Errorf("still failing")
			}

			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: processor,
			}
			ctx := context.Background()

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return([]*failedevent.FailedEvent{tt.event}, nil)
			mockDB.EXPECT().FailedEventUpdate(ctx, tt.event.ID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[failedevent.Field]any) error {
					if fields[failedevent.FieldStatus] != failedevent.StatusExhausted {
						t.Errorf("Wrong status. expect: %s, got: %v", failedevent.StatusExhausted, fields[failedevent.FieldStatus])
					}
					return nil
				},
			)

			err := h.RetryPending(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_RetryPending_unmarshal_error_marks_exhausted(t *testing.T) {

	tests := []struct {
		name  string
		event *failedevent.FailedEvent
	}{
		{
			name: "invalid JSON marks event as exhausted",
			event: &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("bb000005-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{invalid json`,
				ErrorMessage:   "err",
				RetryCount:     0,
				MaxRetries:     5,
				Status:         failedevent.StatusPending,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			processorCalled := false
			processor := func(m *sock.Event) error {
				processorCalled = true
				return nil
			}

			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: processor,
			}
			ctx := context.Background()

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return([]*failedevent.FailedEvent{tt.event}, nil)
			mockDB.EXPECT().FailedEventUpdate(ctx, tt.event.ID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[failedevent.Field]any) error {
					if fields[failedevent.FieldStatus] != failedevent.StatusExhausted {
						t.Errorf("Wrong status. expect: %s, got: %v", failedevent.StatusExhausted, fields[failedevent.FieldStatus])
					}
					return nil
				},
			)

			err := h.RetryPending(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if processorCalled {
				t.Errorf("Event processor should not be called for invalid event data")
			}
		})
	}
}

func Test_RetryPending_empty_list(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	processorCalled := false
	processor := func(m *sock.Event) error {
		processorCalled = true
		return nil
	}

	h := &failedEventHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		db:             mockDB,
		eventProcessor: processor,
	}
	ctx := context.Background()

	mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return(nil, nil)

	err := h.RetryPending(ctx)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if processorCalled {
		t.Errorf("Event processor should not be called when no events")
	}
}

func Test_RetryPending_db_list_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &failedEventHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		db:             mockDB,
		eventProcessor: nil,
	}
	ctx := context.Background()

	mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return(nil, fmt.Errorf("db error"))

	err := h.RetryPending(ctx)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_RetryPending_backoff_calculation(t *testing.T) {

	tests := []struct {
		name       string
		retryCount int
		// expected backoff in minutes: 5^(retryCount+1)
		expectMinBackoffMinutes float64
	}{
		{name: "first retry", retryCount: 0, expectMinBackoffMinutes: 5},    // 5^1
		{name: "second retry", retryCount: 1, expectMinBackoffMinutes: 25},   // 5^2
		{name: "third retry", retryCount: 2, expectMinBackoffMinutes: 125},   // 5^3
		{name: "fourth retry", retryCount: 3, expectMinBackoffMinutes: 625},  // 5^4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			processor := func(m *sock.Event) error {
				return fmt.Errorf("still failing")
			}

			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: processor,
			}
			ctx := context.Background()

			event := &failedevent.FailedEvent{
				ID:             uuid.FromStringOrNil("bb000006-0001-0001-0001-000000000001"),
				EventType:      "call_progressing",
				EventPublisher: "call-manager",
				EventData:      `{"type":"call_progressing","publisher":"call-manager","data_type":"application/json"}`,
				ErrorMessage:   "err",
				RetryCount:     tt.retryCount,
				MaxRetries:     5,
				Status:         failedevent.StatusPending,
			}

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return([]*failedevent.FailedEvent{event}, nil)
			mockDB.EXPECT().FailedEventUpdate(ctx, event.ID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, fields map[failedevent.Field]any) error {
					nextRetry, ok := fields[failedevent.FieldNextRetryAt].(time.Time)
					if !ok {
						t.Fatalf("NextRetryAt field is not time.Time")
					}
					// the backoff should be approximately the expected duration from "now"
					// since we can't know the exact "now", we verify the gap is at least the expected backoff
					expectedDuration := time.Duration(tt.expectMinBackoffMinutes) * time.Minute
					// nextRetry should be at least expectedDuration from epoch (a very rough check)
					// More importantly, verify it's a reasonable future time
					if nextRetry.Before(time.Now().Add(expectedDuration - 1*time.Minute)) {
						t.Errorf("NextRetryAt too early. expected at least %v minutes from now, got: %v", tt.expectMinBackoffMinutes, nextRetry)
					}
					return nil
				},
			)

			err := h.RetryPending(ctx)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
