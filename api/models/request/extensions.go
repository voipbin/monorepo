package request

import "github.com/gofrs/uuid"

// BodyExtensionsPOST is rquest body define for POST /extensions
type BodyExtensionsPOST struct {
	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`
}

// ParamExtensionsGET is rquest param define for GET /extensions
type ParamExtensionsGET struct {
	DomainID string `form:"domain_id" binding:"required,uuid"`
	Pagination
}

// BodyExtensionsIDPUT is rquest body define for PUT /extensions/{id}
type BodyExtensionsIDPUT struct {
	Name     string `json:"name"`
	Detail   string `json:"detail"`
	Password string `json:"password"`
}
