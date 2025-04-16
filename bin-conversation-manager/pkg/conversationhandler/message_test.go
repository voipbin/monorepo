package conversationhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

func Test_MessageSend(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		text           string
		medias         []media.Media

		responseConversation *conversation.Conversation
		responseMessage      *message.Message
	}{
		{
			"line text type",

			uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
			"hello, this is test message.",
			[]media.Media{},

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
					CustomerID: uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				},
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
			},

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9d11dae8-e870-11ec-b319-fb0d0b15716f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockMessage.EXPECT().Send(ctx, tt.responseConversation, tt.text, tt.medias).Return(tt.responseMessage, nil)
			res, err := h.MessageSend(ctx, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}
