package smshandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
)

func Test_Send(t *testing.T) {

	tests := []struct {
		name string

		conversation  *conversation.Conversation
		transactionID string
		text          string

		responseMessage *mmmessage.Message

		expectDestinations []commonaddress.Address
	}{
		{
			name: "received message",

			conversation: &conversation.Conversation{
				ID:          uuid.FromStringOrNil("b3181e20-ffd4-11ed-aa4e-37a91163c788"),
				ReferenceID: "b39d29ee-ffd4-11ed-9b1e-170678b894f5",
			},
			transactionID: "b37322e8-ffd4-11ed-a984-7b6db99c07e8",
			text:          "test message.",

			responseMessage: &mmmessage.Message{
				ID: uuid.FromStringOrNil("b39d29ee-ffd4-11ed-9b1e-170678b894f5"),
			},
			expectDestinations: []commonaddress.Address{
				{
					Target: "b39d29ee-ffd4-11ed-9b1e-170678b894f5",
				},
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

			mockReq.EXPECT().MessageV1MessageSend(ctx, uuid.FromStringOrNil(tt.transactionID), tt.conversation.CustomerID, tt.conversation.Source, tt.expectDestinations, tt.text).Return(tt.responseMessage, nil)

			if err := h.Send(ctx, tt.conversation, tt.transactionID, tt.text); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
