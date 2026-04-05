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
				// Note: resource_id (tmps[3]) is NOT validated against DirectScope.ResourceID
				// because they refer to different things. DirectScope.ResourceID is the parent
				// resource (e.g., AI), while tmps[3] is a child resource (e.g., aicall).
				if tmpID != a.CustomerID {
					return false
				}
				if a.IsDirect() {
					// Direct token: validate resource type is allowed
					if !a.HasAllowedResourceType(tmps[2]) {
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

func (h *websockHandler) validateTopic(ctx context.Context, a *auth.AuthIdentity, topic string) bool {

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
			// Note: resource_id (tmps[3]) is NOT validated against DirectScope.ResourceID
			// because they refer to different things. DirectScope.ResourceID is the parent
			// resource (e.g., AI), while tmps[3] is a child resource (e.g., aicall).
			if tmpID != a.CustomerID {
				return false
			}
			if a.IsDirect() {
				// Direct token: validate resource type is allowed
				if !a.HasAllowedResourceType(tmps[2]) {
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

	return true
}
