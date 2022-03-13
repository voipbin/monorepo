package messagehandlermessagebird

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/messagebird"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/requestexternal"
)

func Test_marshal(t *testing.T) {

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

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		sender       *cmaddress.Address
		destinations []cmaddress.Address
		text         string

		expectSender    string
		expectReceivers []string

		responseSend *messagebird.Message
		expectRes    *message.Message
	}{
		{
			"normal",
			uuid.FromStringOrNil("883112b8-a0f1-11ec-a2da-efa31b2f00ae"),
			uuid.FromStringOrNil("88b74356-a0f1-11ec-bbfc-f3ae56ab6783"),

			&cmaddress.Address{
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Target: "+31616818985",
				},
				{
					Target: "+821021656521",
				},
			},
			"This is a test message10",

			"+821100000001",
			[]string{
				"+31616818985",
				"+821021656521",
			},
			&messagebird.Message{
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
			&message.Message{
				ID:         uuid.FromStringOrNil("883112b8-a0f1-11ec-a2da-efa31b2f00ae"),
				CustomerID: uuid.FromStringOrNil("88b74356-a0f1-11ec-bbfc-f3ae56ab6783"),
				Type:       message.TypeSMS,

				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: cmaddress.Address{
							Type:   cmaddress.TypeTel,
							Target: "+31616818985",
						},
						Status: target.StatusSent,
						Parts:  1,
					},
					{
						Destination: cmaddress.Address{
							Type:   cmaddress.TypeTel,
							Target: "+821021656521",
						},
						Status: target.StatusSent,
						Parts:  1,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "This is a test message10",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockExternalReq.EXPECT().MessagebirdSendMessage(tt.expectSender, tt.expectReceivers, tt.text).Return(tt.responseSend, nil)

			res, err := h.SendMessage(tt.id, tt.customerID, tt.sender, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for i, target := range res.Targets {
				tt.expectRes.Targets[i].TMUpdate = target.TMUpdate
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
