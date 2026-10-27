package servicehandler

import (
	"context"
	"errors"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var (
	guardCustomerID      = uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-000000000001")
	guardOtherCustomerID = uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-000000000002")
	guardResourceID      = uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-000000000003")
	errGuardDownstream   = errors.New("downstream failure")
)

// guardAgent returns an agent identity of guardCustomerID holding the given permission.
func guardAgent(p amagent.Permission) *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1a1a1a1-0000-4000-8000-0000000000aa"),
			CustomerID: guardCustomerID,
		},
		Permission: p,
	})
}

func guardDirect() *auth.AuthIdentity {
	return auth.NewDirectIdentity(&auth.DirectScope{CustomerID: guardCustomerID})
}

// guardCase is one call of a service handler method together with the mock
// expectations that lead to the expected error.
type guardCase struct {
	name    string
	agent   *auth.AuthIdentity
	setup   func(mockReq *requesthandler.MockRequestHandler)
	call    func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) error
	wantErr error
}

func runGuardCases(t *testing.T, tests []guardCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}

			if tt.setup != nil {
				tt.setup(mockReq)
			}

			err := tt.call(context.Background(), h, tt.agent)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
