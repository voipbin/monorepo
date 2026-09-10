package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// V1DataMcpServersPost is
// v1 data type request struct for
// /v1/mcp_servers POST
type V1DataMcpServersPost struct {
	CustomerID   uuid.UUID         `json:"customer_id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Detail       string            `json:"detail,omitempty"`
	URL          string            `json:"url,omitempty"`
	AuthType     mcpserver.AuthType `json:"auth_type,omitempty"`
	APIKeyHeader string            `json:"api_key_header,omitempty"`
	Secret       string            `json:"secret,omitempty"`
}

// V1DataMcpServersIDPut is
// v1 data type request struct for
// /v1/mcp_servers/<mcp-server-id> PUT
//
// Secret is a *string per design §10's PUT semantics: nil means "leave the
// existing encrypted secret untouched", a non-nil pointer to "" means
// "explicitly clear the secret", and a non-nil pointer to a value means
// "re-encrypt and replace". SECURITY-CRITICAL: do not change this to a
// plain string.
type V1DataMcpServersIDPut struct {
	Name         string            `json:"name,omitempty"`
	Detail       string            `json:"detail,omitempty"`
	URL          string            `json:"url,omitempty"`
	Status       mcpserver.Status  `json:"status,omitempty"`
	AuthType     mcpserver.AuthType `json:"auth_type,omitempty"`
	APIKeyHeader string            `json:"api_key_header,omitempty"`
	Secret       *string           `json:"secret,omitempty"`
}
