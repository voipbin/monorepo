package domain

import (
	"github.com/gofrs/uuid"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

// Domain struct for client show
type Domain struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	DomainName string `json:"domain_name"`

	Name   string `json:"name"`   // Name
	Detail string `json:"detail"` // Detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.

}

// ConvertToDomain returns converted data from rmdomain.Domain to domain.Domain
func ConvertToDomain(f *rmdomain.Domain) *Domain {

	res := &Domain{
		ID:         f.ID,
		CustomerID: f.CustomerID,

		DomainName: f.DomainName,

		Name:   f.Name,
		Detail: f.Detail,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}

// CreateDomain returns converted data from domain.Domain to rmdomain.Domain
func CreateDomain(f *Domain) *rmdomain.Domain {

	res := &rmdomain.Domain{
		ID:         f.ID,
		CustomerID: f.CustomerID,

		DomainName: f.DomainName,

		Name:   f.Name,
		Detail: f.Detail,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}
