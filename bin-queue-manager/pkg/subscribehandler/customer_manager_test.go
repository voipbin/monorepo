package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_processEvent_processEventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCustomer *cucustomer.Customer
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := subscribeHandler{
				rabbitSock: mockSock,

				queueHandler:     mockQueue,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)
			mockQueue.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
