package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
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
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      cucustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c7e1cd82-ecb5-11ee-8425-779e09b1f43b"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("c7e1cd82-ecb5-11ee-8425-779e09b1f43b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
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
