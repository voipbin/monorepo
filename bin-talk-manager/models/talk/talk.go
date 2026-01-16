package talk

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Type constants
const (
	TypeNormal = "normal"
	TypeGroup  = "group"
)

type Type string

// Talk represents a talk session
type Talk struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	// Timestamps
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// WebhookMessage is the webhook payload for talk events
type WebhookMessage struct {
	commonidentity.Identity

	Type     Type   `json:"type"`
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
