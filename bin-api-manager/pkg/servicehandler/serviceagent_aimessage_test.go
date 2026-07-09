package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

// Test_ServiceAgentAImessageList verifies a plain Agent permission (not
// Admin/Manager) can list aimessages for an aicall belonging to its own
// customer via the service_agents surface.
func Test_ServiceAgentAImessageList(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aicallID  uuid.UUID
		pageSize  uint64
		pageToken string

		responseAIcall *amaicall.AIcall
		response       []ammessage.Message

		expectFilters map[ammessage.Field]any
		expectRes     []*ammessage.WebhookMessage
	}{
		{
			"agent permission, aicall belongs to the same customer",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			10,
			"2020-10-20T01:00:00.995000Z",

			&amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			[]ammessage.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[ammessage.Field]any{
				ammessage.FieldDeleted:  false,
				ammessage.FieldAIcallID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			},
			[]*ammessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().AIV1MessageGetsByAIcallID(ctx, tt.aicallID, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)

			res, err := h.ServiceAgentAImessageList(ctx, tt.agent, tt.aicallID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAImessageList_TenantIsolation verifies that when the
// aicall's customer does not match the calling agent's own customer, the
// request is rejected -- tenant isolation without any ownership/role check
// beyond that.
func Test_ServiceAgentAImessageList_TenantIsolation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	aicallID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")

	mockReq.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(&amaicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: uuid.FromStringOrNil("6a72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
		},
	}, nil)

	_, err := h.ServiceAgentAImessageList(ctx, agent, aicallID, 10, "")
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}

// Test_ServiceAgentAImessageCreate verifies a plain Agent permission (not
// Admin/Manager) can send an aimessage to an aicall belonging to its own
// customer via the service_agents surface.
func Test_ServiceAgentAImessageCreate(t *testing.T) {

	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aicallID uuid.UUID
		role     ammessage.Role
		content  string

		responseAIcall *amaicall.AIcall
		response       *ammessage.Message

		expectRes *ammessage.WebhookMessage
	}{
		{
			"agent permission, aicall belongs to the same customer",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			ammessage.RoleUser,
			"hello",

			&amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&ammessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				},
			},

			&ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
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

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicallID, tt.role, tt.content, true, false, 30000).Return(tt.response, nil)

			res, err := h.ServiceAgentAImessageCreate(ctx, tt.agent, tt.aicallID, tt.role, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAImessageCreate_TenantIsolation verifies that when the
// aicall's customer does not match the calling agent's own customer, the
// request is rejected -- tenant isolation without any ownership/role check
// beyond that.
func Test_ServiceAgentAImessageCreate_TenantIsolation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	aicallID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")

	mockReq.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(&amaicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: uuid.FromStringOrNil("6a72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
		},
	}, nil)

	_, err := h.ServiceAgentAImessageCreate(ctx, agent, aicallID, ammessage.RoleUser, "hello")
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}
