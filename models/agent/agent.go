package agent

import (
	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
)

// Agent data model
type Agent struct {
	ID           uuid.UUID `json:"id"`       // agent id
	UserID       uint64    `json:"user_id"`  // owned user's id
	Username     string    `json:"username"` // agent's username
	PasswordHash string    `json:"-"`        // hashed Password

	Name       string     `json:"name"`        // agent's name
	Detail     string     `json:"detail"`      // agent's detail
	RingMethod RingMethod `json:"ring_method"` // agent's ring method

	Status     Status            `json:"status"`     // agent's status
	Permission Permission        `json:"permission"` // agent's permission.
	TagIDs     []uuid.UUID       `json:"tag_ids"`    // agent's tag ids
	Addresses  []address.Address `json:"addresses"`  // agent's endpoint addresses

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
	StatusNone      Status = ""
	StatusAvailable Status = "available"
	StatusAway      Status = "away"
	StatusBusy      Status = "busy"
	StatusOffline   Status = "offline"
)

// Serialize serializes user data
// Used it for JWT generation.
func (u *Agent) Serialize() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"user_id":    u.UserID,
		"username":   u.Username,
		"permission": u.Permission,
	}
}

// Read reads the user info
func (u *Agent) Read(m map[string]interface{}) {
	u.ID = uuid.FromStringOrNil(m["id"].(string))
	u.UserID = m["user_id"].(uint64)
	u.Username = m["username"].(string)
	u.Permission = Permission(m["permission"].(float64))
}

// ConvertToAgent define
func ConvertToAgent(a *amagent.Agent) *Agent {

	addresses := []address.Address{}
	for _, addr := range a.Addresses {
		tmpAddr := address.ConvertToAddress(addr)
		addresses = append(addresses, *tmpAddr)
	}

	ag := &Agent{
		ID:           a.ID,
		UserID:       a.UserID,
		Username:     a.Username,
		PasswordHash: a.PasswordHash,
		Name:         a.Name,
		Detail:       a.Detail,
		RingMethod:   RingMethod(a.RingMethod),
		Status:       Status(a.Status),
		Permission:   Permission(a.Permission),
		TagIDs:       a.TagIDs,
		Addresses:    addresses,
		TMCreate:     a.TMCreate,
		TMUpdate:     a.TMUpdate,
		TMDelete:     a.TMDelete,
	}

	return ag
}
