package chatbothandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatbotName string
		detail      string
		engineType  chatbot.EngineType

		responseUUID    uuid.UUID
		responseChatbot *chatbot.Chatbot

		expectChatbot *chatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
			"test name",
			"test detail",
			chatbot.EngineTypeChatGPT,

			uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
			},

			&chatbot.Chatbot{
				ID:         uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
				CustomerID: uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeChatGPT,
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

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().ChatbotCreate(ctx, tt.expectChatbot).Return(nil)
			mockDB.EXPECT().ChatbotGet(ctx, tt.responseUUID).Return(tt.responseChatbot, nil)

			res, err := h.Create(ctx, tt.customerID, tt.chatbotName, tt.detail, tt.engineType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbot *chatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),
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

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotGet(ctx, tt.id).Return(tt.responseChatbot, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbot *chatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),
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

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatbotGet(ctx, tt.id).Return(tt.responseChatbot, nil)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}
