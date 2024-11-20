package chatbotcallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID        uuid.UUID
		chatbotID         uuid.UUID
		chatbotEngineType chatbot.EngineType
		activeflowID      uuid.UUID
		referenceType     chatbotcall.ReferenceType
		referenceID       uuid.UUID
		confbridgeID      uuid.UUID
		gender            chatbotcall.Gender
		language          string

		responseUUID        uuid.UUID
		responseChatbotcall *chatbotcall.Chatbotcall

		expectChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			name: "have all",

			customerID:        uuid.FromStringOrNil("81880ddc-a707-11ed-be35-87b2fee31bb7"),
			chatbotID:         uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
			chatbotEngineType: chatbot.EngineTypeChatGPT,
			activeflowID:      uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
			referenceType:     chatbotcall.ReferenceTypeCall,
			referenceID:       uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
			confbridgeID:      uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
			gender:            chatbotcall.GenderFemale,
			language:          "en-US",

			responseUUID: uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
			responseChatbotcall: &chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
			},

			expectChatbotcall: &chatbotcall.Chatbotcall{
				ID:                uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
				CustomerID:        uuid.FromStringOrNil("81880ddc-a707-11ed-be35-87b2fee31bb7"),
				ChatbotID:         uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				ActiveflowID:      uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
				ConfbridgeID:      uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Status:            chatbotcall.StatusInitiating,
				Messages:          []chatbotcall.Message{},
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUID).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			res, err := h.Create(ctx, tt.customerID, tt.chatbotID, tt.chatbotEngineType, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("5154c3f6-a709-11ed-b011-c7644d9b5fc9"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("5154c3f6-a709-11ed-b011-c7644d9b5fc9"),
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

			mockDB.EXPECT().ChatbotcallGet(ctx, tt.id).Return(tt.responseChatbotcall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a665825e-a709-11ed-967e-538651691d20"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("a964a98a-a709-11ed-ad69-3ff036631417"),
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

			mockDB.EXPECT().ChatbotcallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseChatbotcall, nil)

			res, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_GetByTranscribeID(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("c590415a-a709-11ed-b130-eba649c97eab"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("c5b76622-a709-11ed-8d54-63813a022d9a"),
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

			mockDB.EXPECT().ChatbotcallGetByTranscribeID(ctx, tt.transcribeID).Return(tt.responseChatbotcall, nil)

			res, err := h.GetByTranscribeID(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_UpdateStatusStart(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		transcribeID uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3447ddd8-a70a-11ed-8b76-43164266fbb2"),
			uuid.FromStringOrNil("3470b852-a70a-11ed-9d3f-7feaeeaa417b"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("3447ddd8-a70a-11ed-8b76-43164266fbb2"),
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

			mockDB.EXPECT().ChatbotcallUpdateStatusProgressing(ctx, tt.id, tt.transcribeID).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.id).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallProgressing, tt.responseChatbotcall)

			res, err := h.UpdateStatusStart(ctx, tt.id, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_UpdateStatusEnd(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a3c338ec-a70a-11ed-b305-9bd0df7c9474"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("a3c338ec-a70a-11ed-b305-9bd0df7c9474"),
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

			mockDB.EXPECT().ChatbotcallUpdateStatusEnd(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.id).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallEnd, tt.responseChatbotcall)

			res, err := h.UpdateStatusEnd(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbotcall *chatbotcall.Chatbotcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("301029b6-578f-41c4-905a-906e4e8ebbb3"),

			&chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("301029b6-578f-41c4-905a-906e4e8ebbb3"),
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

			h := &chatbotcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotcallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.id).Return(tt.responseChatbotcall, nil)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		filters    map[string]string

		responseChatbotcalls []*chatbotcall.Chatbotcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("1694a6ac-b485-11ee-9900-ff6bfeb9a3cc"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"deleted": "false",
			},

			responseChatbotcalls: []*chatbotcall.Chatbotcall{
				{
					ID: uuid.FromStringOrNil("16f0c4f0-b485-11ee-81fd-6fe39701733c"),
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

			h := &chatbotcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotcallGets(ctx, tt.customerID, tt.size, tt.token, tt.filters).Return(tt.responseChatbotcalls, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbotcalls, res)
			}
		})
	}
}
