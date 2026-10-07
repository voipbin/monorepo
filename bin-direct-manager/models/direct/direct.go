package direct

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// DirectPrefix is the prefix for direct hashes
const DirectPrefix = "direct."

// Resource type constants for Direct records
const (
	ResourceTypeAI            = "ai"
	ResourceTypeAITeam        = "ai_team"
	ResourceTypeAgent         = "agent"
	ResourceTypeQueue         = "queue"
	ResourceTypeConference    = "conference"
	ResourceTypeExtension     = "extension"
	ResourceTypeWebchatWidget = "webchat_widget"
)

// Direct data model
type Direct struct {
	commonidentity.Identity

	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id" db:"resource_id,uuid"`
	Hash         string    `json:"hash" db:"hash"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
}

// maskHashLen is the number of leading characters of a hash kept in logs.
const maskHashLen = 12

// MaskHash returns a log-safe form of a direct hash (first 12 chars + "...").
// A value of 12 chars or fewer is returned unchanged.
func MaskHash(hash string) string {
	if len(hash) <= maskHashLen {
		return hash
	}
	return hash[:maskHashLen] + "..."
}

// LogFields returns the log-safe identity fields of the direct. The hash is
// a credential-like value (it can mint a resource-scoped token) and is never
// included.
func (d *Direct) LogFields() logrus.Fields {
	if d == nil {
		return logrus.Fields{}
	}
	return logrus.Fields{
		"direct_id":     d.ID,
		"customer_id":   d.CustomerID,
		"resource_type": d.ResourceType,
		"resource_id":   d.ResourceID,
	}
}
