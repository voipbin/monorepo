package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
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
				rabbitSock:    mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().EventCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			h.processEvent(tt.event)
		})
	}
}
