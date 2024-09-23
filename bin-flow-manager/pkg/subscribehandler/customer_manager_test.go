package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_processEvent_processEventCMCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectCustomer *cmcustomer.Customer
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "customer-manager",
				Type:      cmcustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
			},

			expectCustomer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("4f8fbc3c-ccca-11ee-8104-9f5b184cb220"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlow := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := subscribeHandler{
				sockHandler:       mockSock,
				flowHandler:       mockFlow,
				activeflowHandler: mockActive,
			}

			mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockActive.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
