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
