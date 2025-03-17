package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/chatbot"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_ChatbotCreate(t *testing.T) {

	tests := []struct {
		name string

		chatbot *chatbot.Chatbot

		responseCurTime string

		expectRes *chatbot.Chatbot
	}{
		{
			name: "have all",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeNone,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				InitPrompt: "test init prompt",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeNone,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				InitPrompt: "test init prompt",

				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "empty",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
				EngineData: map[string]any{},
				TMCreate:   "2023-01-03 21:35:02.809",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			if err := h.ChatbotCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChatbotGet(ctx, tt.chatbot.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			res, err := h.ChatbotGet(ctx, tt.chatbot.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotDelete(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbot.Chatbot

		id uuid.UUID

		responseCurTime string
		expectRes       *chatbot.Chatbot
	}{
		{
			name: "normal",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
			},

			id: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
				EngineData: map[string]any{},
				TMCreate:   "2023-01-03 21:35:02.809",
				TMUpdate:   "2023-01-03 21:35:02.809",
				TMDelete:   "2023-01-03 21:35:02.809",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			if err := h.ChatbotCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			if errDel := h.ChatbotDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ChatbotGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			res, err := h.ChatbotGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatbotGets(t *testing.T) {

	tests := []struct {
		name     string
		chatbots []*chatbot.Chatbot

		customerID uuid.UUID
		count      int
		filters    map[string]string

		responseCurTime string
		expectRes       []*chatbot.Chatbot
	}{
		{
			name: "normal",
			chatbots: []*chatbot.Chatbot{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
				},
			},

			customerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			count:      10,
			filters: map[string]string{
				"deleted": "false",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: []*chatbot.Chatbot{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					EngineData: map[string]any{},
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					EngineData: map[string]any{},
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			name:     "empty",
			chatbots: []*chatbot.Chatbot{},

			customerID: uuid.FromStringOrNil("b31d32ae-7f45-11ec-82c6-936e22306376"),
			count:      0,
			filters: map[string]string{
				"deleted": "false",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes:       []*chatbot.Chatbot{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, cf := range tt.chatbots {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
				if errCreate := h.ChatbotCreate(ctx, cf); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ChatbotGets(ctx, tt.customerID, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ChatbotSetInfo(t *testing.T) {

	tests := []struct {
		name    string
		chatbot *chatbot.Chatbot

		id          uuid.UUID
		chatbotName string
		detail      string
		engineType  chatbot.EngineType
		engineModel chatbot.EngineModel
		engineData  map[string]any
		initPrompt  string

		responseCurTime string
		expectRes       *chatbot.Chatbot
	}{
		{
			name: "normal",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
			},

			id:          uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
			chatbotName: "new name",
			detail:      "new detail",
			engineType:  chatbot.EngineTypeNone,
			engineModel: chatbot.EngineModelOpenaiGPT3Dot5Turbo,
			engineData: map[string]any{
				"key1": "val1",
				"key2": 2.0,
			},
			initPrompt: "new init prompt",

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
				Name:        "new name",
				Detail:      "new detail",
				EngineType:  chatbot.EngineTypeNone,
				EngineModel: chatbot.EngineModelOpenaiGPT3Dot5Turbo,
				EngineData: map[string]any{
					"key1": "val1",
					"key2": 2.0,
				},
				InitPrompt: "new init prompt",
				TMCreate:   "2023-01-03 21:35:02.809",
				TMUpdate:   "2023-01-03 21:35:02.809",
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			if err := h.ChatbotCreate(ctx, tt.chatbot); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			if errDel := h.ChatbotSetInfo(ctx, tt.id, tt.chatbotName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.initPrompt); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ChatbotGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChatbotSet(ctx, gomock.Any())
			res, err := h.ChatbotGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
