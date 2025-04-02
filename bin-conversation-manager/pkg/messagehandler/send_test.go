package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_SendToConversation_sendToConversationLine(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string
		medias       []media.Media

		responseAccount *account.Account
		responseUUID    uuid.UUID

		expectMessage *message.Message
	}{
		{
			name: "line text type",

			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				AccountID:     uuid.FromStringOrNil("086b4920-fe3f-11ed-b570-bf801ec89642"),
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
				Source: &commonaddress.Address{
					Target: "75a20d08-f1de-11ec-8eb1-97f517197fe2",
				},
			},
			text:   "hello, this is test message.",
			medias: []media.Media{},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("086b4920-fe3f-11ed-b570-bf801ec89642"),
				},
			},
			responseUUID: uuid.FromStringOrNil("a0a1478e-0246-11ee-9a48-57bb5aa639c8"),

			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0a1478e-0246-11ee-9a48-57bb5aa639c8"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				ConversationID: uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusSending,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
				TransactionID:  "",
				Source: &commonaddress.Address{
					Target: "75a20d08-f1de-11ec-8eb1-97f517197fe2",
				},
				Text:   "hello, this is test message.",
				Medias: []media.Media{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.conversation.AccountID).Return(tt.responseAccount, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			mockLine.EXPECT().Send(ctx, tt.conversation, tt.responseAccount, tt.text, tt.medias).Return(nil)

			// update
			mockDB.EXPECT().MessageUpdateStatus(ctx, tt.expectMessage.ID, message.StatusSent).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectMessage.ID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageUpdated, tt.expectMessage)

			res, err := h.Send(ctx, tt.conversation, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectMessage, res)
			}
		})
	}
}
