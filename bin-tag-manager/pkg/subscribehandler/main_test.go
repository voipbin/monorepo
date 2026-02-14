package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_processEvent(t *testing.T) {
	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
	}

	customerData, _ := json.Marshal(customer)

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "processes_customer_deleted_event",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},
		},
		{
			name: "ignores_unknown_event",
			event: &sock.Event{
				Publisher: "unknown-service",
				Type:      "unknown_event",
				Data:      json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &subscribeHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.event.Publisher == publisherCustomerManager {
				mockTag.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			}

			// processEvent runs in background, just call it directly
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "runs_process_event",
			event: &sock.Event{
				Publisher: "unknown-service",
				Type:      "unknown_event",
				Data:      json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &subscribeHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			err := h.processEventRun(tt.event)
			if err != nil {
				t.Errorf("processEventRun should not return error, got: %v", err)
			}
		})
	}
}

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	subscribeQueue := "test-queue"
	subscribeTargets := []string{"target1", "target2"}

	h := NewSubscribeHandler(mockSock, subscribeQueue, subscribeTargets, mockTag)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func Test_Run(t *testing.T) {
	tests := []struct {
		name             string
		subscribeQueue   string
		subscribeTargets []string
		queueCreateErr   error
		expectError      bool
	}{
		{
			name:             "runs_successfully",
			subscribeQueue:   "test-queue",
			subscribeTargets: []string{"target1"},
			queueCreateErr:   nil,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &subscribeHandler{
				sockHandler:      mockSock,
				subscribeQueue:   tt.subscribeQueue,
				subscribeTargets: tt.subscribeTargets,
				tagHandler:       mockTag,
			}

			mockSock.EXPECT().QueueCreate(tt.subscribeQueue, "normal").Return(tt.queueCreateErr)

			if tt.queueCreateErr == nil {
				for _, target := range tt.subscribeTargets {
					mockSock.EXPECT().QueueSubscribe(tt.subscribeQueue, target).Return(nil)
				}
				mockSock.EXPECT().ConsumeMessage(
					context.Background(),
					tt.subscribeQueue,
					gomock.Any(),
					false,
					false,
					false,
					10,
					gomock.Any(),
				).Return(nil).AnyTimes()
			}

			err := h.Run()
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
