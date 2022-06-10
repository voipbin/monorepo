package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
)

func Test_SendToConversation(t *testing.T) {

	tests := []struct {
		name string

		conversation *conversation.Conversation
		messageType  message.Type
		messageData  []byte

		responseMessage *message.Message
	}{
		{
			"line text type",

			&conversation.Conversation{
				ID:            uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
				CustomerID:    uuid.FromStringOrNil("e54ded88-e6ef-11ec-83af-7fac5b21e9aa"),
				ReferenceType: conversation.ReferenceTypeLine,
				ReferenceID:   "18a7a0e8-e6f0-11ec-8cee-47dd7e7164e3",
			},
			message.TypeText,
			[]byte(`"hello, this is test message."`),

			&message.Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}

			ctx := context.Background()

			mockLine.EXPECT().Send(ctx, tt.conversation.CustomerID, tt.conversation.ReferenceID, tt.messageType, tt.messageData).Return(nil)
			mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, gomock.Any()).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), message.EventTypeMessageCreated, gomock.Any())

			res, err := h.SendToConversation(ctx, tt.conversation, tt.messageType, tt.messageData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}
