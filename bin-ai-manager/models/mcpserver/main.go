package mcpserver

import (
	"time"

	"monorepo/bin-common-handler/models/identity"
)

// AuthType defines how the outbound MCP call authenticates against the
// customer's server.
type AuthType string

const (
	AuthTypeNone   AuthType = ""        // no Authorization header sent
	AuthTypeBearer AuthType = "bearer"  // Authorization: Bearer ***
	AuthTypeAPIKey AuthType = "api_key" // <APIKeyHeader>: <secret>
)

var validAuthTypes = map[AuthType]bool{
	AuthTypeNone: true, AuthTypeBearer: true, AuthTypeAPIKey: true,
}

// IsValid returns true if the AuthType is a known valid value.
func (t AuthType) IsValid() bool {
	return validAuthTypes[t]
}

// Status defines the customer-facing lifecycle state of a registered MCP
// server.
type Status string

const (
	StatusActive Status = "active"
	// StatusDisabled = customer explicitly deactivated; excluded from
	// ListTools/CallTool.
	StatusDisabled Status = "disabled"
)

var validStatuses = map[Status]bool{
	StatusActive: true, StatusDisabled: true,
}

// IsValid returns true if the Status is a known valid value.
func (s Status) IsValid() bool {
	return validStatuses[s]
}

// McpServer represents a customer-registered remote MCP (Model Context
// Protocol) server. See docs/plans/2026-09-11-mcp-tool-integration-design.md
// §5.
type McpServer struct {
	identity.Identity // ID, CustomerID

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	URL    string `json:"url,omitempty" db:"url"` // Streamable-HTTP MCP endpoint, https only
	Status Status `json:"status,omitempty" db:"status"`

	AuthType     AuthType `json:"auth_type,omitempty" db:"auth_type"`
	APIKeyHeader string   `json:"api_key_header,omitempty" db:"api_key_header"` // only meaningful for AuthTypeAPIKey, e.g. "X-API-Key"

	// SecretCiphertext is the AES-256-GCM-encrypted bearer token / API key.
	// NEVER included in WebhookMessage or any GET response body -- see webhook.go.
	SecretCiphertext []byte `json:"-" db:"secret_ciphertext"`
	SecretNonce      []byte `json:"-" db:"secret_nonce"`
	// KeyVersion records which entry of MCP_SECRET_ENCRYPTION_KEYS encrypted
	// this row's secret, so rotating the configured key set never breaks
	// decryption of rows encrypted under a still-configured older version.
	KeyVersion int  `json:"-" db:"key_version"`
	HasSecret  bool `json:"has_secret" db:"-"` // derived, exposed instead of the secret itself

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
