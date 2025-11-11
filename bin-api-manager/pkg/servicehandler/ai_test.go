package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amai "monorepo/bin-ai-manager/models/ai"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AICreate(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		aiName      string
		detail      string
		engineType  amai.EngineType
		engineModel amai.EngineModel
		engineData  map[string]any
		engineKey   string
		initPrompt  string
		ttsType     amai.TTSType
		ttsVoiceID  string
		sttType     amai.STTType

		response  *amai.AI
		expectRes *amai.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			aiName:      "test name",
			detail:      "test detail",
			engineType:  amai.EngineTypeNone,
			engineModel: amai.EngineModelOpenaiGPT4,
			engineData: map[string]any{
				"key1": "val1",
			},
			engineKey:  "test-engine-key",
			initPrompt: "test init prompt",
			ttsType:    amai.TTSTypeElevenLabs,
			ttsVoiceID: "test-voice-id",
			sttType:    amai.STTTypeDeepgram,

			response: &amai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
				},
			},
			expectRes: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AIV1AICreate(
				ctx,
				tt.agent.CustomerID,
				tt.aiName,
				tt.detail,
				tt.engineType,
				tt.engineModel,
				tt.engineData,
				tt.engineKey,
				tt.initPrompt,
				tt.ttsType,
				tt.ttsVoiceID,
				tt.sttType,
			).Return(tt.response, nil)

			res, err := h.AICreate(
				ctx,
				tt.agent,
				tt.aiName,
				tt.detail,
				tt.engineType,
				tt.engineModel,
				tt.engineData,
				tt.engineKey,
				tt.initPrompt,
				tt.ttsType,
				tt.ttsVoiceID,
				tt.sttType,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_AIGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []amai.AI
		expectRes []*amai.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",
			filters: map[string]string{
				"deleted":     "false",
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
			},

			response: []amai.AI{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					},
				},
			},
			expectRes: []*amai.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGets(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AIGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		aiID  uuid.UUID

		response  *amai.AI
		expectRes *amai.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			&amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.response, nil)

			res, err := h.AIGet(ctx, tt.agent, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		aiID  uuid.UUID

		responseChat *amai.AI
		expectRes    *amai.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseChat, nil)
			mockReq.EXPECT().AIV1AIDelete(ctx, tt.aiID).Return(tt.responseChat, nil)

			res, err := h.AIDelete(ctx, tt.agent, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
