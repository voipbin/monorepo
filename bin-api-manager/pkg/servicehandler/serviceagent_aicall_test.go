package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmkase "monorepo/bin-contact-manager/models/kase"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
)

// Test_ServiceAgentAIcallList verifies a plain Agent permission (not
// Admin/Manager) can list its own customer's aicalls via the service_agents
// surface, optionally scoped by reference_type/reference_id.
func Test_ServiceAgentAIcallList(t *testing.T) {

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		pageToken     string
		pageSize      uint64
		referenceType string
		referenceID   uuid.UUID
		status        string

		response []amaicall.AIcall

		expectFilters map[amaicall.Field]any
		expectRes     []*amaicall.WebhookMessage
	}{
		{
			"agent permission, no reference filter",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"",
			uuid.Nil,
			"",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:    false,
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},
		},
		{
			"agent permission, filtered by reference_type and reference_id",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"contact_case",
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			"",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID:    uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:       false,
				amaicall.FieldReferenceType: "contact_case",
				amaicall.FieldReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},
		},
		{
			"agent permission, filtered by reference_type, reference_id, and status",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"contact_case",
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			"progressing",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID:    uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:       false,
				amaicall.FieldReferenceType: "contact_case",
				amaicall.FieldReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				amaicall.FieldStatus:        "progressing",
			},
			[]*amaicall.WebhookMessage{
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

			mockReq.EXPECT().AIV1AIcallList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)
			res, err := h.ServiceAgentAIcallList(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.referenceType, tt.referenceID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAIcallCreate verifies a plain Agent permission (not
// Admin/Manager) can create an aicall belonging to its own customer via the
// service_agents surface, and that an activeflow is created automatically.
func Test_ServiceAgentAIcallCreate(t *testing.T) {

	type test struct {
		name string

		agent          *auth.AuthIdentity
		assistanceType amaicall.AssistanceType
		assistanceID   uuid.UUID
		referenceType  amaicall.ReferenceType
		referenceID    uuid.UUID

		// Task 5: only consulted when referenceType == ReferenceTypeContactCase.
		responseCase    *cmkase.Case
		responseCaseErr error

		// Task 6: only consulted when assistanceID == uuid.Nil.
		// responseActiveAIList is what the dedicated is_insight_active=true
		// size-1 query returns; nil/empty exercises the zero-active fallback to
		// the most-recently-created Insight AI (responseAIList).
		responseActiveAIList []amai.AI
		responseAIList       []amai.AI

		responseAI         *amai.AI
		responseActiveflow *fmactiveflow.Activeflow
		responseAIcall     *amaicall.AIcall

		// Task 7: when true, AIV1AIcallStart's returned aicall's ActiveflowID
		// is set (in the test loop below) to something other than
		// responseActiveflow.ID, and FlowV1ActiveflowDelete(responseActiveflow.ID)
		// must fire.
		expectReused bool

		expectErr error
		expectRes *amaicall.WebhookMessage
	}

	tests := []test{
		{
			name: "agent permission, ai assistance",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			// Task 5: pre-existing case, needs a same-customer responseCase now
			// that the ownership check runs unconditionally for contact_case.
			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000010"),
				},
			},
			// ActiveflowID matches responseActiveflow.ID above -- this is the
			// genuine-create path, so Task 7's tmp.ActiveflowID != af.ID check
			// must evaluate false here.
			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000010"),
			},

			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000010"),
			},
		},
		{
			name: "Task 5: contact_case reference_id belongs to a different customer -- rejected",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("70000000-0001-11f0-1111-000000000001"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("70000000-0002-11f0-1111-000000000001"),

			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("70000000-0002-11f0-1111-000000000001"),
				CustomerID: uuid.FromStringOrNil("70000000-9999-11f0-1111-000000000001"), // NOT the caller's customer
			},

			expectErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "contact_case lookup fails -- returns lookup error, not ErrPermissionDenied",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("70000000-0003-11f0-1111-000000000002"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("70000000-0001-11f0-1111-000000000002"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("70000000-0002-11f0-1111-000000000002"),

			responseCaseErr: errors.New("contact-manager RPC timeout"),

			expectErr: errors.New("contact-manager RPC timeout"),
		},
		{
			name: "Task 6: assistance_id omitted, contact_case, exactly one insight AI -- resolved",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.Nil,
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("80000000-0002-11f0-2222-000000000001"),

			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("80000000-0002-11f0-2222-000000000001"),
				CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001"),
			},
			responseAIList: []amai.AI{
				{Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("80000000-0004-11f0-2222-000000000001"),
					CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001"),
				}, Type: amai.TypeInsight},
			},
			responseAI: &amai.AI{Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("80000000-0004-11f0-2222-000000000001"),
				CustomerID: uuid.FromStringOrNil("80000000-0003-11f0-2222-000000000001"),
			}},
			responseActiveflow: &fmactiveflow.Activeflow{Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("80000000-0007-11f0-2222-000000000001"),
			}},
			// ActiveflowID matches responseActiveflow.ID above -- genuine
			// create, not reuse.
			responseAIcall: &amaicall.AIcall{
				Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("80000000-0008-11f0-2222-000000000001")},
				ActiveflowID: uuid.FromStringOrNil("80000000-0007-11f0-2222-000000000001"),
			},

			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("80000000-0008-11f0-2222-000000000001"),
				},
				ActiveflowID: uuid.FromStringOrNil("80000000-0007-11f0-2222-000000000001"),
			},
		},
		{
			name: "Task 6: assistance_id omitted, contact_case, zero insight AIs -- not found",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("80000000-0006-11f0-2222-000000000001"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.Nil,
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("80000000-0005-11f0-2222-000000000001"),

			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("80000000-0005-11f0-2222-000000000001"),
				CustomerID: uuid.FromStringOrNil("80000000-0006-11f0-2222-000000000001"),
			},
			responseAIList: []amai.AI{},

			expectErr: serviceerrors.ErrNotFound,
		},
		{
			name: "Task 7: AIV1AIcallStart returns a reused aicall -- orphaned activeflow is deleted",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("90000000-0003-11f0-3333-000000000001"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("90000000-0001-11f0-3333-000000000001"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("90000000-0002-11f0-3333-000000000001"),

			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("90000000-0002-11f0-3333-000000000001"),
				CustomerID: uuid.FromStringOrNil("90000000-0003-11f0-3333-000000000001"),
			},
			responseAI: &amai.AI{Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("90000000-0001-11f0-3333-000000000001"),
				CustomerID: uuid.FromStringOrNil("90000000-0003-11f0-3333-000000000001"),
			}},
			responseActiveflow: &fmactiveflow.Activeflow{Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("90000000-0004-11f0-3333-000000000001"),
			}},
			responseAIcall: &amaicall.AIcall{
				Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("90000000-0005-11f0-3333-000000000001")},
				ActiveflowID: uuid.FromStringOrNil("90000000-9999-11f0-3333-000000000001"), // a DIFFERENT, pre-existing activeflow -- NOT responseActiveflow.ID
			},
			expectReused: true,

			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("90000000-0005-11f0-3333-000000000001"),
				},
				ActiveflowID: uuid.FromStringOrNil("90000000-9999-11f0-3333-000000000001"),
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

			if tt.referenceType == amaicall.ReferenceTypeContactCase {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.referenceID).Return(tt.responseCase, tt.responseCaseErr)
			}

			if tt.responseCaseErr != nil {
				_, err := h.ServiceAgentAIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				if errors.Is(err, serviceerrors.ErrPermissionDenied) {
					t.Errorf("Wrong match. expect: a non-403 lookup error (ContactV1CaseGet failure must not be classified as ErrPermissionDenied), got: %v", err)
				}
				return
			}

			if tt.expectErr == serviceerrors.ErrPermissionDenied && tt.responseCase.CustomerID != tt.agent.CustomerID {
				_, err := h.ServiceAgentAIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
					t.Errorf("Wrong match. expect: ErrPermissionDenied, got: %v", err)
				}
				return
			}

			resolvedAssistanceID := tt.assistanceID
			if tt.assistanceType == amaicall.AssistanceTypeAI && tt.assistanceID == uuid.Nil {
				// Two-query resolution: the dedicated size-1 active query runs
				// first, and only an empty result falls back to the 100-row
				// most-recently-created query.
				mockReq.EXPECT().AIV1AIList(ctx, "", uint64(1), gomock.Any()).Return(tt.responseActiveAIList, nil)
				if len(tt.responseActiveAIList) > 0 {
					resolvedAssistanceID = tt.responseActiveAIList[0].ID
				} else {
					mockReq.EXPECT().AIV1AIList(ctx, "", uint64(100), gomock.Any()).Return(tt.responseAIList, nil)
					if len(tt.responseAIList) == 0 {
						_, err := h.ServiceAgentAIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
						if err == nil {
							t.Errorf("Wrong match. expect: err, got: ok")
						}
						return
					}
					resolvedAssistanceID = tt.responseAIList[0].ID
				}
			}

			switch tt.assistanceType {
			case amaicall.AssistanceTypeAI:
				mockReq.EXPECT().AIV1AIGet(ctx, resolvedAssistanceID).Return(tt.responseAI, nil)
			case amaicall.AssistanceTypeTeam:
				mockReq.EXPECT().AIV1TeamGet(ctx, resolvedAssistanceID).Return(nil, nil)
			}

			mockReq.EXPECT().FlowV1ActiveflowCreate(
				ctx, uuid.Nil, tt.agent.CustomerID, uuid.Nil, fmactiveflow.ReferenceTypeAPI,
				uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(tt.responseActiveflow, nil)

			mockReq.EXPECT().AIV1AIcallStart(
				ctx, tt.assistanceType, resolvedAssistanceID, tt.responseActiveflow.ID, tt.referenceType, tt.referenceID,
			).Return(tt.responseAIcall, nil)

			if tt.expectReused {
				mockReq.EXPECT().FlowV1ActiveflowDelete(ctx, tt.responseActiveflow.ID).Return(nil, nil)
			}

			res, err := h.ServiceAgentAIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAIcallCreate_TenantIsolation verifies that when the
// resolved assistance entity's customer does not match the calling agent's
// own customer, the request is rejected -- tenant isolation without any
// ownership/role check beyond that.
func Test_ServiceAgentAIcallCreate_TenantIsolation(t *testing.T) {
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
	assistanceID := uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46")

	mockReq.EXPECT().ContactV1CaseGet(ctx, agent.CustomerID, uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a")).Return(&cmkase.Case{
		ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
		CustomerID: agent.CustomerID,
	}, nil)

	mockReq.EXPECT().AIV1AIGet(ctx, assistanceID).Return(&amai.AI{
		Identity: commonidentity.Identity{
			ID:         assistanceID,
			CustomerID: uuid.FromStringOrNil("other-customer-does-not-match"),
		},
	}, nil)

	_, err := h.ServiceAgentAIcallCreate(ctx, agent, amaicall.AssistanceTypeAI, assistanceID, amaicall.ReferenceTypeContactCase, uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"))
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}

// Test_ServiceAgentAIcallListen covers the endpoint's authorization contract
// (design §7 item 2).
//
// The permission row is the important one and is the direct regression test for
// design review round 13's BLOCKING-1: this endpoint MUST gate on the AGENT
// tier (amagent.PermissionAll), not the admin/manager bitmask the top-level
// AIcallTerminate uses. At the admin tier an ordinary agent in square-talk
// would get ErrPermissionDenied and listening would never start in the
// feature's actual primary use case.
func Test_ServiceAgentAIcallListen(t *testing.T) {
	aicallID := uuid.FromStringOrNil("2b7c1d90-8f3a-11f0-9c5e-3f1a2b3c4d5e")
	customerID := uuid.FromStringOrNil("2b7c1d90-8f3a-11f0-9c5e-3f1a2b3c4d5f")

	t.Run("non-agent identity is refused with zero RPC calls", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &serviceHandler{reqHandler: mockReq}
		ctx := context.Background()

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), gomock.Any()).Times(0)
		mockReq.EXPECT().AIV1AIcallListen(gomock.Any(), gomock.Any()).Times(0)

		// A nil-agent identity is not an agent.
		_, err := h.ServiceAgentAIcallListen(ctx, &auth.AuthIdentity{}, aicallID)
		if !errors.Is(err, serviceerrors.ErrAuthenticationRequired) {
			t.Errorf("expected ErrAuthenticationRequired. got: %v", err)
		}
	})

	t.Run("cross-customer aicall is rejected before AIV1AIcallListen is called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
		ctx := context.Background()

		agent := auth.NewAgentIdentity(&amagent.Agent{
			Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
			Permission: amagent.PermissionCustomerAgent,
		})

		// Fetched, but owned by somebody else.
		mockReq.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(&amaicall.AIcall{
			Identity: commonidentity.Identity{
				ID:         aicallID,
				CustomerID: uuid.FromStringOrNil("2b7c1d90-8f3a-11f0-9c5e-3f1a2b3c4d60"),
			},
		}, nil)
		mockReq.EXPECT().AIV1AIcallListen(gomock.Any(), gomock.Any()).Times(0)

		_, err := h.ServiceAgentAIcallListen(ctx, agent, aicallID)
		if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
			t.Errorf("expected ErrPermissionDenied. got: %v", err)
		}
	})

	t.Run("an ordinary agent on its own aicall reaches AIV1AIcallListen", func(t *testing.T) {
		// PermissionCustomerAgent is the ordinary agent tier. If this method
		// ever gated on the admin/manager bitmask instead, this row would fail
		// with ErrPermissionDenied -- which is exactly BLOCKING-1.
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
		ctx := context.Background()

		agent := auth.NewAgentIdentity(&amagent.Agent{
			Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
			Permission: amagent.PermissionCustomerAgent,
		})

		owned := &amaicall.AIcall{
			Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID},
		}
		mockReq.EXPECT().AIV1AIcallGet(ctx, aicallID).Return(owned, nil)
		mockReq.EXPECT().AIV1AIcallListen(ctx, aicallID).Return(owned, nil).Times(1)

		res, err := h.ServiceAgentAIcallListen(ctx, agent, aicallID)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res == nil || res.ID != aicallID {
			t.Errorf("expected the aicall webhook message back. got: %v", res)
		}
	})
}
