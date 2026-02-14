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
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_processEventCMCustomerDeleted(t *testing.T) {
	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
	}

	customerData, _ := json.Marshal(customer)

	tests := []struct {
		name        string
		event       *sock.Event
		handlerErr  error
		expectError bool
	}{
		{
			name: "processes_customer_deleted_event",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},
			handlerErr:  nil,
			expectError: false,
		},
		{
			name: "handles_handler_error",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},
			handlerErr:  fmt.Errorf("handler error"),
			expectError: true,
		},
		{
			name: "handles_invalid_json",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage("invalid json"),
			},
			handlerErr:  nil,
			expectError: true,
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
			ctx := context.Background()

			if string(tt.event.Data) != "invalid json" {
				mockTag.EXPECT().EventCustomerDeleted(ctx, gomock.Any()).Return(tt.handlerErr)
			}

			err := h.processEventCMCustomerDeleted(ctx, tt.event)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
