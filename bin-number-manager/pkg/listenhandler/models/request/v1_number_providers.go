package request

import (
	"github.com/gofrs/uuid"
)

// V1DataNumberProvidersIDPut is
// v1 data type request struct for
// /v1/number_providers/<name> PUT
type V1DataNumberProvidersIDPut struct {
	ActorID  uuid.UUID         `json:"actor_id"`
	Enabled  *bool             `json:"enabled,omitempty"`
	Priority *int              `json:"priority,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
}

// V1DataNumberProvidersIDSecretPut is
// v1 data type request struct for
// /v1/number_providers/<name>/secret PUT
type V1DataNumberProvidersIDSecretPut struct {
	ActorID uuid.UUID `json:"actor_id"`
	Secret  string    `json:"secret"`
}

// V1DataNumberProvidersIDSecretDelete is
// v1 data type request struct for
// /v1/number_providers/<name>/secret DELETE
type V1DataNumberProvidersIDSecretDelete struct {
	ActorID uuid.UUID `json:"actor_id"`
}
