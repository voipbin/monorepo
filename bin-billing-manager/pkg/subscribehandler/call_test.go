package subscribehandler

import (
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventCMCallProgressing(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c9966842-0a00-11ee-aeec-6f47de9442d0"}`),
			},

			expectCall: &cmcall.Call{
				ID: uuid.FromStringOrNil("c9966842-0a00-11ee-aeec-6f47de9442d0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				rabbitSock:     mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventCMCallProgressing(gomock.Any(), tt.expectCall).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCall *cmcall.Call
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d0876e10-0a02-11ee-b210-37573dac67b2"}`),
			},

			expectCall: &cmcall.Call{
				ID: uuid.FromStringOrNil("d0876e10-0a02-11ee-b210-37573dac67b2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				rabbitSock:     mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventCMCallHangup(gomock.Any(), tt.expectCall).Return(nil)
			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
