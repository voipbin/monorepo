package identity

import "github.com/gofrs/uuid"

// Owner represents the owner of the given resource
type Owner struct {
	OwnerType OwnerType `json:"owner_type" db:"owner_type"` // resource's owner type
	OwnerID   uuid.UUID `json:"owner_id" db:"owner_id,uuid"` // resource's owner id
}

// OwnerType defines
type OwnerType string

// list of owner types
const (
	OwnerTypeNone  OwnerType = ""
	OwnerTypeAgent OwnerType = "agent" // the owner id is agent's id.
)
