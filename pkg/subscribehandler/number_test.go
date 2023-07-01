package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/billinghandler"
)

func Test_processEventNMNumberCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseBilling *billing.Billing

		expectCustomerID     uuid.UUID
		expectReferenceType  billing.ReferenceType
		expectReferenceID    uuid.UUID
		expectTMBillingStart string
		expectSource         *commonaddress.Address
		expectDestinations   []*commonaddress.Address
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "number-manager",
				Type:      nmnumber.EventTypeNumberCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"cd6fd39a-16a8-11ee-9904-df3ccc63bfb3","customer_id":"cd9e7db2-16a8-11ee-ba21-e39663e1a49a","tm_create":"2023-06-08 03:22:17.995000"}`),
			},

			responseBilling: &billing.Billing{
				ID: uuid.FromStringOrNil("deeda930-16a8-11ee-833d-3355c466a775"),
			},

			expectCustomerID:     uuid.FromStringOrNil("cd9e7db2-16a8-11ee-ba21-e39663e1a49a"),
			expectReferenceType:  billing.ReferenceTypeNumber,
			expectReferenceID:    uuid.FromStringOrNil("cd6fd39a-16a8-11ee-9904-df3ccc63bfb3"),
			expectTMBillingStart: "2023-06-08 03:22:17.995000",
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

			mockBilling.EXPECT().BillingStart(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectTMBillingStart, &commonaddress.Address{}, &commonaddress.Address{}).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventNMNumberRenewed(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseBilling *billing.Billing

		expectCustomerID     uuid.UUID
		expectReferenceType  billing.ReferenceType
		expectReferenceID    uuid.UUID
		expectTMBillingStart string
		expectSource         *commonaddress.Address
		expectDestinations   []*commonaddress.Address
	}{
		{
			name: "normal",

			event: &rabbitmqhandler.Event{
				Publisher: "number-manager",
				Type:      nmnumber.EventTypeNumberRenewed,
				DataType:  "application/json",
				Data:      []byte(`{"id":"2ffc1ac8-16a9-11ee-9ed1-ef47bf12e497","customer_id":"30254330-16a9-11ee-9cbe-833ff43e836e","tm_create":"2023-06-08 03:22:17.995000"}`),
			},

			responseBilling: &billing.Billing{
				ID: uuid.FromStringOrNil("deeda930-16a8-11ee-833d-3355c466a775"),
			},

			expectCustomerID:     uuid.FromStringOrNil("30254330-16a9-11ee-9cbe-833ff43e836e"),
			expectReferenceType:  billing.ReferenceTypeNumber,
			expectReferenceID:    uuid.FromStringOrNil("2ffc1ac8-16a9-11ee-9ed1-ef47bf12e497"),
			expectTMBillingStart: "2023-06-08 03:22:17.995000",
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

			mockBilling.EXPECT().BillingStart(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectTMBillingStart, &commonaddress.Address{}, &commonaddress.Address{}).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
