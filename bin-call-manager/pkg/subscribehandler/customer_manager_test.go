package subscribehandler

import (
	"testing"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processEvent_processEventCUCustomerDeleted(t *testing.T) {

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
				Data:      []byte(`{"id":"79348cda-f159-11ee-ab1d-8fb25dd5896d"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("79348cda-f159-11ee-ab1d-8fb25dd5896d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroup := groupcallhandler.NewMockGroupcallHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := subscribeHandler{
				sockHandler: mockSock,

				callHandler:       mockCall,
				groupcallHandler:  mockGroup,
				confbridgeHandler: mockConf,
			}

			mockCall.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockGroup.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockConf.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
