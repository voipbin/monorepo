package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventCMCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cucustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d5b4a056-0ad4-11ee-b813-bfb4ab48539d"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("d5b4a056-0ad4-11ee-b813-bfb4ab48539d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				billingHandler: mockBilling,
			}

			mockAccount.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer.Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventCMCustomerCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cucustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cucustomer.EventTypeCustomerCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"94ad9d60-c8ec-11ef-8b96-a7032a337c80"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("94ad9d60-c8ec-11ef-8b96-a7032a337c80"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				billingHandler: mockBilling,
			}

			mockAccount.EXPECT().EventCUCustomerCreated(gomock.Any(), tt.expectCustomer.Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
