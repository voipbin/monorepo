package permission

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

// Permission data model
type Permission struct {
	ID uuid.UUID `json:"id"` // ID

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// list of reserved permissions
var (
	PermissionAdmin = Permission{
		ID:       uuid.FromStringOrNil("03796e14-7cb4-11ec-9dba-e72023efd1c6"),
		Name:     "admin",
		Detail:   "reserved admin",
		TMCreate: dbhandler.DefaultTimeStamp,
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
)
