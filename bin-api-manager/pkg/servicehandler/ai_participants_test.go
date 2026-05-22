package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amai "monorepo/bin-ai-manager/models/ai"
	amparticipant "monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIcallParticipantGets(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aicallID  uuid.UUID
		pageToken string
		pageSize  uint64

		responseAIcall       *amaicall.AIcall
		responseAIcallErr    error
		responseParticipants []*amparticipant.WebhookMessage

		expectErr bool
		expectRes []*amparticipant.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID:  uuid.FromStringOrNil("b01a3c1a-f31c-11ef-8b45-8782c358d446"),
			pageToken: "2020-09-20T03:23:20.995000Z",
			pageSize:  10,

			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b01a3c1a-f31c-11ef-8b45-8782c358d446"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseAIcallErr: nil,
			responseParticipants: []*amparticipant.WebhookMessage{
				{
					AIID:     uuid.FromStringOrNil("b03f1e2e-f31c-11ef-9fcd-9fde09fbb4e8"),
					AIcallID: uuid.FromStringOrNil("b01a3c1a-f31c-11ef-8b45-8782c358d446"),
				},
			},

			expectErr: false,
			expectRes: []*amparticipant.WebhookMessage{
				{
					AIID:     uuid.FromStringOrNil("b03f1e2e-f31c-11ef-9fcd-9fde09fbb4e8"),
					AIcallID: uuid.FromStringOrNil("b01a3c1a-f31c-11ef-8b45-8782c358d446"),
				},
			},
		},
		{
			name: "aicall_not_found",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID:  uuid.FromStringOrNil("b05c1d3a-f31c-11ef-8b45-8782c358d446"),
			pageToken: "2020-09-20T03:23:20.995000Z",
			pageSize:  10,

			responseAIcall:       nil,
			responseAIcallErr:    fmt.Errorf("not found"),
			responseParticipants: nil,

			expectErr: true,
			expectRes: nil,
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, tt.responseAIcallErr)
			if !tt.expectErr {
				mockReq.EXPECT().AIV1AIcallParticipantList(ctx, tt.aicallID, tt.pageToken, tt.pageSize).Return(tt.responseParticipants, nil)
			}

			res, err := h.AIcallParticipantGets(ctx, tt.agent, tt.aicallID, tt.pageToken, tt.pageSize)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIParticipantGets(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aiID      uuid.UUID
		pageToken string
		pageSize  uint64

		responseAI           *amai.AI
		responseAIErr        error
		responseParticipants []*amparticipant.WebhookMessage

		expectErr bool
		expectRes []*amparticipant.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiID:      uuid.FromStringOrNil("c01a3c1a-f31c-11ef-8b45-8782c358d446"),
			pageToken: "2020-09-20T03:23:20.995000Z",
			pageSize:  10,

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c01a3c1a-f31c-11ef-8b45-8782c358d446"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseAIErr: nil,
			responseParticipants: []*amparticipant.WebhookMessage{
				{
					AIID:     uuid.FromStringOrNil("c01a3c1a-f31c-11ef-8b45-8782c358d446"),
					AIcallID: uuid.FromStringOrNil("c03f1e2e-f31c-11ef-9fcd-9fde09fbb4e8"),
				},
			},

			expectErr: false,
			expectRes: []*amparticipant.WebhookMessage{
				{
					AIID:     uuid.FromStringOrNil("c01a3c1a-f31c-11ef-8b45-8782c358d446"),
					AIcallID: uuid.FromStringOrNil("c03f1e2e-f31c-11ef-9fcd-9fde09fbb4e8"),
				},
			},
		},
		{
			name: "ai_not_found",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiID:      uuid.FromStringOrNil("c05c1d3a-f31c-11ef-8b45-8782c358d446"),
			pageToken: "2020-09-20T03:23:20.995000Z",
			pageSize:  10,

			responseAI:           nil,
			responseAIErr:        fmt.Errorf("not found"),
			responseParticipants: nil,

			expectErr: true,
			expectRes: nil,
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

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, tt.responseAIErr)
			if !tt.expectErr {
				mockReq.EXPECT().AIV1AIParticipantList(ctx, tt.aiID, tt.pageToken, tt.pageSize).Return(tt.responseParticipants, nil)
			}

			res, err := h.AIParticipantGets(ctx, tt.agent, tt.aiID, tt.pageToken, tt.pageSize)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
