package customer

import "github.com/gofrs/uuid"

// IsInternalSystemID returns true for VoIPbin internal system customer IDs
// that bypass outbound permission and whitelist checks.
func IsInternalSystemID(id uuid.UUID) bool {
	return id == IDCallManager || id == IDAIManager || id == IDSystem || id == IDBasicRoute
}
