package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentContactCaseNoteList(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	contactID := uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f12345678901")

	agentIdentity := func(perm amagent.Permission) *auth.AuthIdentity {
		return auth.NewAgentIdentity(&amagent.Agent{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: customerID,
			},
			Permission: perm,
		})
	}

	notes := []*cmcasenote.CaseNote{
		{ID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"), CaseID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"), Text: "n"},
	}

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		expectCall  bool
		callErr     error
		expectNotes []*cmcasenote.CaseNote
		expectToken string
		expectErr   error
	}{
		{
			name:        "plain agent permission",
			agent:       agentIdentity(amagent.PermissionCustomerAgent),
			expectCall:  true,
			expectNotes: notes,
			expectToken: "2026-06-01T09:30:01.000000Z",
		},
		{
			name: "direct identity rejected before any call",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID: customerID,
			}),
			expectErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name:       "downstream error is returned",
			agent:      agentIdentity(amagent.PermissionCustomerAgent),
			expectCall: true,
			callErr:    serviceerrors.ErrNotFound,
			expectErr:  serviceerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectCall {
				mockReq.EXPECT().ContactV1ContactCaseNoteList(ctx, customerID, contactID, uint64(50), "tok").
					Return(tt.expectNotes, tt.expectToken, tt.callErr)
			}

			res, token, err := h.ServiceAgentContactCaseNoteList(ctx, tt.agent, contactID, 50, "tok")
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectNotes) || token != tt.expectToken {
				t.Errorf("Wrong match.\nexpect: %v %q\ngot: %v %q", tt.expectNotes, tt.expectToken, res, token)
			}
		})
	}
}
