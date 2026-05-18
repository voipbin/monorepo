package auth

import (
	"slices"

	amagent "monorepo/bin-agent-manager/models/agent"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
)

// Type represents the authentication method used.
type Type string

const (
	TypeAgent     Type = "agent"
	TypeAccesskey Type = "accesskey"
	TypeDirect    Type = "direct"
	TypeDelegate  Type = "delegate"
)

// AuthIdentity is the unified authentication identity for all request types.
type AuthIdentity struct {
	Type        Type
	CustomerID  uuid.UUID // always set — populated from agent, accesskey, or direct scope

	Agent         *amagent.Agent         // non-nil for TypeAgent
	Accesskey     *csaccesskey.Accesskey // non-nil for TypeAccesskey
	DirectScope   *DirectScope           // non-nil for TypeDirect
	DelegateScope *DelegateScope         // non-nil for TypeDelegate
}

// DirectScope represents a resource-scoped JWT claim set.
type DirectScope struct {
	CustomerID           uuid.UUID `json:"customer_id"`
	ResourceType         string    `json:"resource_type"`
	ResourceID           uuid.UUID `json:"resource_id"`
	AllowedResourceTypes []string  `json:"allowed_resource_types"`
}

// DelegateScope represents a superadmin-issued delegate JWT claim set.
type DelegateScope struct {
	CustomerID uuid.UUID `json:"customer_id"`
	IssuedBy   uuid.UUID `json:"issued_by"`
	JTI        string    `json:"jti"`
}

// IsAgent returns true if the identity was created from an agent JWT.
func (a *AuthIdentity) IsAgent() bool {
	return a.Type == TypeAgent
}

// IsAccesskey returns true if the identity was created from an accesskey.
func (a *AuthIdentity) IsAccesskey() bool {
	return a.Type == TypeAccesskey
}

// IsDirect returns true if the identity was created from a direct-scope JWT.
func (a *AuthIdentity) IsDirect() bool {
	return a.Type == TypeDirect
}

// IsDelegate returns true if the identity was created from a delegate JWT.
func (a *AuthIdentity) IsDelegate() bool {
	return a.Type == TypeDelegate
}

// HasPermission checks agent permissions.
// Agent: delegates to Agent.HasPermission().
// Accesskey: hardcoded CustomerAdmin (matches current behavior).
// Direct: returns false (use HasAllowedResourceType instead).
func (a *AuthIdentity) HasPermission(p amagent.Permission) bool {
	switch a.Type {
	case TypeAgent:
		if a.Agent == nil {
			return false
		}
		return a.Agent.HasPermission(p)
	case TypeAccesskey:
		return (amagent.PermissionCustomerAdmin & p) != 0
	case TypeDelegate:
		// Delegate tokens grant PermissionCustomerAdmin-equivalent access.
		// Explicitly excludes all project-level permissions.
		return (amagent.PermissionCustomerAdmin&p) != 0 &&
			(p&(amagent.PermissionProjectSuperAdmin|amagent.PermissionProjectAll)) == 0
	default:
		return false
	}
}

// HasAllowedResourceType checks if the direct scope permits the given resource type.
func (a *AuthIdentity) HasAllowedResourceType(rt string) bool {
	if a.DirectScope == nil {
		return false
	}
	return slices.Contains(a.DirectScope.AllowedResourceTypes, rt)
}

// AgentID returns the agent UUID. Returns uuid.Nil for non-agent auth.
func (a *AuthIdentity) AgentID() uuid.UUID {
	if a.Agent == nil {
		return uuid.Nil
	}
	return a.Agent.ID
}

// AccesskeyID returns the accesskey UUID. Returns uuid.Nil for non-accesskey auth.
func (a *AuthIdentity) AccesskeyID() uuid.UUID {
	if a.Accesskey == nil {
		return uuid.Nil
	}
	return a.Accesskey.ID
}

// AgentUsername returns the agent username. Returns "" for non-agent auth.
func (a *AuthIdentity) AgentUsername() string {
	if a.Agent == nil {
		return ""
	}
	return a.Agent.Username
}

// DisplayName returns a human-readable name for logging.
func (a *AuthIdentity) DisplayName() string {
	switch a.Type {
	case TypeAgent:
		if a.Agent != nil {
			return a.Agent.Username
		}
		return "agent"
	case TypeAccesskey:
		if a.Accesskey != nil {
			return "accesskey:" + a.Accesskey.Name
		}
		return "accesskey"
	case TypeDirect:
		if a.DirectScope != nil {
			return "direct:" + a.DirectScope.ResourceType
		}
		return "direct"
	case TypeDelegate:
		if a.DelegateScope != nil {
			return "delegate:" + a.DelegateScope.CustomerID.String()
		}
		return "delegate"
	default:
		return "unknown"
	}
}

// NewAgentIdentity constructs an AuthIdentity from an agent.
func NewAgentIdentity(agent *amagent.Agent) *AuthIdentity {
	return &AuthIdentity{
		Type:       TypeAgent,
		CustomerID: agent.CustomerID,
		Agent:      agent,
	}
}

// NewAccesskeyIdentity constructs an AuthIdentity from an accesskey.
func NewAccesskeyIdentity(ak *csaccesskey.Accesskey) *AuthIdentity {
	return &AuthIdentity{
		Type:       TypeAccesskey,
		CustomerID: ak.CustomerID,
		Accesskey:  ak,
	}
}

// NewDirectIdentity constructs an AuthIdentity from a direct scope.
func NewDirectIdentity(scope *DirectScope) *AuthIdentity {
	return &AuthIdentity{
		Type:        TypeDirect,
		CustomerID:  scope.CustomerID,
		DirectScope: scope,
	}
}

// NewDelegateIdentity constructs an AuthIdentity from a delegate scope.
func NewDelegateIdentity(scope *DelegateScope) *AuthIdentity {
	return &AuthIdentity{
		Type:          TypeDelegate,
		CustomerID:    scope.CustomerID,
		DelegateScope: scope,
	}
}
