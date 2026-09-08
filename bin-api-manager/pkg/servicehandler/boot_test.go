package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"
)

// Fixtures for the resource binding claims. Values are arbitrary; what the
// tests pin is that the same value reaches both the response and the claims.
var (
	testAllowedResourceID = uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-00000000beef")
	testBootExpire        = "2026-09-09T00:00:00.000000Z"
)

// parseDirectScope pulls the direct scope back out of a signed token so a test
// can compare the claims against the response body.
func parseDirectScope(t *testing.T, token string, key []byte) auth.DirectScope {
	t.Helper()

	parsed, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) { return key, nil })
	if err != nil {
		t.Fatalf("Could not parse token. err: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("Unexpected claims type")
	}

	raw, err := json.Marshal(claims["direct"])
	if err != nil {
		t.Fatalf("Could not marshal direct claim. err: %v", err)
	}
	var scope auth.DirectScope
	if errUnmarshal := json.Unmarshal(raw, &scope); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal direct claim. err: %v", errUnmarshal)
	}
	return scope
}

func Test_AuthBoot(t *testing.T) {

	tests := []struct {
		name string

		directHash string

		responseDirect      *dmdirect.Direct
		responseDirectErr   error
		responseCustomer    *cscustomer.Customer
		responseCustomerErr error
		responseCurTime     string

		// webchat_widget-only: controls resourceDisplayConfigFetchers'
		// WebchatV1WidgetGet mock, when responseDirect.ResourceType ==
		// webchat_widget.
		expectWidgetFetch bool
		responseWidget    *wcwidget.Widget
		responseWidgetErr error

		expectErr                  bool
		expectAllowedResourceTypes []string
		expectResourceDataNil      bool
		expectPublicDisplayConfig  *wcwidget.ThemeConfig
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

			expectWidgetFetch: true,
			responseWidget: &wcwidget.Widget{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000003"), CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")},
				ThemeConfig: &wcwidget.ThemeConfig{
					PrimaryColor: "#2563eb",
					HeaderTitle:  "Acme Support",
				},
			},

			expectErr: false,

			expectAllowedResourceTypes: []string{"webchat_session"},
			expectPublicDisplayConfig: &wcwidget.ThemeConfig{
				PrimaryColor: "#2563eb",
				HeaderTitle:  "Acme Support",
			},
		},
		{
			// (b) WebchatV1WidgetGet RPC failure: /auth/boot must still
			// return HTTP 200 with resource_data entirely omitted --
			// best-effort, fail-open (design doc §3.3).
			name: "webchat_widget - RPC failure is best-effort, still HTTP 200",

			directHash: "direct.webchat124",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000004"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeWebchatWidget,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000004"),
				Hash:         "direct.webchat124",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectWidgetFetch: true,
			responseWidgetErr: fmt.Errorf("rpc timeout"),

			expectErr:             false,
			expectResourceDataNil: true,
		},
		{
			// (c) no fetcher registered for this resource_type (ai) --
			// resource_data must be omitted entirely, and this existing
			// consumer's response shape must be byte-identical to before
			// this change (no WebchatV1WidgetGet call at all).
			name: "ai - no fetcher registered, resource_data omitted",

			directHash: "direct.ai125",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000005"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeAI,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000005"),
				Hash:         "direct.ai125",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectWidgetFetch: false,

			expectErr:             false,
			expectResourceDataNil: true,
		},
		{
			// (d) fetcher SUCCEEDS but the widget has no
			// customer-configured theme (ThemeConfig == nil) -- this is
			// the exact scenario that motivated the typed-nil
			// normalization in the fetcher (design doc §9.1/§9.2):
			// resource_data must be omitted entirely, not present as
			// {"public_display_config": null}.
			name: "webchat_widget - fetcher succeeds, no theme configured, resource_data omitted",

			directHash: "direct.webchat126",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000006"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				ResourceType: dmdirect.ResourceTypeWebchatWidget,
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000006"),
				Hash:         "direct.webchat126",
			},
			responseDirectErr: nil,
			responseCustomer: &cscustomer.Customer{
				ID:     uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				Status: cscustomer.StatusActive,
			},
			responseCustomerErr: nil,
			responseCurTime:     "2026-04-06T16:00:00Z",

			expectWidgetFetch: true,
			responseWidget: &wcwidget.Widget{
				Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000006"), CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")},
				ThemeConfig: nil,
			},

			expectErr:             false,
			expectResourceDataNil: true,
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
				mockUtil.EXPECT().UUIDCreate().Return(testAllowedResourceID)
				mockUtil.EXPECT().TimeGetCurTimeAdd(BootSessionMaxLifetime).Return(testBootExpire)
				mockUtil.EXPECT().TimeGetCurTimeAdd(BootExpiration).Return(tt.responseCurTime)
			}

			if tt.expectWidgetFetch {
				mockReq.EXPECT().WebchatV1WidgetGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseWidget, tt.responseWidgetErr)
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

			// The boot response and the JWT claims must report the same
			// binding. If the response is filled from anything other than the
			// scope struct, enforcement can turn on while the client still
			// sees the old values -- and the client gates its mismatch reboot
			// on scope_version, so it would stay asleep exactly when needed.
			if res.AllowedResourceID != testAllowedResourceID {
				t.Errorf("Expected allowed resource id %v, got: %v", testAllowedResourceID, res.AllowedResourceID)
			}
			if res.ScopeVersion != DirectScopeVersionCurrent {
				t.Errorf("Expected scope version %d, got: %d", DirectScopeVersionCurrent, res.ScopeVersion)
			}
			claimScope := parseDirectScope(t, res.Token, h.jwtKey)
			if claimScope.AllowedResourceID != res.AllowedResourceID {
				t.Errorf("Response and claim disagree on allowed resource id. response: %v, claim: %v", res.AllowedResourceID, claimScope.AllowedResourceID)
			}
			if claimScope.ScopeVersion != res.ScopeVersion {
				t.Errorf("Response and claim disagree on scope version. response: %d, claim: %d", res.ScopeVersion, claimScope.ScopeVersion)
			}
			if claimScope.DirectID != tt.responseDirect.ID {
				t.Errorf("Expected direct id %v, got: %v", tt.responseDirect.ID, claimScope.DirectID)
			}
			if claimScope.HashFingerprint != h.directHashFingerprint(tt.responseDirect.Hash) {
				t.Errorf("Unexpected hash fingerprint: %v", claimScope.HashFingerprint)
			}
			if claimScope.BootExpire != testBootExpire {
				t.Errorf("Expected boot expire %v, got: %v", testBootExpire, claimScope.BootExpire)
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

			if tt.expectResourceDataNil && res.ResourceData != nil {
				t.Errorf("Expected resource_data to be omitted (nil), got: %v", res.ResourceData)
			}

			if tt.expectPublicDisplayConfig != nil {
				if res.ResourceData == nil {
					t.Fatalf("Expected resource_data to be populated, got nil")
				}
				got, ok := res.ResourceData["public_display_config"]
				if !ok {
					t.Fatalf("Expected resource_data to contain 'public_display_config' key, got: %v", res.ResourceData)
				}
				gotThemeConfig, ok := got.(*wcwidget.ThemeConfig)
				if !ok {
					t.Fatalf("Expected public_display_config to be *wcwidget.ThemeConfig, got: %T", got)
				}
				if gotThemeConfig.PrimaryColor != tt.expectPublicDisplayConfig.PrimaryColor {
					t.Errorf("Expected primary_color '%s', got: %v", tt.expectPublicDisplayConfig.PrimaryColor, gotThemeConfig.PrimaryColor)
				}
				if gotThemeConfig.HeaderTitle != tt.expectPublicDisplayConfig.HeaderTitle {
					t.Errorf("Expected header_title '%s', got: %v", tt.expectPublicDisplayConfig.HeaderTitle, gotThemeConfig.HeaderTitle)
				}
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
