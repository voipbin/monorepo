package servicehandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"
)

func Test_AuthBoot(t *testing.T) {

	tests := []struct {
		name string

		directHash string

		responseDirect      *dmdirect.Direct
		responseDirectErr   error
		responseCustomer    *cscustomer.Customer
		responseCustomerErr error
		responseCurTime     string

		expectErr                  bool
		expectAllowedResourceTypes []string
	}{
		{
			name: "happy path",

			directHash: "direct.abc123",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeAI,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				Hash:         "direct.abc123",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectErr: false,
		},
		{
			name: "happy path - ai_team",

			directHash: "direct.team123",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000002"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeAITeam,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000002"),
				Hash:         "direct.team123",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectErr: false,
		},
		{
			name: "invalid hash prefix",

			directHash: "invalid_hash",

			expectErr: true,
		},
		{
			name: "hash not found",

			directHash: "direct.notfound",

			responseDirectErr: fmt.Errorf("not found"),

			expectErr: true,
		},
		{
			name: "customer not active",

			directHash: "direct.abc123",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeAI,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				Hash:         "direct.abc123",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusFrozen,
			},
			responseCustomerErr: nil,

			expectErr: true,
		},
		{
			name: "unsupported resource type",

			directHash: "direct.abc123",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: "unknown",
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				Hash:         "direct.abc123",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,

			expectErr: true,
		},
		{
			name: "happy path - webchat_widget",

			directHash: "direct.webchat123",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeWebchatWidget,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000003"),
				Hash:         "direct.webchat123",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectErr: false,

			expectAllowedResourceTypes: []string{"webchat_session"},
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

			// set up mocks based on test case
			if tt.directHash != "" && len(tt.directHash) > len("direct.") && tt.directHash[:len("direct.")] == "direct." {
				mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.directHash).Return(tt.responseDirect, tt.responseDirectErr)
			}

			if tt.responseDirect != nil && tt.responseDirectErr == nil {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseDirect.CustomerID).Return(tt.responseCustomer, tt.responseCustomerErr)
			}

			if !tt.expectErr {
				mockUtil.EXPECT().TimeGetCurTimeAdd(BootExpiration).Return(tt.responseCurTime)
			}

			res, err := h.AuthBoot(ctx, tt.directHash)
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

			if res.Type != "direct" {
				t.Errorf("Expected type 'direct', got: %v", res.Type)
			}

			if res.ResourceType != tt.responseDirect.ResourceType {
				t.Errorf("Expected resource_type '%s', got: %v", tt.responseDirect.ResourceType, res.ResourceType)
			}

			if res.ResourceID != tt.responseDirect.ResourceID {
				t.Errorf("Expected resource_id '%s', got: %v", tt.responseDirect.ResourceID, res.ResourceID)
			}

			if res.CustomerID != tt.responseDirect.CustomerID {
				t.Errorf("Expected customer_id '%s', got: %v", tt.responseDirect.CustomerID, res.CustomerID)
			}

			if res.Token == "" {
				t.Errorf("Expected non-empty token string, got empty")
			}

			if res.Expire != tt.responseCurTime {
				t.Errorf("Expected expire '%s', got: %v", tt.responseCurTime, res.Expire)
			}

			if tt.expectAllowedResourceTypes != nil {
				parsed, parseErr := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
					return h.jwtKey, nil
				})
				if parseErr != nil {
					t.Fatalf("Could not parse issued JWT: %v", parseErr)
				}

				claims, ok := parsed.Claims.(jwt.MapClaims)
				if !ok {
					t.Fatalf("Could not read JWT claims")
				}

				directClaim, ok := claims["direct"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected 'direct' claim to be an object, got: %v", claims["direct"])
				}

				gotAllowed, ok := directClaim["allowed_resource_types"].([]interface{})
				if !ok {
					t.Fatalf("Expected 'allowed_resource_types' to be an array, got: %v", directClaim["allowed_resource_types"])
				}

				if len(gotAllowed) != len(tt.expectAllowedResourceTypes) {
					t.Fatalf("Expected %d allowed_resource_types, got %d: %v", len(tt.expectAllowedResourceTypes), len(gotAllowed), gotAllowed)
				}

				for i, want := range tt.expectAllowedResourceTypes {
					if gotAllowed[i] != want {
						t.Errorf("Expected allowed_resource_types[%d] = %q, got: %v", i, want, gotAllowed[i])
					}
				}
			}
		})
	}
}
