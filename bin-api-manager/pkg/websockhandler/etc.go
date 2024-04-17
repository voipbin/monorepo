package websockhandler

import (
	"context"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
)

// validateTopics returns true if the given topics are valid all for the given agent.
func (h *websockHandler) validateTopics(ctx context.Context, a *amagent.Agent, topics []string) bool {

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

		if tmps[0] == "customer_id" {
			if !a.HasPermission(amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager) {
				// the agent has no permission for this topic
				return false
			}

			if tmpID != a.CustomerID {
				// the customer id must be the same
				return false
			}
		} else if tmps[0] == "agent_id" {
			if tmpID != a.ID {
				//
				return false
			}
		} else {
			// the first part should be "customer_id" or "agent_id"
			return false
		}
	}

	return true
}

func (h *websockHandler) validateTopic(ctx context.Context, a *amagent.Agent, topic string) bool {

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

	if tmps[0] == "customer_id" {
		if !a.HasPermission(amagent.PermissionCustomerAdmin | amagent.PermissionCustomerManager) {
			// the agent has no permission for this topic
			return false
		}

		if tmpID != a.CustomerID {
			// the customer id must be the same
			return false
		}
	} else if tmps[0] == "agent_id" {
		if tmpID != a.ID {
			//
			return false
		}
	} else {
		// the first part should be "customer_id" or "agent_id"
		return false
	}

	return true
}
