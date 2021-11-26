package agent

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// Agent data model
type Agent struct {
	ID           uuid.UUID `json:"id"`            // agent id
	UserID       uint64    `json:"user_id"`       // owned user's id
	Username     string    `json:"username"`      // agent's username
	PasswordHash string    `json:"password_hash"` // hashed Password

	Name       string     `json:"name"`        // agent's name
	Detail     string     `json:"detail"`      // agent's detail
	RingMethod RingMethod `json:"ring_method"` // agent's ring method

	Status     Status              `json:"status"`     // agent's status
	Permission Permission          `json:"permission"` // agent's permission.
	TagIDs     []uuid.UUID         `json:"tag_ids"`    // agent's tag ids
	Addresses  []cmaddress.Address `json:"addresses"`  // agent's endpoint addresses

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// HasPermission returns true if the user has the given permission
func (u *Agent) HasPermission(perm Permission) bool {
	return u.Permission&perm != 0
}

// Permission type
type Permission uint64

// Permission
const (
	PermissionNone  Permission = 0x0000
	PermissionAdmin Permission = 0x0001
)

// RingMethod type
type RingMethod string

// List of RingMethod types
const (
	RingMethodRingAll = "ringall"
	RingMethodLinear  = "linear"
)

// Status type
type Status string

// List of Status types
const (
	StatusAvailable Status = "available"
	StatusAway      Status = "away"
	StatusOffline   Status = "offline"
)
