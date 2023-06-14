package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/billinghandler"
)

func Test_processEventCMCallProgressing(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		expectCustomerID    uuid.UUID
		expectReferenceType billing.ReferenceType
		expectReferenceID   uuid.UUID
		expectTMProgressing string
		expectSource        *commonaddress.Address
		expectDestination   *commonaddress.Address
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallProgressing,
				DataType:  "application/json",
				Data:      []byte(`{"id":"c9966842-0a00-11ee-aeec-6f47de9442d0","customer_id":"cab6c6ae-0a00-11ee-b5ee-7f66d2a6b84b","source":{"target":"+821100000001"},"destination":{"target":"+821100000002"},"tm_progressing":"2023-06-08 03:22:17.995000"}`),
			},

			expectCustomerID:    uuid.FromStringOrNil("cab6c6ae-0a00-11ee-b5ee-7f66d2a6b84b"),
			expectReferenceType: billing.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("c9966842-0a00-11ee-aeec-6f47de9442d0"),
			expectTMProgressing: "2023-06-08 03:22:17.995000",
			expectSource: &commonaddress.Address{
				Target: "+821100000001",
			},
			expectDestination: &commonaddress.Address{
				Target: "+821100000002",
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

			mockBilling.EXPECT().BillingStart(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectTMProgressing, tt.expectSource, tt.expectDestination).Return(nil)

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

		responseBilling *billing.Billing

		expectCustomerID    uuid.UUID
		expectReferenceType billing.ReferenceType
		expectReferenceID   uuid.UUID
		expectTMHangup      string
		expectSource        *commonaddress.Address
		expectDestination   *commonaddress.Address
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "call-manager",
				Type:      cmcall.EventTypeCallHangup,
				DataType:  "application/json",
				Data:      []byte(`{"id":"d0876e10-0a02-11ee-b210-37573dac67b2","customer_id":"d0be2928-0a02-11ee-ba66-abf8f33982c7","source":{"target":"+821100000001"},"destination":{"target":"+821100000002"},"tm_progressing":"2023-06-08 03:22:17.995000","tm_hangup":"2023-06-08 03:22:27.995000"}`),
			},

			responseBilling: &billing.Billing{
				ID: uuid.FromStringOrNil("d0efef8a-0a02-11ee-8a9f-8f115907c927"),
			},

			expectCustomerID:    uuid.FromStringOrNil("d0be2928-0a02-11ee-ba66-abf8f33982c7"),
			expectReferenceType: billing.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("d0876e10-0a02-11ee-b210-37573dac67b2"),
			expectTMHangup:      "2023-06-08 03:22:27.995000",
			expectSource: &commonaddress.Address{
				Target: "+821100000001",
			},
			expectDestination: &commonaddress.Address{
				Target: "+821100000002",
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

			mockBilling.EXPECT().GetByReferenceID(gomock.Any(), tt.expectReferenceID).Return(tt.responseBilling, nil)
			mockBilling.EXPECT().BillingEnd(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectTMHangup, tt.expectSource, tt.expectDestination).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
