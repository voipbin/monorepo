package outboundconfig

import (
	"time"

	"github.com/gofrs/uuid"
)

// OutboundConfig holds per-customer outbound call configuration.
type OutboundConfig struct {
	ID                   uuid.UUID  `json:"id"`
	CustomerID           uuid.UUID  `json:"customer_id"`
	Name                 string     `json:"name"`
	Detail               string     `json:"detail"`
	DestinationWhitelist []string   `json:"destination_whitelist"` // ISO 3166 alpha-2 lowercase
	Codecs               string     `json:"codecs"`                // comma-separated; empty = server default
	TMCreate             *time.Time `json:"tm_create"`
	TMUpdate             *time.Time `json:"tm_update"`
	TMDelete             *time.Time `json:"tm_delete"`
}

// UpdateRequest uses pointer fields so callers can distinguish "absent" (nil = no change)
// from "explicit empty" (pointer to zero value = clear the field).
type UpdateRequest struct {
	Name                 *string   `json:"name,omitempty"`
	Detail               *string   `json:"detail,omitempty"`
	DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
	Codecs               *string   `json:"codecs,omitempty"`
}
