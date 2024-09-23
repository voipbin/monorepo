package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_processEvent_processEventCMCustomerDeleted(t *testing.T) {

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
				Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("4f8fbc3c-ccca-11ee-8104-9f5b184cb220"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := subscribeHandler{
				sockHandler: mockSock,

				extensionHandler: mockExtension,
				trunkHandler:     mockTrunk,
			}

			mockExtension.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockTrunk.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
