package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/pkg/contacthandler"
)

func Test_processEventCMCustomerDeleted(t *testing.T) {
	tests := []struct {
		name string

		customer *cmcustomer.Customer

		expectError bool
	}{
		{
			name: "normal - customer deleted",
			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &subscribeHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}
			ctx := context.Background()

			// Marshal customer data for the event
			customerData, _ := json.Marshal(tt.customer)

			event := &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      customerData,
			}

			mockContact.EXPECT().EventCustomerDeleted(ctx, gomock.Any()).Return(nil)

			err := h.processEventCMCustomerDeleted(ctx, event)
			if (err != nil) != tt.expectError {
				t.Errorf("processEventCMCustomerDeleted() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func Test_processEventCMCustomerDeleted_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}
	ctx := context.Background()

	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
	}
	customerData, _ := json.Marshal(customer)

	event := &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      customerData,
	}

	mockContact.EXPECT().EventCustomerDeleted(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	err := h.processEventCMCustomerDeleted(ctx, event)
	if err == nil {
		t.Error("processEventCMCustomerDeleted() expected error")
	}
}

func Test_processEventCMCustomerDeleted_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}
	ctx := context.Background()

	event := &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage("invalid json"),
	}

	err := h.processEventCMCustomerDeleted(ctx, event)
	if err == nil {
		t.Error("processEventCMCustomerDeleted() expected error for invalid JSON")
	}
}

func Test_processEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		expectErr bool
	}{
		{
			name: "customer deleted event",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(`{"id":"11111111-1111-1111-1111-111111111111"}`),
			},
			expectErr: false,
		},
		{
			name: "unknown event - should be ignored",
			event: &sock.Event{
				Publisher: "unknown-publisher",
				Type:      "unknown_event",
				Data:      json.RawMessage(`{}`),
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &subscribeHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			// Only expect EventCustomerDeleted for customer deleted events
			if tt.event.Publisher == publisherCustomerManager && tt.event.Type == string(cmcustomer.EventTypeCustomerDeleted) {
				mockContact.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			}

			// processEvent runs asynchronously, but we call it synchronously here
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	event := &sock.Event{
		Publisher: "unknown",
		Type:      "unknown",
		Data:      json.RawMessage(`{}`),
	}

	// processEventRun should not return error (it just spawns goroutine)
	err := h.processEventRun(event)
	if err != nil {
		t.Errorf("processEventRun() error = %v", err)
	}
}

func TestNewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := NewSubscribeHandler(mockSock, "test-queue", []string{"target1", "target2"}, mockContact)
	if h == nil {
		t.Error("NewSubscribeHandler() returned nil")
	}
}

func Test_Run_QueueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{"target1"},
		contactHandler:   mockContact,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(fmt.Errorf("queue create error"))

	err := h.Run()
	if err == nil {
		t.Error("Run() expected error for queue create failure")
	}
}

func Test_Run_SubscribeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{"target1"},
		contactHandler:   mockContact,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe("test-queue", "target1").Return(fmt.Errorf("subscribe error"))

	err := h.Run()
	if err == nil {
		t.Error("Run() expected error for subscribe failure")
	}
}

func Test_Run_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{"target1", "target2"},
		contactHandler:   mockContact,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe("test-queue", "target1").Return(nil)
	mockSock.EXPECT().QueueSubscribe("test-queue", "target2").Return(nil)
	// ConsumeMessage runs in a goroutine, so we use AnyTimes() to avoid blocking
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), "test-queue", gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	err := h.Run()
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func Test_processEvent_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Create an event that will trigger processEventCMCustomerDeleted which returns an error
	event := &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage(`{"id":"11111111-1111-1111-1111-111111111111"}`),
	}

	mockContact.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(fmt.Errorf("database error"))

	// processEvent handles error internally by logging (doesn't return error)
	h.processEvent(event)
}

func Test_processEvent_UnknownPublisher(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Unknown publisher - should be ignored silently
	event := &sock.Event{
		Publisher: "some-other-service",
		Type:      "some_event",
		Data:      json.RawMessage(`{}`),
	}

	// No expectations - event should be ignored
	h.processEvent(event)
}

func Test_processEvent_UnknownEventType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &subscribeHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Known publisher but unknown event type - should be ignored silently
	event := &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      "unknown_event_type",
		Data:      json.RawMessage(`{}`),
	}

	// No expectations - event should be ignored
	h.processEvent(event)
}
