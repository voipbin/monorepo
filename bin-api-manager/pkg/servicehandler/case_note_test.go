package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CaseNoteList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("6f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		responseCase  *cmkase.Case
		responseNotes []*cmcasenote.CaseNote
		expectErr     bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			responseNotes: []*cmcasenote.CaseNote{},
			expectErr:     false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: true,
		},
		{
			// The case belongs to otherCustomerID. caseGet returns the
			// same not-found error a genuinely missing case would (see
			// case.go's caseGet doc comment) -- ContactV1CaseGet is
			// tenant-checked on the contact-manager side, and hasPermission
			// is never reached because caseGet fails first.
			name: "cross-tenant case returns not found, never reaches permission check",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: otherCustomerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCase: nil,
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			if tt.responseCase != nil {
				mockReq.EXPECT().
					ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).
					Return(tt.responseCase, nil)
			} else {
				mockReq.EXPECT().
					ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).
					Return(nil, serviceerrors.ErrNotFound)
			}

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1CaseNoteList(ctx, tt.agent.CustomerID, tt.caseID).
					Return(tt.responseNotes, nil)
			}

			res, err := h.CaseNoteList(ctx, tt.agent, tt.caseID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if res == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_CaseNoteCreate(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	noteID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		caseID     uuid.UUID
		authorType string
		authorID   *uuid.UUID
		text       string

		responseCase *cmkase.Case
		responseNote *cmcasenote.CaseNote
		expectErr    bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:     caseID,
			authorType: cmcasenote.AuthorTypeAgent,
			authorID:   &agentID,
			text:       "Called the customer back, no answer.",

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			responseNote: &cmcasenote.CaseNote{
				ID:         noteID,
				CustomerID: customerID,
				CaseID:     caseID,
				AuthorType: cmcasenote.AuthorTypeAgent,
				AuthorID:   &agentID,
				Text:       "Called the customer back, no answer.",
			},
			expectErr: false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:     caseID,
			authorType: cmcasenote.AuthorTypeAgent,
			authorID:   &agentID,
			text:       "note",

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().
				ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).
				Return(tt.responseCase, nil)

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1CaseNoteCreate(ctx, tt.agent.CustomerID, tt.caseID, tt.authorType, tt.authorID, tt.text).
					Return(tt.responseNote, nil)
			}

			res, err := h.CaseNoteCreate(ctx, tt.agent, tt.caseID, tt.authorType, tt.authorID, tt.text)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if res == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_CaseNoteDelete(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	noteID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID
		noteID uuid.UUID

		responseCase *cmkase.Case
		expectErr    bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,
			noteID: noteID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,
			noteID: noteID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().
				ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).
				Return(tt.responseCase, nil)

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1CaseNoteDelete(ctx, tt.agent.CustomerID, tt.caseID, tt.noteID).
					Return(nil)
			}

			err := h.CaseNoteDelete(ctx, tt.agent, tt.caseID, tt.noteID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Test_CaseNoteList_DirectAccessDenied confirms direct-access callers
// (accesskey auth, no agent identity) are rejected before any RPC call,
// mirroring case.go's CaseGet/CaseClose/CaseContinue convention.
func Test_CaseNoteList_DirectAccessDenied(t *testing.T) {
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")})

	_, err := h.CaseNoteList(ctx, a, caseID)
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("Expected ErrDirectAccessNotSupported, got: %v", err)
	}
}
