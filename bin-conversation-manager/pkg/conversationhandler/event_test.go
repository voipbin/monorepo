package conversationhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
)

func Test_Event_eventSMS_single_target(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		responseUUID uuid.UUID

		expectedMessageID          uuid.UUID
		expectedSelf               commonaddress.Address
		expectedPeer               commonaddress.Address
		expectedConversation       *conversation.Conversation
		expectedMessageDirection   message.Direction
		expectedMessageReferenceID uuid.UUID
		expectedMessageText        string
	}{
		{
			name: "normal",

			data: []byte(`{"id":"1fe68e20-f12f-11ec-84fe-03665484eeb6","customer_id":"7aba87f6-1a81-11f0-9dd4-9f991300dc23","source":{"type":"tel","target":"+886912345678"},"targets":[{"destination":{"type":"tel","target":"+886987654321"}}],"text":"hello, this is test message.","direction":"inbound"}`),

			responseUUID: uuid.FromStringOrNil("2aa9bc18-1a82-11f0-acbf-1fa64c8c8586"),

			expectedMessageID: uuid.FromStringOrNil("1fe68e20-f12f-11ec-84fe-03665484eeb6"),
			expectedSelf: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+886987654321",
			},
			expectedPeer: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+886912345678",
			},
			expectedConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2aa9bc18-1a82-11f0-acbf-1fa64c8c8586"),
					CustomerID: uuid.FromStringOrNil("7aba87f6-1a81-11f0-9dd4-9f991300dc23"),
				},
				Name:     "conversation with ",
				Detail:   "conversation with ",
				Type:     conversation.TypeMessage,
				DialogID: "",
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+886987654321",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+886912345678",
				},
			},
			expectedMessageDirection:   message.DirectionIncoming,
			expectedMessageReferenceID: uuid.FromStringOrNil("1fe68e20-f12f-11ec-84fe-03665484eeb6"),
			expectedMessageText:        "hello, this is test message.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSMS := smshandler.NewMockSMSHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				messageHandler: mockMessage,
				smsHandler:     mockSMS,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, tt.expectedSelf, tt.expectedPeer).Return(nil, fmt.Errorf(""))

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ConversationCreate(ctx, tt.expectedConversation).Return(nil)
			mockDB.EXPECT().ConversationGet(ctx, tt.responseUUID).Return(tt.expectedConversation, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedConversation.CustomerID, conversation.EventTypeConversationCreated, tt.expectedConversation)

			mockMessage.EXPECT().Create(
				ctx,
				tt.expectedMessageID,
				tt.expectedConversation.CustomerID,
				tt.expectedConversation.ID,
				tt.expectedMessageDirection,
				message.StatusDone,
				message.ReferenceTypeMessage,
				tt.expectedMessageReferenceID,
				"",
				tt.expectedMessageText,
				[]media.Media{},
			).Return(&message.Message{}, nil)

			if err := h.Event(ctx, conversation.TypeMessage, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
