package chatbotcallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/service"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbothandler"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatgpthandler"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_ServiceStart(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		chatbotID     uuid.UUID
		referenceType chatbotcall.ReferenceType
		referenceID   uuid.UUID
		gender        chatbotcall.Gender
		language      string

		responseConfbridge      *cmconfbridge.Confbridge
		responseUUIDChatbotcall uuid.UUID
		responseChatbotcall     *chatbotcall.Chatbotcall
		responseUUIDAction      uuid.UUID

		expectChatbotcall *chatbotcall.Chatbotcall
		expectRes         *service.Service
	}{
		{
			"normal",

			uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
			uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
			chatbotcall.ReferenceTypeCall,
			uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
			chatbotcall.GenderFemale,
			"en-US",

			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			},
			uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

			&chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				CustomerID:    uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				ChatbotID:     uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:        chatbotcall.GenderFemale,
				Language:      "en-US",
				Status:        chatbotcall.StatusInitiating,
			},
			&service.Service{
				ID:   uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				Type: service.TypeChatbotcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),
						Type:   fmaction.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"ec6d153d-dd5a-4eef-bc27-8fcebe100704"}`),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := chatgpthandler.NewMockChatgptHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				chatgptHandler: mockChatgpt,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.customerID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDAction)

			res, err := h.ServiceStart(ctx, tt.customerID, tt.chatbotID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
