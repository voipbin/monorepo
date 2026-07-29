package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentCaseNoteList(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	responseCase := &cmkase.Case{
		ID:         caseID,
		CustomerID: customerID,
	}
	responseNotes := []*cmcasenote.CaseNote{
		{
			ID:     uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
			CaseID: caseID,
			Text:   "note text",
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	mockReq.EXPECT().ContactV1CaseGet(ctx, agent.CustomerID, caseID).Return(responseCase, nil)
	mockReq.EXPECT().ContactV1CaseNoteList(ctx, agent.CustomerID, caseID).Return(responseNotes, nil)

	res, err := h.ServiceAgentCaseNoteList(ctx, agent, caseID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if reflect.DeepEqual(res, responseNotes) != true {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", responseNotes, res)
	}
}

func Test_ServiceAgentCaseNoteCreate(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	type test struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID
		text   string

		responseCaseGet *cmkase.Case
		responseNote    *cmcasenote.CaseNote

		expectCaseGetCall bool
		expectCreateCall  bool
		expectRes         *cmcasenote.CaseNote
		expectErr         bool
	}

	tests := []test{
		{
			// The author is derived server-side from the caller's own
			// agent identity -- the caller never supplies author_type/
			// author_id.
			name: "agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,
			text:   "Called the customer back, no answer.",

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
			},
			responseNote: &cmcasenote.CaseNote{
				ID:         uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
				CaseID:     caseID,
				AuthorType: cmcasenote.AuthorTypeAgent,
				AuthorID:   &agentID,
				Text:       "Called the customer back, no answer.",
			},

			expectCaseGetCall: true,
			expectCreateCall:  true,
			expectRes: &cmcasenote.CaseNote{
				ID:         uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
				CaseID:     caseID,
				AuthorType: cmcasenote.AuthorTypeAgent,
				AuthorID:   &agentID,
				Text:       "Called the customer back, no answer.",
			},
		},
		{
			// direct identities have no AgentID(), so they must be
			// rejected before any downstream call.
			name: "direct identity rejected",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID: customerID,
			}),
			caseID: caseID,
			text:   "text",

			expectCaseGetCall: false,
			expectCreateCall:  false,
			expectErr:         true,
		},
		{
			// An accesskey identity satisfies hasPermission(PermissionAll)
			// (accesskeys get admin-equivalent access) but AgentID() would
			// return uuid.Nil for it, which would make note authorship
			// unattributable to a real agent -- must be rejected too, not
			// just direct identities.
			name: "accesskey identity rejected",
			agent: auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{
				CustomerID: customerID,
			}),
			caseID: caseID,
			text:   "text",

			expectCaseGetCall: false,
			expectCreateCall:  false,
			expectErr:         true,
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

			if tt.expectCaseGetCall {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)
			}
			if tt.expectCreateCall {
				expectedAgentID := tt.agent.AgentID()
				mockReq.EXPECT().ContactV1CaseNoteCreate(ctx, tt.agent.CustomerID, tt.caseID, cmcasenote.AuthorTypeAgent, &expectedAgentID, tt.text).Return(tt.responseNote, nil)
			}

			res, err := h.ServiceAgentCaseNoteCreate(ctx, tt.agent, tt.caseID, tt.text)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentCaseNoteDelete(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID := uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	noteID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAgent,
	})

	responseCaseGet := &cmkase.Case{
		ID:         caseID,
		CustomerID: customerID,
	}

	type test struct {
		name string

		agent *auth.AuthIdentity

		responseNotes []*cmcasenote.CaseNote

		expectCaseGetCall bool
		expectListCall    bool
		expectDeleteCall  bool
		expectErr         bool
	}

	tests := []test{
		{
			// The caller authored the note -- delete is allowed.
			name:  "agent owns the note",
			agent: agent,
			responseNotes: []*cmcasenote.CaseNote{
				{ID: noteID, CaseID: caseID, AuthorID: &agentID},
			},
			expectCaseGetCall: true,
			expectListCall:    true,
			expectDeleteCall:  true,
		},
		{
			// The note was authored by a different agent -- must be
			// rejected, not deleted.
			name:  "note authored by a different agent",
			agent: agent,
			responseNotes: []*cmcasenote.CaseNote{
				{ID: noteID, CaseID: caseID, AuthorID: &otherAgentID},
			},
			expectCaseGetCall: true,
			expectListCall:    true,
			expectDeleteCall:  false,
			expectErr:         true,
		},
		{
			// A nil AuthorID (system/admin-authored note) must never
			// match the calling agent's own id.
			name:  "note has nil author (system-authored)",
			agent: agent,
			responseNotes: []*cmcasenote.CaseNote{
				{ID: noteID, CaseID: caseID, AuthorID: nil},
			},
			expectCaseGetCall: true,
			expectListCall:    true,
			expectDeleteCall:  false,
			expectErr:         true,
		},
		{
			// The target note ID does not appear in the list at all.
			name:              "note not found",
			agent:             agent,
			responseNotes:     []*cmcasenote.CaseNote{},
			expectCaseGetCall: true,
			expectListCall:    true,
			expectDeleteCall:  false,
			expectErr:         true,
		},
		{
			// An accesskey identity would have AgentID() == uuid.Nil,
			// which would falsely "own" any note with a nil AuthorID --
			// must be rejected before caseGet/list/delete are ever
			// reached.
			name: "accesskey identity rejected",
			agent: auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{
				CustomerID: customerID,
			}),
			expectCaseGetCall: false,
			expectListCall:    false,
			expectDeleteCall:  false,
			expectErr:         true,
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

			if tt.expectCaseGetCall {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, caseID).Return(responseCaseGet, nil)
			}

			if tt.expectListCall {
				mockReq.EXPECT().ContactV1CaseNoteList(ctx, tt.agent.CustomerID, caseID).Return(tt.responseNotes, nil)
			}
			if tt.expectDeleteCall {
				mockReq.EXPECT().ContactV1CaseNoteDelete(ctx, tt.agent.CustomerID, caseID, noteID).Return(nil)
			}

			err := h.ServiceAgentCaseNoteDelete(ctx, tt.agent, caseID, noteID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				if err != nil && err != serviceerrors.ErrPermissionDenied && err != serviceerrors.ErrNotFound && err != serviceerrors.ErrDirectAccessNotSupported {
					t.Errorf("Unexpected error type: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
