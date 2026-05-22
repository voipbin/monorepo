package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amaiprompthistory "monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIPromptHistoryGetsByAIID(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity
		aiID  uuid.UUID
		size  uint64
		token string

		responseAI      *amai.AI
		responseHistory []amaiprompthistory.AIPromptHistory
		expectRes       []*amaiprompthistory.AIPromptHistory
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
			aiID:  uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
			size:  10,
			token: "2020-09-20T03:23:20.995000Z",

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseHistory: []amaiprompthistory.AIPromptHistory{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					},
					AIID:   uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					Prompt: "test prompt v1",
					TMCreate: func() *time.Time {
						t := time.Date(2020, 9, 20, 3, 23, 20, 0, time.UTC)
						return &t
					}(),
				},
			},
			expectRes: []*amaiprompthistory.AIPromptHistory{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					},
					AIID:   uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					Prompt: "test prompt v1",
					TMCreate: func() *time.Time {
						t := time.Date(2020, 9, 20, 3, 23, 20, 0, time.UTC)
						return &t
					}(),
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

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().AIV1AIPromptHistoryList(ctx, tt.aiID, tt.token, tt.size).Return(tt.responseHistory, nil)

			res, err := h.AIPromptHistoryGetsByAIID(ctx, tt.agent, tt.aiID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptHistoryGet(t *testing.T) {
	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aiID      uuid.UUID
		historyID uuid.UUID

		responseAI      *amai.AI
		responseHistory *amaiprompthistory.AIPromptHistory
		expectRes       *amaiprompthistory.AIPromptHistory
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
			aiID:      uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
			historyID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseHistory: &amaiprompthistory.AIPromptHistory{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
				},
				AIID:   uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				Prompt: "test prompt v1",
				TMCreate: func() *time.Time {
					t := time.Date(2020, 9, 20, 3, 23, 20, 0, time.UTC)
					return &t
				}(),
			},
			expectRes: &amaiprompthistory.AIPromptHistory{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
				},
				AIID:   uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				Prompt: "test prompt v1",
				TMCreate: func() *time.Time {
					t := time.Date(2020, 9, 20, 3, 23, 20, 0, time.UTC)
					return &t
				}(),
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

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().AIV1AIPromptHistoryGet(ctx, tt.aiID, tt.historyID).Return(tt.responseHistory, nil)

			res, err := h.AIPromptHistoryGet(ctx, tt.agent, tt.aiID, tt.historyID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptHistoryGetsByAIID_PermissionDenied(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity
		aiID  uuid.UUID

		responseAI *amai.AI
	}{
		{
			name: "permission_denied",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-8e5f-11ee-97b2-cfe7337b701c"), // different customer
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiID: uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"), // original owner
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

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)

			_, err := h.AIPromptHistoryGetsByAIID(ctx, tt.agent, tt.aiID, 10, "")
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_AIPromptHistoryGetsByAIID_DirectAccess(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity
		aiID  uuid.UUID
	}{
		{
			name: "direct_access_not_supported",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				ResourceType:         "ai",
				ResourceID:           uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			aiID: uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
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

			_, err := h.AIPromptHistoryGetsByAIID(ctx, tt.agent, tt.aiID, 10, "")
			if err != serviceerrors.ErrDirectAccessNotSupported {
				t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrDirectAccessNotSupported, err)
			}
		})
	}
}

func Test_AIPromptHistoryGet_DirectAccess(t *testing.T) {
	tests := []struct {
		name      string
		agent     *auth.AuthIdentity
		aiID      uuid.UUID
		historyID uuid.UUID
	}{
		{
			name: "direct_access_not_supported",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				ResourceType:         "ai",
				ResourceID:           uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			aiID:      uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
			historyID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
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

			_, err := h.AIPromptHistoryGet(ctx, tt.agent, tt.aiID, tt.historyID)
			if err != serviceerrors.ErrDirectAccessNotSupported {
				t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrDirectAccessNotSupported, err)
			}
		})
	}
}

func Test_AIPromptHistoryGet_PermissionDenied(t *testing.T) {
	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aiID      uuid.UUID
		historyID uuid.UUID

		responseAI *amai.AI
	}{
		{
			name: "permission_denied",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-8e5f-11ee-97b2-cfe7337b701c"), // different customer
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiID:      uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
			historyID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"), // original owner
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

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)

			_, err := h.AIPromptHistoryGet(ctx, tt.agent, tt.aiID, tt.historyID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
