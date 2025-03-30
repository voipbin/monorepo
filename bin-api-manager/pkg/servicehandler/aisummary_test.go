package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	amsummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-api-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AISummaryCreate_referencetype_call(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		onEndFlowID   uuid.UUID
		referenceType amsummary.ReferenceType
		referenceID   uuid.UUID
		language      string

		responseCall    *cmcall.Call
		responseSummary *amsummary.Summary

		expectRes *amsummary.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("de017e84-0cc9-11f0-a6dd-cb4073c0bd22"),
					CustomerID: uuid.FromStringOrNil("de33d3a2-0cc9-11f0-a377-37b9ae47ee38"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			onEndFlowID:   uuid.FromStringOrNil("dec1600a-0cc9-11f0-b561-0f25e91332c8"),
			referenceType: amsummary.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("deefe77c-0cc9-11f0-9a60-f338d9acb104"),
			language:      "en-US",

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("deefe77c-0cc9-11f0-9a60-f338d9acb104"),
					CustomerID: uuid.FromStringOrNil("de33d3a2-0cc9-11f0-a377-37b9ae47ee38"),
				},
				TMDelete: defaultTimestamp,
			},
			responseSummary: &amsummary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de81959c-0cc9-11f0-a170-bbf2ef66b6bc"),
				},
			},

			expectRes: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de81959c-0cc9-11f0-a170-bbf2ef66b6bc"),
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockReq.EXPECT().AIV1SummaryCreate(
				ctx,
				tt.agent.CustomerID,
				uuid.Nil,
				tt.onEndFlowID,
				tt.referenceType,
				tt.referenceID,
				tt.language,
				gomock.Any(),
			).Return(tt.responseSummary, nil)

			res, err := h.AISummaryCreate(ctx, tt.agent, tt.onEndFlowID, tt.referenceType, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AISummaryGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []amsummary.Summary
		expectRes []*amsummary.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1fe023c2-0ccb-11f0-919d-b3e9faacb57a"),
					CustomerID: uuid.FromStringOrNil("2017e1fe-0ccb-11f0-9c4f-73268b39a2cc"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",
			filters: map[string]string{
				"deleted":     "false",
				"customer_id": "2017e1fe-0ccb-11f0-9c4f-73268b39a2cc",
			},

			response: []amsummary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("20698ef0-0ccb-11f0-bd3f-278de4a3e853"),
					},
				},
			},
			expectRes: []*amsummary.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("20698ef0-0ccb-11f0-bd3f-278de4a3e853"),
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

			mockReq.EXPECT().AIV1SummaryGets(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AISummaryGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AISummaryGet(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		aisummaryID uuid.UUID

		response  *amsummary.Summary
		expectRes *amsummary.WebhookMessage
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
			aisummaryID: uuid.FromStringOrNil("209b8da6-0ccb-11f0-a7ac-23a112c89568"),

			response: &amsummary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("209b8da6-0ccb-11f0-a7ac-23a112c89568"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("209b8da6-0ccb-11f0-a7ac-23a112c89568"),
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

			mockReq.EXPECT().AIV1SummaryGet(ctx, tt.aisummaryID).Return(tt.response, nil)

			res, err := h.AISummaryGet(ctx, tt.agent, tt.aisummaryID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AISummaryDelete(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		aisummaryID uuid.UUID

		responseAISummary *amsummary.Summary
		expectRes         *amsummary.WebhookMessage
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
			aisummaryID: uuid.FromStringOrNil("b54b6336-0ccb-11f0-818d-07adf86344ed"),

			responseAISummary: &amsummary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b54b6336-0ccb-11f0-818d-07adf86344ed"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b54b6336-0ccb-11f0-818d-07adf86344ed"),
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

			mockReq.EXPECT().AIV1SummaryGet(ctx, tt.aisummaryID).Return(tt.responseAISummary, nil)
			mockReq.EXPECT().AIV1SummaryDelete(ctx, tt.aisummaryID).Return(tt.responseAISummary, nil)

			res, err := h.AISummaryDelete(ctx, tt.agent, tt.aisummaryID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
