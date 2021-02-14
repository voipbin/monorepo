package rmdomain

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
)

// Domain struct
type Domain struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	DomainName string `json:"domain_name"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertDomain returns converted data from rmdomain.Domain to domain.Domain
func (f *Domain) ConvertDomain() *domain.Domain {

	res := &domain.Domain{
		ID:     f.ID,
		UserID: f.UserID,

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
func CreateDomain(f *domain.Domain) *Domain {

	res := &Domain{
		ID:     f.ID,
		UserID: f.UserID,

		DomainName: f.DomainName,

		Name:   f.Name,
		Detail: f.Detail,

		TMCreate: f.TMCreate,
		TMUpdate: f.TMUpdate,
		TMDelete: f.TMDelete,
	}

	return res
}
