package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventNMNumberCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectNumber *nmnumber.Number
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "number-manager",
				Type:      nmnumber.EventTypeNumberCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"cd6fd39a-16a8-11ee-9904-df3ccc63bfb3"}`),
			},

			expectNumber: &nmnumber.Number{
				ID: uuid.FromStringOrNil("cd6fd39a-16a8-11ee-9904-df3ccc63bfb3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventNMNumberCreated(gomock.Any(), tt.expectNumber).Return(nil)
			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventNMNumberRenewed(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectNumber *nmnumber.Number
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "number-manager",
				Type:      nmnumber.EventTypeNumberRenewed,
				DataType:  "application/json",
				Data:      []byte(`{"id":"2ffc1ac8-16a9-11ee-9ed1-ef47bf12e497"}`),
			},

			expectNumber: &nmnumber.Number{
				ID: uuid.FromStringOrNil("2ffc1ac8-16a9-11ee-9ed1-ef47bf12e497"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventNMNumberRenewed(gomock.Any(), tt.expectNumber).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
