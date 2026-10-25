// Package provider holds the number provider setting model, its sentinel errors and the
// credential type. It is a leaf package so the provider implementations and the registry can
// both import it without an import cycle.
package provider

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

// sentinel errors
var (
	// ErrPurchaseRejected means the provider received the purchase request and definitively
	// refused it (no stock, invalid number, order failed). The next provider may be tried.
	ErrPurchaseRejected = errors.New("purchase rejected by the provider")

	// ErrPurchaseUnconfirmed means the order may have been accepted by the provider (network
	// error, timeout, 5xx, lookup failure after the order). The purchase must not fall back.
	ErrPurchaseUnconfirmed = errors.New("purchase result is unconfirmed")

	// ErrSecretKeyNotConfigured means no encryption key is configured, so a secret cannot be stored.
	ErrSecretKeyNotConfigured = errors.New("secret encryption key is not configured")

	// ErrNotImplemented means the provider does not implement the operation.
	ErrNotImplemented = errors.New("not implemented by the provider")

	// ErrInvalidArgument means the admin request failed validation.
	ErrInvalidArgument = errors.New("invalid argument")
)

// CredentialSource tells where the credentials of a provider come from.
type CredentialSource string

// list of credential sources
const (
	CredentialSourceDB              CredentialSource = "db"
	CredentialSourceEnv             CredentialSource = "env"
	CredentialSourceNone            CredentialSource = "none"
	CredentialSourceDBUndecryptable CredentialSource = "db_undecryptable"
)

const redacted = "[REDACTED]"

// Credentials is the resolved credential of a provider. The secret never appears in its
// formatted or JSON output.
type Credentials struct {
	Secret string            `json:"-"`
	Config map[string]string `json:"config,omitempty"`
}

// String implements fmt.Stringer and hides the secret.
func (c Credentials) String() string {
	return "Credentials{Secret:" + redacted + "}"
}

// GoString implements fmt.GoStringer and hides the secret.
func (c Credentials) GoString() string {
	return c.String()
}

// Provider is a row of number_providers.
type Provider struct {
	ProviderName number.ProviderName `json:"provider_name" db:"provider_name"`

	Enabled  bool              `json:"enabled" db:"enabled"`
	Priority int               `json:"priority" db:"priority"`
	Config   map[string]string `json:"config,omitempty" db:"config,json"`

	SecretCiphertext []byte `json:"-" db:"secret_ciphertext"`
	KeyVersion       int    `json:"key_version" db:"key_version"`

	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty" db:"updated_by,uuid"`
	SecretUpdatedBy *uuid.UUID `json:"secret_updated_by,omitempty" db:"secret_updated_by,uuid"`

	TMSecretUpdate *time.Time `json:"tm_secret_update,omitempty" db:"tm_secret_update"`
	TMCreate       *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate       *time.Time `json:"tm_update,omitempty" db:"tm_update"`
}

// String implements fmt.Stringer and hides the ciphertext.
func (p Provider) String() string {
	return "Provider{ProviderName:" + string(p.ProviderName) + ", SecretCiphertext:" + redacted + "}"
}

// GoString implements fmt.GoStringer and hides the ciphertext.
func (p Provider) GoString() string {
	return p.String()
}

// Info is the admin-facing view of a provider: the stored row merged with the registry
// defaults and the resolved credential state. It carries no secret material.
type Info struct {
	ProviderName number.ProviderName `json:"provider_name"`

	Enabled  bool              `json:"enabled"`
	Priority int               `json:"priority"`
	Config   map[string]string `json:"config,omitempty"`

	HasSecret        bool             `json:"has_secret"`
	KeyVersion       int              `json:"key_version"`
	CredentialSource CredentialSource `json:"credential_source"`
	Usable           bool             `json:"usable"`

	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
	SecretUpdatedBy *uuid.UUID `json:"secret_updated_by,omitempty"`

	TMUpdate       *time.Time `json:"tm_update,omitempty"`
	TMSecretUpdate *time.Time `json:"tm_secret_update,omitempty"`
}
