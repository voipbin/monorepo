package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AIcallCreate(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		aiID          uuid.UUID
		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID
		gender        amaicall.Gender
		language      string

		responseAI     *amai.AI
		responseAIcall *amaicall.AIcall

		expectRes *amaicall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			aiID:          uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			referenceType: amaicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
			gender:        amaicall.GenderMale,
			language:      "en-US",

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
			},

			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().AIV1AIcallStart(ctx, tt.aiID, tt.referenceType, tt.referenceID, tt.gender, tt.language).Return(tt.responseAIcall, nil)

			res, err := h.AIcallCreate(ctx, tt.agent, tt.aiID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []amaicall.AIcall
		expectRes []*amaicall.WebhookMessage
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
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted":     "false",
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
			},

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
					},
				},
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
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

			mockReq.EXPECT().AIV1AIcallGets(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AIcallGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallGet(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		aicallID uuid.UUID

		response  *amaicall.AIcall
		expectRes *amaicall.WebhookMessage
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
			uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),

			&amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.response, nil)

			res, err := h.AIcallGet(ctx, tt.agent, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallDelete(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		aicallID uuid.UUID

		responseChat *amaicall.AIcall
		expectRes    *amaicall.WebhookMessage
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

			&amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseChat, nil)
			mockReq.EXPECT().AIV1AIcallDelete(ctx, tt.aicallID).Return(tt.responseChat, nil)

			res, err := h.AIcallDelete(ctx, tt.agent, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
