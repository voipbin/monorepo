package failedeventhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_NewFailedEventHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	called := 0
	processor := func(_ *sock.Event) error {
		called++
		return nil
	}

	res := NewFailedEventHandler(mockDB, processor)
	h, ok := res.(*failedEventHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *failedEventHandler, got: %T", res)
	}
	if h.db != mockDB || h.utilHandler == nil || h.eventProcessor == nil {
		t.Fatalf("Wrong match. the dependencies must be wired. got: %+v", h)
	}

	if err := h.eventProcessor(&sock.Event{}); err != nil || called != 1 {
		t.Errorf("Wrong match. the given processor must be kept. err: %v, called: %d", err, called)
	}
}

// Test_RetryPending_storeFailures verifies that a failing status write does not stop the retry run:
// the counters still reflect the processing outcome of each event.
func Test_RetryPending_storeFailures(t *testing.T) {
	retryableEvent := &failedevent.FailedEvent{
		ID:             uuid.FromStringOrNil("bb100001-0001-0001-0001-000000000001"),
		EventType:      "call_progressing",
		EventPublisher: "call-manager",
		EventData:      `{"type":"call_progressing","publisher":"call-manager","data_type":"application/json"}`,
		RetryCount:     0,
		MaxRetries:     5,
		Status:         failedevent.StatusPending,
	}
	exhaustedEvent := &failedevent.FailedEvent{
		ID:             uuid.FromStringOrNil("bb100002-0001-0001-0001-000000000001"),
		EventType:      "call_hangup",
		EventPublisher: "call-manager",
		EventData:      `{"type":"call_hangup","publisher":"call-manager","data_type":"application/json"}`,
		RetryCount:     4,
		MaxRetries:     5,
		Status:         failedevent.StatusRetrying,
	}

	tests := []struct {
		name string

		event         *failedevent.FailedEvent
		processorErr  error
		expectUpdate  bool
		expectDelete  bool
		expectSuccess int
		expectExhaust int
	}{
		{
			name:         "retry bookkeeping update fails",
			event:        retryableEvent,
			processorErr: fmt.Errorf("still failing"),
			expectUpdate: true,
		},
		{
			name:          "delete of the retried event fails",
			event:         retryableEvent,
			expectDelete:  true,
			expectSuccess: 1,
		},
		{
			name:          "exhausted mark fails",
			event:         exhaustedEvent,
			processorErr:  fmt.Errorf("still failing"),
			expectUpdate:  true,
			expectExhaust: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &failedEventHandler{
				utilHandler:    utilhandler.NewUtilHandler(),
				db:             mockDB,
				eventProcessor: func(_ *sock.Event) error { return tt.processorErr },
			}
			ctx := context.Background()

			mockDB.EXPECT().FailedEventListPendingRetry(ctx, gomock.Any()).Return([]*failedevent.FailedEvent{tt.event}, nil)
			if tt.expectUpdate {
				mockDB.EXPECT().FailedEventUpdate(ctx, tt.event.ID, gomock.Any()).Return(fmt.Errorf("store unavailable"))
			}
			if tt.expectDelete {
				mockDB.EXPECT().FailedEventDelete(ctx, tt.event.ID).Return(fmt.Errorf("store unavailable"))
			}

			retried, succeeded, exhausted, err := h.RetryPending(ctx)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if retried != 1 || succeeded != tt.expectSuccess || exhausted != tt.expectExhaust {
				t.Errorf("Wrong match. expect: 1/%d/%d, got: %d/%d/%d", tt.expectSuccess, tt.expectExhaust, retried, succeeded, exhausted)
			}
		})
	}
}

// Test_Save_eventCannotBeEncoded verifies that an event whose payload is not valid JSON is rejected
// before anything is stored.
func Test_Save_eventCannotBeEncoded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectation on the store: gomock fails the test if a write is attempted
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &failedEventHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		db:             mockDB,
		eventProcessor: func(_ *sock.Event) error { return nil },
	}

	event := &sock.Event{
		Type:      "call_progressing",
		Publisher: "call-manager",
		DataType:  "application/json",
		Data:      json.RawMessage(`{broken`),
	}

	err := h.Save(context.Background(), event, fmt.Errorf("processing failed"))
	if err == nil || !strings.Contains(err.Error(), "could not marshal event") {
		t.Errorf("wrong error. expect the marshal failure, got: %v", err)
	}
}
