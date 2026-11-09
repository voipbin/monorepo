package device

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Platform is the platform of a push device.
type Platform string

// list of platforms
const (
	PlatformAndroid Platform = "android"
)

// MaxTokenLength is the longest accepted token in bytes. It is the size of the token column.
const MaxTokenLength = 512

// Device is a push device of an agent: one installed app that can receive notifications.
//
// The token is a credential. It never leaves this service through json (json:"-"), the sender reads
// it from the database.
type Device struct {
	commonidentity.Identity

	AgentID  uuid.UUID `json:"agent_id" db:"agent_id,uuid"` // owner agent id
	Platform Platform  `json:"platform" db:"platform"`      // device platform
	Token    string    `json:"-" db:"token"`                // push token (credential)

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Refreshed timestamp. Used for the expiry.
}

// IsValidPlatform returns true if the platform is known.
func IsValidPlatform(p Platform) bool {
	return p == PlatformAndroid
}
