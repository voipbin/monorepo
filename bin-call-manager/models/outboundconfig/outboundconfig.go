package outboundconfig

import (
	"time"

	"github.com/gofrs/uuid"
)

// OutboundConfig holds per-customer outbound call configuration.
//
// db tag table (VOIP-1078 squirrel migration, plan §5 B-3 / R-C / R-D):
//
//	Go field                      | column                            | conversion
//	------------------------------+-----------------------------------+-----------
//	ID                            | id                                | uuid
//	CustomerID                    | customer_id                       | uuid
//	Name                          | name                              | -
//	Detail                        | detail                            | -
//	DestinationWhitelist          | destination_whitelist             | json
//	Codecs                        | codecs                            | -
//	DefaultOutgoingSourceNumberID | default_outgoing_source_number_id | uuid
//	TMCreate                      | tm_create                         | -
//	TMUpdate                      | tm_update                         | -
//	TMDelete                      | tm_delete                         | -
//
// R-D: id/customer_id/default_outgoing_source_number_id are BINARY(16) columns.
// The ,uuid tag is a WRITE-side requirement: it makes PrepareFields call
// .Bytes() before the driver would otherwise invoke gofrs/uuid's Value(),
// which always returns a 36-char string and would not fit BINARY(16).
// Reads are tag-independent for gofrs (sql.Scanner), but the ,uuid read path
// (copyUUID -> uuid.FromBytes) requires exactly 16 bytes, which is why the
// SQLite test fixture must store real 16-byte binary values.
type OutboundConfig struct {
	ID                            uuid.UUID  `json:"id" db:"id,uuid"`
	CustomerID                    uuid.UUID  `json:"customer_id" db:"customer_id,uuid"`
	Name                          string     `json:"name" db:"name"`
	Detail                        string     `json:"detail" db:"detail"`
	DestinationWhitelist          []string   `json:"destination_whitelist" db:"destination_whitelist,json"` // ISO 3166 alpha-2 lowercase
	Codecs                        string     `json:"codecs" db:"codecs"`                                    // comma-separated; empty = server default
	DefaultOutgoingSourceNumberID uuid.UUID  `json:"default_outgoing_source_number_id" db:"default_outgoing_source_number_id,uuid"`
	TMCreate                      *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate                      *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete                      *time.Time `json:"tm_delete" db:"tm_delete"`
}

// UpdateRequest uses pointer fields so callers can distinguish "absent" (nil = no change)
// from "explicit empty" (pointer to zero value = clear the field).
type UpdateRequest struct {
	Name                 *string   `json:"name,omitempty"`
	Detail               *string   `json:"detail,omitempty"`
	DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
	Codecs               *string   `json:"codecs,omitempty"`
	// DefaultOutgoingSourceNumberID — pointer semantics:
	//   nil pointer            → no change
	//   pointer → uuid.Nil     → clear (set to uuid.Nil)
	//   pointer → valid UUID   → set (after validation)
	DefaultOutgoingSourceNumberID *uuid.UUID `json:"default_outgoing_source_number_id,omitempty"`
}
