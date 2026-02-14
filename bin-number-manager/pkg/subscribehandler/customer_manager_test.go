package subscribehandler

import (
	"context"
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

func Test_processEvent_processEventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cmcustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      cmcustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c7e1cd82-ecb5-11ee-8425-779e09b1f43b"}`),
			},

			expectCustomer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("c7e1cd82-ecb5-11ee-8425-779e09b1f43b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := subscribeHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCUCustomerDeleted_UnmarshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := subscribeHandler{
		sockHandler:   mockSock,
		numberHandler: mockNumber,
	}

	event := &sock.Event{
		Publisher: "customer-manager",
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		DataType:  "application/json",
		Data:      []byte(`{invalid json`),
	}

	err := h.processEventCUCustomerDeleted(context.Background(), event)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func Test_processEventCUCustomerDeleted_HandlerError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := subscribeHandler{
		sockHandler:   mockSock,
		numberHandler: mockNumber,
	}

	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("12345678-1234-1234-1234-123456789012"),
	}

	event := &sock.Event{
		Publisher: "customer-manager",
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		DataType:  "application/json",
		Data:      []byte(`{"id":"12345678-1234-1234-1234-123456789012"}`),
	}

	mockNumber.EXPECT().EventCustomerDeleted(gomock.Any(), customer).Return(fmt.Errorf("handler error"))

	err := h.processEventCUCustomerDeleted(context.Background(), event)
	if err == nil {
		t.Error("Expected error from handler, got nil")
	}
}
