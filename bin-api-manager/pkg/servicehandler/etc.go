package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// hasPermission returns true if the given agent has correct permission
func (h *serviceHandler) hasPermission(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, permission amagent.Permission) bool {
	if a.HasPermission(amagent.PermissionProjectSuperAdmin) {
		return true
	}

	if a.CustomerID != customerID { 
		return false
	}

	if a.HasPermission(permission) {
		return true
	}

	return false
}
