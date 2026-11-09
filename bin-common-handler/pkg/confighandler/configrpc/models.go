package configrpc

import (
	"time"

	"github.com/gofrs/uuid"
)

// Service is the response of GET /v1/configs: the keys of the service and whether its secrets can
// be written. There is no field that holds a value, a length or a hash of a secret.
//
// Service is the name that the caller used for the service. The manager does not set it: the
// request handler fills it in.
type Service struct {
	Service        string `json:"service,omitempty"`
	SecretWritable bool   `json:"secret_writable"`
	Keys           []Key  `json:"keys"`
}

// Key describes one key. EnvName is the name of the environment variable, never its value.
type Key struct {
	Key         string     `json:"key"`
	Kind        string     `json:"kind"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Source      string     `json:"source"`
	HasSecret   bool       `json:"has_secret"`
	EnvName     string     `json:"env_name,omitempty"`
	EnvShadowed bool       `json:"env_shadowed"`
	KeyVersion  int        `json:"key_version"`
	Revision    uint32     `json:"revision"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	TMUpdate    *time.Time `json:"tm_update,omitempty"`
}

// History is one change of a key. A secret change has the key versions only.
type History struct {
	ID            uuid.UUID `json:"id"`
	Key           string    `json:"key"`
	Action        string    `json:"action"`
	IsSecret      bool      `json:"is_secret"`
	OldKeyVersion int       `json:"old_key_version"`
	NewKeyVersion int       `json:"new_key_version"`
	ActorID       uuid.UUID `json:"actor_id"`
	Source        string    `json:"source"`
	TMCreate      time.Time `json:"tm_create"`
}

// HistoryList is one page of the history, the newest first. NextPageToken is empty on the last page.
type HistoryList struct {
	Result        []History `json:"result"`
	NextPageToken string    `json:"next_page_token"`
}

// SecretPutRequest is the body of PUT /v1/configs/<key>/secret. Revision is the revision that the
// caller read: 0 means the key has no stored value.
type SecretPutRequest struct {
	ActorID  uuid.UUID `json:"actor_id"`
	Secret   string    `json:"secret"`
	Revision *uint32   `json:"revision"`
}

// KeyDeleteRequest is the body of DELETE /v1/configs/<key>.
type KeyDeleteRequest struct {
	ActorID  uuid.UUID `json:"actor_id"`
	Revision *uint32   `json:"revision"`
}
