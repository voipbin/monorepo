package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
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
				Data:      []byte(`{"id":"44d91d6e-f2e6-11ee-b176-fb8b4a9ef5a9"}`),
			},

			expectCustomer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("44d91d6e-f2e6-11ee-b176-fb8b4a9ef5a9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				rabbitSock:        mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
