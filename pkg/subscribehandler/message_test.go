package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/billinghandler"
)

func Test_processEventMMMessageCreated(t *testing.T) {

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
				Publisher: "message-manager",
				Type:      mmmessage.EventTypeMessageCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bacfd05c-0a03-11ee-aff4-37ba74639f85","customer_id":"bb028254-0a03-11ee-ad61-6baee66c3ff3","source":{"target":"+821100000001"},"targets":[{"destination":{"target":"+821100000002"}},{"destination":{"target":"+821100000003"}}],"tm_create":"2023-06-08 03:22:17.995000"}`),
			},

			responseBilling: &billing.Billing{
				ID: uuid.FromStringOrNil("d0efef8a-0a02-11ee-8a9f-8f115907c927"),
			},

			expectCustomerID:     uuid.FromStringOrNil("bb028254-0a03-11ee-ad61-6baee66c3ff3"),
			expectReferenceType:  billing.ReferenceTypeSMS,
			expectReferenceID:    uuid.FromStringOrNil("bacfd05c-0a03-11ee-aff4-37ba74639f85"),
			expectTMBillingStart: "2023-06-08 03:22:17.995000",
			expectSource: &commonaddress.Address{
				Target: "+821100000001",
			},
			expectDestinations: []*commonaddress.Address{
				{
					Target: "+821100000002",
				},
				{
					Target: "+821100000003",
				},
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

			for _, destination := range tt.expectDestinations {
				mockBilling.EXPECT().BillingStart(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectTMBillingStart, tt.expectSource, destination).Return(nil)
			}

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
