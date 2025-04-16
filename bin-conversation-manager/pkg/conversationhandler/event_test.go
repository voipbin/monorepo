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

// func Test_Event(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		referenceType conversation.ReferenceType
// 		data          []byte

// 		responseEvent        []*message.Message
// 		responseCurTime      string
// 		responseConversation *conversation.Conversation
// 	}{
// 		{
// 			name: "normal",

// 			referenceType: conversation.ReferenceTypeMessage,
// 			data:          []byte(`{"id":"5e65e04e-f12d-11ec-b951-53cf815f86a4"}`),

// 			responseEvent: []*message.Message{
// 				{
// 					Identity: commonidentity.Identity{
// 						ID: uuid.FromStringOrNil("c4fd2a56-f12d-11ec-b443-1f1133008bfc"),
// 					},
// 				},
// 			},
// 			responseCurTime: "2022-04-18 03:22:17.995000",
// 			responseConversation: &conversation.Conversation{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("f45df2d0-f12d-11ec-bd7f-2f3e6d9a6218"),
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockSMS := smshandler.NewMockSMSHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)
// 			h := &conversationHandler{
// 				utilHandler:   mockUtil,
// 				db:            mockDB,
// 				notifyHandler: mockNotify,

// 				messageHandler: mockMessage,
// 				smsHandler:     mockSMS,
// 			}

// 			ctx := context.Background()

// 			mockSMS.EXPECT().Event(ctx, tt.data).Return(tt.responseEvent, &commonaddress.Address{}, nil)
// 			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
// 			mockMessage.EXPECT().GetsByTransactionID(ctx, tt.responseEvent[0].TransactionID, tt.responseCurTime, uint64(10)).Return([]*message.Message{}, nil)
// 			for _, tmp := range tt.responseEvent {

// 				mockDB.EXPECT().ConversationGetByReferenceInfo(ctx, tmp.CustomerID, tmp.ReferenceType, gomock.Any()).Return(tt.responseConversation, nil)
// 				mockMessage.EXPECT().Create(
// 					ctx,
// 					tt.responseConversation.CustomerID,
// 					tt.responseConversation.ID,
// 					message.DirectionIncoming,
// 					message.StatusReceived,
// 					conversation.ReferenceTypeMessage,
// 					tmp.ReferenceID,
// 					tmp.ID.String(),
// 					tmp.Text,
// 					tmp.Medias,
// 				).Return(tmp, nil)
// 			}

// 			if err := h.Event(ctx, tt.referenceType, tt.data); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 		})
// 	}
// }

func Test_eventSMS_single_target(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		responseUUID uuid.UUID

		expectedSelf               commonaddress.Address
		expectedPeer               commonaddress.Address
		expectedConversation       *conversation.Conversation
		expectedMessageDirection   message.Direction
		expectedMessageReferenceID string
		expectedMessageText        string
	}{
		{
			name: "normal",

			data: []byte(`{"id":"1fe68e20-f12f-11ec-84fe-03665484eeb6","customer_id":"7aba87f6-1a81-11f0-9dd4-9f991300dc23","source":{"type":"tel","target":"+886912345678"},"targets":[{"destination":{"type":"tel","target":"+886987654321"}}],"text":"hello, this is test message.","direction":"inbound"}`),

			responseUUID: uuid.FromStringOrNil("2aa9bc18-1a82-11f0-acbf-1fa64c8c8586"),

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
				Name:     "conversation",
				Detail:   "conversation with ",
				Type:     conversation.TypeMessage,
				DialogID: "1fe68e20-f12f-11ec-84fe-03665484eeb6",
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
			expectedMessageReferenceID: "1fe68e20-f12f-11ec-84fe-03665484eeb6",
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
			// mockSMS.EXPECT().Event(ctx, tt.data).Return(tt.responseEvent, &commonaddress.Address{}, nil)
			// mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			// mockMessage.EXPECT().GetsByTransactionID(ctx, tt.responseEvent[0].TransactionID, gomock.Any(), uint64(10)).Return([]*message.Message{}, nil)

			// for _, tmp := range tt.responseEvent {

			// 	mockDB.EXPECT().ConversationGetByReferenceInfo(ctx, tmp.CustomerID, tmp.ReferenceType, gomock.Any()).Return(tt.responseConversation, nil)
			// 	mockMessage.EXPECT().Create(
			// 		ctx,
			// 		tt.responseConversation.CustomerID,
			// 		tt.responseConversation.ID,
			// 		message.DirectionIncoming,
			// 		message.StatusReceived,
			// 		conversation.ReferenceTypeMessage,
			// 		tmp.ReferenceID,
			// 		tmp.ID.String(),
			// 		tmp.Text,
			// 		tmp.Medias,
			// 	).Return(tmp, nil)
			// }

			if err := h.eventSMS(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
