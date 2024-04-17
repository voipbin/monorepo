package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/activeflowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
)

func Test_processEvent_processEventCMCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCustomer *cmcustomer.Customer
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockFlow := flowhandler.NewMockFlowHandler(mc)
			mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

			h := subscribeHandler{
				rabbitSock:        mockSock,
				flowHandler:       mockFlow,
				activeflowHandler: mockActive,
			}

			mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockActive.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
