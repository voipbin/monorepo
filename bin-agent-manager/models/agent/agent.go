package agent

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Agent data model
type Agent struct {
	commonidentity.Identity

	Username     string `json:"username" db:"username"`           // agent's username
	PasswordHash string `json:"password_hash" db:"password_hash"` // hashed Password

	Name   string `json:"name" db:"name"`     // agent's name
	Detail string `json:"detail" db:"detail"` // agent's detail

	RingMethod RingMethod `json:"ring_method" db:"ring_method"` // agent's ring method

	Status     Status                  `json:"status" db:"status"`           // agent's status
	Permission Permission              `json:"permission" db:"permission"`   // agent's permission.
	TagIDs     []uuid.UUID             `json:"tag_ids" db:"tag_ids,json"`    // agent's tag ids
	Addresses  []commonaddress.Address `json:"addresses" db:"addresses,json"` // agent's endpoint addresses

	TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"` // Deleted timestamp.
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
	PermissionAll  Permission = 0xFFFF // 65535

	// project level permission
	PermissionProjectSuperAdmin Permission = 0x0001 // 1. permission for voipbin project super admin.
	// 0x0002	// 2. reserved.
	// 0x0004	// 4. reserved.
	// 0x0008	// 8. reserved.
	PermissionProjectAll Permission = 0x000F // 15. permission for project level

	// customer level permission
	PermissionCustomerAgent   Permission = 0x0010 // 16. Permission for customer level agent
	PermissionCustomerAdmin   Permission = 0x0020 // 32. Permission for customer level admin
	PermissionCustomerManager Permission = 0x0040 // 64. Permission for customer level manager
	// 0x0080
	PermissionCustomerAll Permission = 0x00F0 // 240. Permission for customer level

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

// List of guest account
var (
	GuestAgentID uuid.UUID = uuid.FromStringOrNil("d819c626-0284-4df8-99d6-d03e1c6fba88")
)
