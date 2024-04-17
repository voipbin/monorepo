package messagehandlermessagebird

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/messagebird"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/requestexternal"
)

func Test_marshal(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		customerID uuid.UUID
		sender     *commonaddress.Address
		targets    []target.Target
		text       string

		expectSender    string
		expectReceivers []string

		responseSend *messagebird.Message
		expectRes    []target.Target
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("883112b8-a0f1-11ec-a2da-efa31b2f00ae"),
			customerID: uuid.FromStringOrNil("88b74356-a0f1-11ec-bbfc-f3ae56ab6783"),
			sender: &commonaddress.Address{
				Target: "+821100000001",
			},
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Target: "+31616818985",
					},
				},
				{
					Destination: commonaddress.Address{
						Target: "+821021656521",
					},
				},
			},
			text: "This is a test message10",

			expectSender: "+821100000001",
			expectReceivers: []string{
				"+31616818985",
				"+821021656521",
			},
			responseSend: &messagebird.Message{
				ID:              "6b79e50e426c4d64ac45345bae84fe55",
				Href:            "https://rest.messagebird.com/messages/6b79e50e426c4d64ac45345bae84fe55",
				Direction:       "mt",
				Type:            "sms",
				Originator:      "+821100000001",
				Body:            "This is a test message10",
				Gateway:         10,
				DataCoding:      "plain",
				MClass:          1,
				CreatedDatetime: "2022-03-09T05:21:45+00:00",
				Recipients: messagebird.RecipientStruct{
					TotalCount:               2,
					TotalSentCount:           2,
					TotalDeliveredCount:      0,
					TotalDeliveryFailedCount: 0,
					Items: []messagebird.Recipient{
						{
							Recipient:        31616818985,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
						{
							Recipient:        821021656521,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
					},
				},
			},
			expectRes: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+31616818985",
					},
					Status: target.StatusSent,
					Parts:  1,
				},
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
					Status: target.StatusSent,
					Parts:  1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockExternalReq := requestexternal.NewMockRequestExternal(mc)

			h := &messageHandlerMessagebird{
				reqHandler:      mockReq,
				db:              mockDB,
				requestExternal: mockExternalReq,
			}

			mockExternalReq.EXPECT().MessagebirdSendMessage(tt.expectSender, tt.expectReceivers, tt.text).Return(tt.responseSend, nil)

			res, err := h.SendMessage(tt.id, tt.customerID, tt.sender, tt.targets, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
