package servicehandler

import (
	"context"
	"fmt"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AuthDelegate(t *testing.T) {
	superAdminAgentID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")

	superAdminIdentity := &auth.AuthIdentity{
		Type: auth.TypeAgent,
		Agent: &amagent.Agent{
			Identity:   commonidentity.Identity{ID: superAdminAgentID},
			Permission: amagent.PermissionProjectSuperAdmin,
		},
	}

	delegateIdentity := &auth.AuthIdentity{
		Type: auth.TypeDelegate,
		DelegateScope: &auth.DelegateScope{
			CustomerID: customerID,
			IssuedBy:   superAdminAgentID,
			JTI:        "some-jti",
		},
	}

	customerAdminIdentity := &auth.AuthIdentity{
		Type: auth.TypeAgent,
		Agent: &amagent.Agent{
			Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000002")},
			Permission: amagent.PermissionCustomerAdmin,
		},
	}

	activeCustomer := &cscustomer.Customer{
		ID:     customerID,
		Status: cscustomer.StatusActive,
	}

	validReason := "Investigating customer issue JIRA-1234"

	tests := []struct {
		name string

		identity         *auth.AuthIdentity
		targetCustomerID uuid.UUID
		reason           string

		responseCustomer    *cscustomer.Customer
		responseCustomerErr error
		responseCurTime     string

		expectErr bool
	}{
		{
			name: "superadmin gets delegate token",

			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           validReason,

			responseCustomer:    activeCustomer,
			responseCustomerErr: nil,
			responseCurTime:     "2026-05-18T12:00:00Z",

			expectErr: false,
		},
		{
			name: "TypeDelegate caller rejected",

			identity:         delegateIdentity,
			targetCustomerID: customerID,
			reason:           validReason,

			expectErr: true,
		},
		{
			name: "non-superadmin rejected",

			identity:         customerAdminIdentity,
			targetCustomerID: customerID,
			reason:           validReason,

			expectErr: true,
		},
		{
			name: "customer not found returns error",

			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           validReason,

			responseCustomer:    nil,
			responseCustomerErr: fmt.Errorf("not found"),

			expectErr: true,
		},
		{
			name: "reason too short rejected",

			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           "short",

			expectErr: true,
		},
		{
			name: "reason with control char rejected",

			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           "reason with newline\ninside",

			expectErr: true,
		},
		{
			name: "reason too long rejected",

			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           string(make([]byte, 201)),

			expectErr: true,
		},
		{
			name:             "deleted customer returns error",
			identity:         superAdminIdentity,
			targetCustomerID: customerID,
			reason:           "investigating dropped call for customer",
			responseCustomer: &cscustomer.Customer{
				ID:     customerID,
				Status: cscustomer.StatusDeleted,
			},
			responseCustomerErr: nil,

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
				jwtKey:      []byte("testkey"),
			}
			ctx := context.Background()

			// Only set up CustomerGet mock when we are past the early guards:
			// - identity is not TypeDelegate
			// - identity has PermissionProjectSuperAdmin
			// - reason is valid (10-200 chars, no control chars)
			needsCustomerLookup := !tt.identity.IsDelegate() &&
				tt.identity.HasPermission(amagent.PermissionProjectSuperAdmin) &&
				len(tt.reason) >= 10 &&
				len(tt.reason) <= 200 &&
				!containsControlChar(tt.reason)

			if needsCustomerLookup {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.targetCustomerID).Return(tt.responseCustomer, tt.responseCustomerErr)
			}

			// Only expect TimeGetCurTimeAdd when success is expected
			if !tt.expectErr {
				mockUtil.EXPECT().TimeGetCurTimeAdd(DelegateExpiration).Return(tt.responseCurTime)
			}

			res, err := h.AuthDelegate(ctx, tt.identity, tt.targetCustomerID, tt.reason)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			if res.Token == "" {
				t.Errorf("Expected non-empty token, got empty")
			}

			if res.CustomerID != tt.targetCustomerID {
				t.Errorf("Expected customer_id %s, got %s", tt.targetCustomerID, res.CustomerID)
			}

			if res.Expire != tt.responseCurTime {
				t.Errorf("Expected expire %s, got %s", tt.responseCurTime, res.Expire)
			}
		})
	}
}

// containsControlChar is a test helper mirroring validateDelegateReason's printable ASCII check.
// Returns true if the string contains any character outside the printable ASCII range (0x20–0x7E).
func containsControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			return true
		}
	}
	return false
}
