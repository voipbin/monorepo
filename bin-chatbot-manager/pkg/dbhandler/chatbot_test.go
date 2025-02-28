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

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/pkg/cachehandler"
)

func Test_ChatbotCreate(t *testing.T) {

	tests := []struct {
		name string

		chatbot *chatbot.Chatbot

		responseCurTime string

		expectRes *chatbot.Chatbot
	}{
		{
			"have all",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "test is test init prompt",
			},

			"2023-01-03 21:35:02.809",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("165c9f1e-a5e0-11ed-8521-db074e85944c"),
					CustomerID: uuid.FromStringOrNil("168e154e-a5e0-11ed-b40c-e7bb8f3f9928"),
				},
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "test is test init prompt",

				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
			},

			"2023-01-03 21:35:02.809",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("16bbdc18-a5e0-11ed-8762-5771d36fd113"),
				},
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
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
			"normal",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
			},

			uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),

			"2023-01-03 21:35:02.809",
			&chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5b769ed2-a5e1-11ed-8ad0-5bc10434535b"),
				},
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: "2023-01-03 21:35:02.809",
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
			"normal",
			[]*chatbot.Chatbot{
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

			uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
			10,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*chatbot.Chatbot{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("6d060150-a76d-11ed-9e96-fb09644b04ca"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					TMCreate: "2023-01-03 21:35:02.809",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("ad76ec88-94c9-11ed-9651-df2f9c2178aa"),
						CustomerID: uuid.FromStringOrNil("6d35368c-a76d-11ed-9699-235c9e4a0117"),
					},
					TMCreate: "2023-01-03 21:35:02.809",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*chatbot.Chatbot{},

			uuid.FromStringOrNil("b31d32ae-7f45-11ec-82c6-936e22306376"),
			0,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*chatbot.Chatbot{},
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

		id                  uuid.UUID
		chatbotName         string
		detail              string
		engineType          chatbot.EngineType
		engineModel         chatbot.EngineModel
		initPrompt          string
		credentialBase64    string
		credentialProjectID string

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

			id:                  uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
			chatbotName:         "new name",
			detail:              "new detail",
			engineType:          chatbot.EngineTypeNone,
			engineModel:         chatbot.EngineModelOpenaiGPT3Dot5Turbo,
			initPrompt:          "new init prompt",
			credentialBase64:    "CredentialBASE64",
			credentialProjectID: "543ba65c-ecda-11ef-883f-ab6f1d3a08da",

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8bdc0568-f82e-11ed-9b13-0fb0a7490981"),
				},
				Name:                "new name",
				Detail:              "new detail",
				EngineType:          chatbot.EngineTypeNone,
				EngineModel:         chatbot.EngineModelOpenaiGPT3Dot5Turbo,
				InitPrompt:          "new init prompt",
				CredentialBase64:    "CredentialBASE64",
				CredentialProjectID: "543ba65c-ecda-11ef-883f-ab6f1d3a08da",
				TMCreate:            "2023-01-03 21:35:02.809",
				TMUpdate:            "2023-01-03 21:35:02.809",
				TMDelete:            DefaultTimeStamp,
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
			if errDel := h.ChatbotSetInfo(ctx, tt.id, tt.chatbotName, tt.detail, tt.engineType, tt.engineModel, tt.initPrompt, tt.credentialBase64, tt.credentialProjectID); errDel != nil {
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
