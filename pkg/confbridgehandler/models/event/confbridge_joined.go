package event

import "github.com/gofrs/uuid"

// ConfbridgeJoinedLeaved event struct for confbridge joined
type ConfbridgeJoinedLeaved struct {
	ID     uuid.UUID `json:"id"`      // confbridge id
	CallID uuid.UUID `json:"call_id"` // call id.
}
