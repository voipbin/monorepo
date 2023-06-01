package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
)

func Test_SendToConversation(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		text         string
		medias       []media.Media

		responseAccount *account.Account
		responseMessage *message.Message
	}{
		{
			name: "line text type",

			conversation: &conversation.Conversation{
				ID:            uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
				CustomerID:    uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
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
				ID: uuid.FromStringOrNil("086b4920-fe3f-11ed-b570-bf801ec89642"),
			},
			responseMessage: &message.Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.conversation.AccountID).Return(tt.responseAccount, nil)

			// create
			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)

			mockLine.EXPECT().Send(ctx, tt.conversation, tt.responseAccount, tt.text, tt.medias).Return(nil)

			// update
			mockDB.EXPECT().MessageUpdateStatus(ctx, tt.responseMessage.ID, message.StatusSent).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseMessage.ID).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageUpdated, tt.responseMessage)

			res, err := h.SendToConversation(ctx, tt.conversation, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}
