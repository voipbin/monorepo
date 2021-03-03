package rmextension

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// Extension struct
type Extension struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	EndpointID string `json:"endpoint_id"`
	AORID      string `json:"aor_id"`
	AuthID     string `json:"auth_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertExtension returns converted data from rmextension.Extension to extension.Extension
func (f *Extension) ConvertExtension() *models.Extension {

	res := &models.Extension{
		ID:     f.ID,
		UserID: f.UserID,

		Name:     f.Name,
		Detail:   f.Detail,
		DomainID: f.DomainID,

		Extension: f.Extension,
		Password:  f.Password,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}

// CreateDomain returns converted data from domain.Domain to rmdomain.Domain
func CreateDomain(f *models.Extension) *Extension {

	res := &Extension{
		ID:     f.ID,
		UserID: f.UserID,

		Name:     f.Name,
		Detail:   f.Detail,
		DomainID: f.DomainID,

		Extension: f.Extension,
		Password:  f.Password,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}
