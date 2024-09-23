package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			h := subscribeHandler{
				sockHandler:       mockSock,
				transcribeHandler: mockTranscribe,
			}

			mockTranscribe.EXPECT().EventCUCustomerDeleted(gomock.Any(), tt.expectCustomer).Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
