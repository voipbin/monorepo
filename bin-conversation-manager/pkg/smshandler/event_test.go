package smshandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_Event(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		responseAccount *account.Account

		expectResMessages []*message.Message
		expectResLocal    *commonaddress.Address
	}{
		{
			name: "received message",

			data: []byte(`{"id":"eeafd418-7a4e-11eb-8750-9bb0ca1d7926","customer_id":"197609d6-a29b-11ec-b884-5b8a227db58a","type":"","source":{"target":"+821100000001"},"targets":[{"destination":{"target":"+821100000002"}}],"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"inbound","tm_create":"","tm_update":"","tm_delete":""}`),

			expectResMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("197609d6-a29b-11ec-b884-5b8a227db58a"),
					},
					Status:        message.StatusDone,
					ReferenceType: conversation.ReferenceTypeMessage,
					ReferenceID:   "+821100000001",
					TransactionID: "eeafd418-7a4e-11eb-8750-9bb0ca1d7926",
				},
			},
			expectResLocal: &commonaddress.Address{
				Target: "+821100000002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := smsHandler{
				reqHandler:     mockReq,
				accountHandler: mockAccount,
			}

			ctx := context.Background()

			res, resLocal, err := h.Event(ctx, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectResMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResMessages, res)
			}
			if !reflect.DeepEqual(resLocal, tt.expectResLocal) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResLocal, resLocal)
			}
		})
	}
}

func Test_getReferenceID(t *testing.T) {

	tests := []struct {
		name string

		message *mmmessage.Message
		idx     int

		expectRes string
	}{
		{
			name: "received message",

			message: &mmmessage.Message{
				Direction: mmmessage.DirectionInbound,
				Source: &commonaddress.Address{
					Target: "+821100000001",
				},
			},
			idx: 0,

			expectRes: "+821100000001",
		},
		{
			name: "outbound message",

			message: &mmmessage.Message{
				Direction: mmmessage.DirectionOutbound,
				Source: &commonaddress.Address{
					Target: "+821100000001",
				},
				Targets: []mmtarget.Target{
					{
						Destination: commonaddress.Address{
							Target: "+821100000002",
						},
					},
				},
			},
			idx: 0,

			expectRes: "+821100000002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := smsHandler{
				reqHandler:     mockReq,
				accountHandler: mockAccount,
			}

			res := h.getReferenceID(tt.message, tt.idx)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
