package auth

import (
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
)

func Test_NewAgentIdentity(t *testing.T) {
	type test struct {
		name             string
		agent            *amagent.Agent
		expectType       Type
		expectCustomerID uuid.UUID
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	agentID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Username:   "test-user",
				Permission: amagent.PermissionCustomerAdmin,
			},
			TypeAgent,
			customerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewAgentIdentity(tt.agent)
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomerID, res.CustomerID)
			}
			if res.Agent != tt.agent {
				t.Errorf("Wrong Agent. expect: %v, got: %v", tt.agent, res.Agent)
			}
		})
	}
}

func Test_NewAccesskeyIdentity(t *testing.T) {
	type test struct {
		name             string
		accesskey        *csaccesskey.Accesskey
		expectType       Type
		expectCustomerID uuid.UUID
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	accesskeyID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	tests := []test{
		{
			"normal",
			&csaccesskey.Accesskey{
				ID:         accesskeyID,
				CustomerID: customerID,
				Name:       "test-key",
			},
			TypeAccesskey,
			customerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewAccesskeyIdentity(tt.accesskey)
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomerID, res.CustomerID)
			}
			if res.Accesskey != tt.accesskey {
				t.Errorf("Wrong Accesskey. expect: %v, got: %v", tt.accesskey, res.Accesskey)
			}
		})
	}
}

func Test_NewDirectIdentity(t *testing.T) {
	type test struct {
		name             string
		scope            *DirectScope
		expectType       Type
		expectCustomerID uuid.UUID
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	resourceID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")

	tests := []test{
		{
			"normal",
			&DirectScope{
				CustomerID:           customerID,
				ResourceType:         "call",
				ResourceID:           resourceID,
				AllowedResourceTypes: []string{"call", "recording"},
			},
			TypeDirect,
			customerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewDirectIdentity(tt.scope)
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomerID, res.CustomerID)
			}
			if res.DirectScope != tt.scope {
				t.Errorf("Wrong DirectScope. expect: %v, got: %v", tt.scope, res.DirectScope)
			}
		})
	}
}

func Test_IsAgent(t *testing.T) {
	type test struct {
		name      string
		identity  AuthIdentity
		expectRes bool
	}

	tests := []test{
		{
			"agent type returns true",
			AuthIdentity{Type: TypeAgent},
			true,
		},
		{
			"accesskey type returns false",
			AuthIdentity{Type: TypeAccesskey},
			false,
		},
		{
			"direct type returns false",
			AuthIdentity{Type: TypeDirect},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.IsAgent()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsAccesskey(t *testing.T) {
	type test struct {
		name      string
		identity  AuthIdentity
		expectRes bool
	}

	tests := []test{
		{
			"accesskey type returns true",
			AuthIdentity{Type: TypeAccesskey},
			true,
		},
		{
			"agent type returns false",
			AuthIdentity{Type: TypeAgent},
			false,
		},
		{
			"direct type returns false",
			AuthIdentity{Type: TypeDirect},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.IsAccesskey()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsDirect(t *testing.T) {
	type test struct {
		name      string
		identity  AuthIdentity
		expectRes bool
	}

	tests := []test{
		{
			"direct type returns true",
			AuthIdentity{Type: TypeDirect},
			true,
		},
		{
			"agent type returns false",
			AuthIdentity{Type: TypeAgent},
			false,
		},
		{
			"accesskey type returns false",
			AuthIdentity{Type: TypeAccesskey},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.IsDirect()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_HasPermission(t *testing.T) {
	type test struct {
		name       string
		identity   AuthIdentity
		permission amagent.Permission
		expectRes  bool
	}

	tests := []test{
		{
			"agent with matching permission",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Permission: amagent.PermissionCustomerAdmin,
				},
			},
			amagent.PermissionCustomerAdmin,
			true,
		},
		{
			"agent without matching permission",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Permission: amagent.PermissionCustomerAgent,
				},
			},
			amagent.PermissionCustomerAdmin,
			false,
		},
		{
			"agent with PermissionAll always true",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Permission: amagent.PermissionCustomerAgent,
				},
			},
			amagent.PermissionAll,
			true,
		},
		{
			"agent with nil Agent returns false",
			AuthIdentity{
				Type:  TypeAgent,
				Agent: nil,
			},
			amagent.PermissionCustomerAdmin,
			false,
		},
		{
			"accesskey with CustomerAdmin permission",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			amagent.PermissionCustomerAdmin,
			true,
		},
		{
			"accesskey with combined permission including CustomerAdmin",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager,
			true,
		},
		{
			"accesskey without CustomerAdmin permission",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			amagent.PermissionProjectSuperAdmin,
			false,
		},
		{
			"direct always returns false",
			AuthIdentity{
				Type: TypeDirect,
			},
			amagent.PermissionCustomerAdmin,
			false,
		},
		{
			"unknown type returns false",
			AuthIdentity{
				Type: Type("unknown"),
			},
			amagent.PermissionCustomerAdmin,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_HasAllowedResourceType(t *testing.T) {
	type test struct {
		name         string
		identity     AuthIdentity
		resourceType string
		expectRes    bool
	}

	tests := []test{
		{
			"direct with matching resource type",
			AuthIdentity{
				Type: TypeDirect,
				DirectScope: &DirectScope{
					AllowedResourceTypes: []string{"call", "recording"},
				},
			},
			"call",
			true,
		},
		{
			"direct with non-matching resource type",
			AuthIdentity{
				Type: TypeDirect,
				DirectScope: &DirectScope{
					AllowedResourceTypes: []string{"call", "recording"},
				},
			},
			"billing",
			false,
		},
		{
			"direct with empty allowed list",
			AuthIdentity{
				Type: TypeDirect,
				DirectScope: &DirectScope{
					AllowedResourceTypes: []string{},
				},
			},
			"call",
			false,
		},
		{
			"direct with nil DirectScope",
			AuthIdentity{
				Type:        TypeDirect,
				DirectScope: nil,
			},
			"call",
			false,
		},
		{
			"agent type returns false",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Permission: amagent.PermissionCustomerAdmin,
				},
			},
			"call",
			false,
		},
		{
			"accesskey type returns false",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			"call",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.HasAllowedResourceType(tt.resourceType)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentID(t *testing.T) {
	type test struct {
		name     string
		identity AuthIdentity
		expectID uuid.UUID
	}

	agentID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []test{
		{
			"agent returns agent ID",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Identity: commonidentity.Identity{
						ID: agentID,
					},
				},
			},
			agentID,
		},
		{
			"agent with nil Agent returns uuid.Nil",
			AuthIdentity{
				Type:  TypeAgent,
				Agent: nil,
			},
			uuid.Nil,
		},
		{
			"accesskey returns uuid.Nil",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			uuid.Nil,
		},
		{
			"direct returns uuid.Nil",
			AuthIdentity{
				Type: TypeDirect,
			},
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.AgentID()
			if res != tt.expectID {
				t.Errorf("Wrong AgentID. expect: %v, got: %v", tt.expectID, res)
			}
		})
	}
}

func Test_AccesskeyID(t *testing.T) {
	type test struct {
		name     string
		identity AuthIdentity
		expectID uuid.UUID
	}

	accesskeyID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	tests := []test{
		{
			"accesskey returns accesskey ID",
			AuthIdentity{
				Type: TypeAccesskey,
				Accesskey: &csaccesskey.Accesskey{
					ID: accesskeyID,
				},
			},
			accesskeyID,
		},
		{
			"accesskey with nil Accesskey returns uuid.Nil",
			AuthIdentity{
				Type:      TypeAccesskey,
				Accesskey: nil,
			},
			uuid.Nil,
		},
		{
			"agent returns uuid.Nil",
			AuthIdentity{
				Type: TypeAgent,
			},
			uuid.Nil,
		},
		{
			"direct returns uuid.Nil",
			AuthIdentity{
				Type: TypeDirect,
			},
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.AccesskeyID()
			if res != tt.expectID {
				t.Errorf("Wrong AccesskeyID. expect: %v, got: %v", tt.expectID, res)
			}
		})
	}
}

func Test_AgentUsername(t *testing.T) {
	type test struct {
		name           string
		identity       AuthIdentity
		expectUsername string
	}

	tests := []test{
		{
			"agent returns username",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Username: "test-user",
				},
			},
			"test-user",
		},
		{
			"agent with nil Agent returns empty string",
			AuthIdentity{
				Type:  TypeAgent,
				Agent: nil,
			},
			"",
		},
		{
			"accesskey returns empty string",
			AuthIdentity{
				Type: TypeAccesskey,
			},
			"",
		},
		{
			"direct returns empty string",
			AuthIdentity{
				Type: TypeDirect,
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.AgentUsername()
			if res != tt.expectUsername {
				t.Errorf("Wrong AgentUsername. expect: %v, got: %v", tt.expectUsername, res)
			}
		})
	}
}

func Test_NewDelegateIdentity(t *testing.T) {
	type test struct {
		name             string
		scope            *DelegateScope
		expectType       Type
		expectCustomerID uuid.UUID
		expectJTI        string
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	tests := []test{
		{
			"normal",
			&DelegateScope{
				CustomerID: customerID,
				IssuedBy:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				JTI:        "jti-value-123",
			},
			TypeDelegate,
			customerID,
			"jti-value-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewDelegateIdentity(tt.scope)
			if res.Type != tt.expectType {
				t.Errorf("Wrong Type. expect: %v, got: %v", tt.expectType, res.Type)
			}
			if res.CustomerID != tt.expectCustomerID {
				t.Errorf("Wrong CustomerID. expect: %v, got: %v", tt.expectCustomerID, res.CustomerID)
			}
			if res.DelegateScope == nil {
				t.Fatal("DelegateScope is nil")
			}
			if res.DelegateScope.JTI != tt.expectJTI {
				t.Errorf("Wrong JTI. expect: %v, got: %v", tt.expectJTI, res.DelegateScope.JTI)
			}
		})
	}
}

func Test_HasPermission_Delegate(t *testing.T) {
	type test struct {
		name       string
		identity   AuthIdentity
		permission amagent.Permission
		expectRes  bool
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	tests := []test{
		{
			"delegate grants PermissionCustomerAdmin",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionCustomerAdmin,
			true,
		},
		{
			"delegate denies PermissionProjectSuperAdmin",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionProjectSuperAdmin,
			false,
		},
		{
			"delegate denies PermissionProjectAll",
			AuthIdentity{
				Type:       TypeDelegate,
				CustomerID: customerID,
				DelegateScope: &DelegateScope{
					CustomerID: customerID,
				},
			},
			amagent.PermissionProjectAll,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.HasPermission(tt.permission)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsDelegate(t *testing.T) {
	type test struct {
		name      string
		identity  AuthIdentity
		expectRes bool
	}

	tests := []test{
		{"delegate type returns true", AuthIdentity{Type: TypeDelegate}, true},
		{"agent type returns false", AuthIdentity{Type: TypeAgent}, false},
		{"accesskey type returns false", AuthIdentity{Type: TypeAccesskey}, false},
		{"direct type returns false", AuthIdentity{Type: TypeDirect}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.IsDelegate()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DisplayName(t *testing.T) {
	type test struct {
		name       string
		identity   AuthIdentity
		expectName string
	}

	tests := []test{
		{
			"agent with username",
			AuthIdentity{
				Type: TypeAgent,
				Agent: &amagent.Agent{
					Username: "admin@example.com",
				},
			},
			"admin@example.com",
		},
		{
			"agent with nil Agent",
			AuthIdentity{
				Type:  TypeAgent,
				Agent: nil,
			},
			"agent",
		},
		{
			"accesskey with name",
			AuthIdentity{
				Type: TypeAccesskey,
				Accesskey: &csaccesskey.Accesskey{
					Name: "my-api-key",
				},
			},
			"accesskey:my-api-key",
		},
		{
			"accesskey with nil Accesskey",
			AuthIdentity{
				Type:      TypeAccesskey,
				Accesskey: nil,
			},
			"accesskey",
		},
		{
			"direct with resource type",
			AuthIdentity{
				Type: TypeDirect,
				DirectScope: &DirectScope{
					ResourceType: "call",
				},
			},
			"direct:call",
		},
		{
			"direct with nil DirectScope",
			AuthIdentity{
				Type:        TypeDirect,
				DirectScope: nil,
			},
			"direct",
		},
		{
			"unknown type",
			AuthIdentity{
				Type: Type("unknown"),
			},
			"unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.identity.DisplayName()
			if res != tt.expectName {
				t.Errorf("Wrong DisplayName. expect: %v, got: %v", tt.expectName, res)
			}
		})
	}
}
