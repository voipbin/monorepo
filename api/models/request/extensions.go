package request

import "github.com/gofrs/uuid"

// BodyExtensionsPOST is rquest body define for
// POST /v1.0/extensions
type BodyExtensionsPOST struct {
	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`
}

// ParamExtensionsGET is rquest param define for
// GET /v1.0/extensions
type ParamExtensionsGET struct {
	DomainID string `form:"domain_id"`
	Pagination
}

// BodyExtensionsIDPUT is rquest body define for
// PUT /v1.0/extensions/<extension-id>
type BodyExtensionsIDPUT struct {
	Name     string `json:"name"`
	Detail   string `json:"detail"`
	Password string `json:"password"`
}
