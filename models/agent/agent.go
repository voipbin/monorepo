package agent

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// Agent data model
type Agent struct {
	ID           uuid.UUID `json:"id"`            // agent id
	CustomerID   uuid.UUID `json:"customer_id"`   // customer's id
	Username     string    `json:"username"`      // agent's username
	PasswordHash string    `json:"password_hash"` // hashed Password

	Name   string `json:"name"`   // agent's name
	Detail string `json:"detail"` // agent's detail

	RingMethod RingMethod `json:"ring_method"` // agent's ring method

	Status     Status                  `json:"status"`     // agent's status
	Permission Permission              `json:"permission"` // agent's permission.
	TagIDs     []uuid.UUID             `json:"tag_ids"`    // agent's tag ids
	Addresses  []commonaddress.Address `json:"addresses"`  // agent's endpoint addresses

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// HasPermission returns true if the user has the given permission
func (u *Agent) HasPermission(perm Permission) bool {

	if perm == PermissionAll {
		return true
	}

	return u.Permission&perm != 0
}

// Permission type
type Permission uint64

// Permission
const (
	PermissionNone Permission = 0x0000
	PermissionAll  Permission = 0xFFFF

	// project level permission
	PermissionProjectSuperAdmin Permission = 0x0001 // permission for voipbin project super admin.
	// 0x0002
	// 0x0004
	// 0x0008

	// customer level permission
	// 0x0010
	PermissionCustomerAdmin   Permission = 0x0020 // Permission for customer level admin
	PermissionCustomerManager Permission = 0x0040 // Permission for customer level manager
	// 0x0080
	// ...

	// reserved level permission
	// 0x0100 //

	// reserved level permission
	// 0x1000 //
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
	StatusNone      Status = ""          // none
	StatusAvailable Status = "available" // available
	StatusAway      Status = "away"      // away
	StatusBusy      Status = "busy"      // busy
	StatusOffline   Status = "offline"   // offline
	StatusRinging   Status = "ringing"   // voipbin is making a call to the agent
)
