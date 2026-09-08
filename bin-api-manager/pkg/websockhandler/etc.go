package websockhandler

import (
	"context"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
)

// validateTopics returns true if the given topics are valid all for the given agent.
func (h *websockHandler) validateTopics(ctx context.Context, a *auth.AuthIdentity, topics []string) bool {

	if a.HasPermission(amagent.PermissionProjectSuperAdmin) {
		// for the project super admin, all topics are valid
		return true
	}

	for _, topic := range topics {

		tmps := strings.Split(topic, ":")
		if len(tmps) < 2 {
			// too short.
			return false
		}

		// check the second part
		tmpID := uuid.FromStringOrNil(tmps[1])
		if tmpID == uuid.Nil {
			// the second part should be a valid uuid
			return false
		}

		switch tmps[0] {
		case "customer_id":
			if len(tmps) == 4 {
				// 4-part: "customer_id:<uuid>:<resource_type>:<resource_id>"
				//
				// tmps[3] is the child resource (an aicall, a session), not the
				// parent in DirectScope.ResourceID (the AI, the widget). For a
				// direct token it is checked against AllowedResourceID, which
				// is the per-visitor assignment; comparing it to ResourceID
				// would be meaningless, since every visitor of the same public
				// link shares that value.
				if tmpID != a.CustomerID {
					return false
				}
				if a.IsDirect() {
					// Guard the dereference below. validateTopics runs inside a
					// bare goroutine with no recover and outside gin.Recovery,
					// so a nil panic here takes down the process rather than
					// the socket.
					if a.DirectScope == nil {
						return false
					}
					// Direct token: validate resource type is allowed
					if !a.HasAllowedResourceType(tmps[2]) {
						return false
					}

					// The resource id must be a canonical UUID. An empty or
					// partial value would otherwise act as a wildcard: delivery
					// matches on strings.HasPrefix, so "customer_id:<cid>:aicall:"
					// would receive every aicall event in the tenant.
					//
					// This is stricter than the REST middleware, which accepts
					// braced and upper-case forms. That asymmetry is deliberate:
					// a subscription string has to byte-match what the publisher
					// emits, so a non-canonical form would silently receive
					// nothing. Do not "harmonize" this by loosening it.
					parsed, err := uuid.FromString(tmps[3])
					if err != nil || parsed.String() != tmps[3] {
						return false
					}

					// And it must be this visitor's own resource.
					if parsed != a.DirectScope.AllowedResourceID {
						return false
					}
				} else {
					// Agent/accesskey: same permission check as 2-part
					if !a.HasPermission(amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager) {
						return false
					}
				}
			} else if len(tmps) >= 2 {
				// 2-part or 3-part: "customer_id:<uuid>" or "customer_id:<uuid>:<suffix>"
				if a.IsDirect() {
					// Direct tokens cannot subscribe to broad customer topics
					return false
				}
				if !a.HasPermission(amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager) {
					return false
				}
				if tmpID != a.CustomerID {
					return false
				}
			} else {
				return false
			}

		case "agent_id":
			if tmpID != a.AgentID() {
				return false
			}

		default:
			// the first part should be "customer_id" or "agent_id"
			return false
		}
	}

	return true
}
