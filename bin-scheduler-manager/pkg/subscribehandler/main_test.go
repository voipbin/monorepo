package subscribehandler

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-scheduler-manager/pkg/schedulehandler"
)

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	h := NewSubscribeHandler(mockSock, "bin-manager.scheduler-manager.subscribe", []string{"bin-manager.customer-manager.event"}, mockSchedule)
	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func Test_processEvent_routing(t *testing.T) {
	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("29c7a2be-6f23-11f0-b3f6-9f2a3ba4bd12"),
	}
	customerData, _ := json.Marshal(customer)

	tests := []struct {
		name  string
		event *sock.Event

		expectHandled bool
	}{
		{
			name: "customer_deleted is handled",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},

			expectHandled: true,
		},
		{
			name: "other publisher is ignored",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},

			expectHandled: false,
		},
		{
			name: "other event type is ignored",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      "customer_created",
				Data:      json.RawMessage(customerData),
			},

			expectHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

			h := &subscribeHandler{
				sockHandler:     mockSock,
				scheduleHandler: mockSchedule,
			}

			if tt.expectHandled {
				mockSchedule.EXPECT().EventCustomerDeleted(gomock.Any(), customer).Return(nil)
			}

			// call processEvent synchronously (processEventRun spawns it on a
			// goroutine in production; the routing logic is what is under test)
			h.processEvent(tt.event)
		})
	}
}
