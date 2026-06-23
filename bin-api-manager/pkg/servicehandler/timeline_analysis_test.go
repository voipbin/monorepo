package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_TimelineAnalysisCreate(t *testing.T) {
	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		activeflowID uuid.UUID
		reanalyze    bool

		responseAnalysis *tmanalysis.Analysis

		expectCustomerID uuid.UUID
		expectRes        *tmanalysis.WebhookMessage
	}{
		{
			name: "normal admin",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			activeflowID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000003"),
			reanalyze:    true,

			responseAnalysis: &tmanalysis.Analysis{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000004"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000002"),
				},
				ActiveflowID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000003"),
				Status:       tmanalysis.StatusProgressing,
				Model:        "internal-engine-id",
			},

			expectCustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000002"),
			expectRes: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000004"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000002"),
				},
				ActiveflowID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000003"),
				Status:       tmanalysis.StatusProgressing,
			},
		},
		{
			name: "normal manager",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000011"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000012"),
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			activeflowID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000013"),
			reanalyze:    false,

			responseAnalysis: &tmanalysis.Analysis{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000014"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000012"),
				},
				Status: tmanalysis.StatusCompleted,
			},

			expectCustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000012"),
			expectRes: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000014"),
					CustomerID: uuid.FromStringOrNil("a1110000-0000-0000-0000-000000000012"),
				},
				Status: tmanalysis.StatusCompleted,
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

			mockReq.EXPECT().TimelineV1AnalysisCreate(ctx, tt.expectCustomerID, tt.activeflowID, tt.reanalyze).Return(tt.responseAnalysis, nil)

			res, err := h.TimelineAnalysisCreate(ctx, tt.agent, tt.activeflowID, tt.reanalyze)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TimelineAnalysisCreate_permissionDenied(t *testing.T) {
	// A CustomerAgent (read-only floor) must NOT be able to trigger a paid
	// analysis (review F6: POST requires CustomerAdmin+).
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1120000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a1120000-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	activeflowID := uuid.FromStringOrNil("a1120000-0000-0000-0000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h := serviceHandler{
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}
	ctx := context.Background()

	// no TimelineV1AnalysisCreate expectation: it must never reach the RPC.
	_, err := h.TimelineAnalysisCreate(ctx, agent, activeflowID, false)
	if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
	}
}

func Test_TimelineAnalysisCreate_missingActiveflowID(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1130000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a1130000-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	h := serviceHandler{
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}
	ctx := context.Background()

	_, err := h.TimelineAnalysisCreate(ctx, agent, uuid.Nil, false)
	if !errors.Is(err, serviceerrors.ErrInvalidArgument) {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrInvalidArgument, err)
	}
}

func Test_TimelineAnalysisGet(t *testing.T) {
	// A CustomerAgent (read floor) is allowed to read.
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a2110000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a2110000-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	id := uuid.FromStringOrNil("a2110000-0000-0000-0000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}
	ctx := context.Background()

	responseAnalysis := &tmanalysis.Analysis{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("a2110000-0000-0000-0000-000000000002"),
		},
		Status: tmanalysis.StatusCompleted,
		Model:  "internal-engine-id",
	}
	expectRes := &tmanalysis.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("a2110000-0000-0000-0000-000000000002"),
		},
		Status: tmanalysis.StatusCompleted,
	}

	mockReq.EXPECT().TimelineV1AnalysisGet(ctx, agent.CustomerID, id).Return(responseAnalysis, nil)

	res, err := h.TimelineAnalysisGet(ctx, agent, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_TimelineAnalysisGetsByCustomerID(t *testing.T) {
	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		size         uint64
		token        string
		activeflowID uuid.UUID
		status       tmanalysis.Status

		mockToken        string
		responseAnalyses []tmanalysis.Analysis

		expectFilters map[tmanalysis.Field]any
		expectRes     []*tmanalysis.WebhookMessage
	}{
		{
			name: "with activeflow and status filters",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:         10,
			token:        "2020-09-20T03:23:20.995000Z",
			activeflowID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000009"),
			status:       tmanalysis.StatusCompleted,

			responseAnalyses: []tmanalysis.Analysis{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000003"),
					},
					Status: tmanalysis.StatusCompleted,
				},
			},

			expectFilters: map[tmanalysis.Field]any{
				tmanalysis.FieldActiveflowID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000009"),
				tmanalysis.FieldStatus:       "completed",
			},
			expectRes: []*tmanalysis.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000003"),
					},
					Status: tmanalysis.StatusCompleted,
				},
			},
		},
		{
			name: "no optional filters, empty token resolved",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000011"),
					CustomerID: uuid.FromStringOrNil("a3110000-0000-0000-0000-000000000012"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:         20,
			token:        "",
			activeflowID: uuid.Nil,
			status:       tmanalysis.Status(""),

			mockToken:        "2021-01-01T00:00:00.000000Z",
			responseAnalyses: []tmanalysis.Analysis{},

			expectFilters: map[tmanalysis.Field]any{},
			expectRes:     []*tmanalysis.WebhookMessage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   dbhandler.NewMockDBHandler(mc),
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			expectToken := tt.token
			if tt.token == "" {
				expectToken = tt.mockToken
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.mockToken)
			}

			mockReq.EXPECT().TimelineV1AnalysisList(ctx, tt.agent.CustomerID, expectToken, tt.size, tt.expectFilters).Return(tt.responseAnalyses, nil)

			res, err := h.TimelineAnalysisGetsByCustomerID(ctx, tt.agent, tt.size, tt.token, tt.activeflowID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TimelineAnalysisDelete(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a4110000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a4110000-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	id := uuid.FromStringOrNil("a4110000-0000-0000-0000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}
	ctx := context.Background()

	responseAnalysis := &tmanalysis.Analysis{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("a4110000-0000-0000-0000-000000000002"),
		},
		Status: tmanalysis.StatusCompleted,
	}
	expectRes := &tmanalysis.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("a4110000-0000-0000-0000-000000000002"),
		},
		Status: tmanalysis.StatusCompleted,
	}

	mockReq.EXPECT().TimelineV1AnalysisDelete(ctx, agent.CustomerID, id).Return(responseAnalysis, nil)

	res, err := h.TimelineAnalysisDelete(ctx, agent, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_TimelineAnalysisDelete_permissionDenied(t *testing.T) {
	// CustomerAgent must NOT be able to delete (review F6: DELETE requires CustomerAdmin+).
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a4120000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a4120000-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	id := uuid.FromStringOrNil("a4120000-0000-0000-0000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h := serviceHandler{
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
		dbHandler:   dbhandler.NewMockDBHandler(mc),
		utilHandler: utilhandler.NewMockUtilHandler(mc),
	}
	ctx := context.Background()

	_, err := h.TimelineAnalysisDelete(ctx, agent, id)
	if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
	}
}
