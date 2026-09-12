package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// V1DataMcpServersPost is
// v1 data type request struct for
// /v1/mcp_servers POST
type V1DataMcpServersPost struct {
	CustomerID   uuid.UUID          `json:"customer_id,omitempty"`
	Name         string             `json:"name,omitempty"`
	Detail       string             `json:"detail,omitempty"`
	URL          string             `json:"url,omitempty"`
	AuthType     mcpserver.AuthType `json:"auth_type,omitempty"`
	APIKeyHeader string             `json:"api_key_header,omitempty"`
	Secret       string             `json:"secret,omitempty"`
}

// V1DataMcpServersIDPut is
// v1 data type request struct for
// /v1/mcp_servers/<mcp-server-id> PUT
//
// Every field is a pointer: nil means "leave the existing value
// untouched", matching the OpenAPI-layer contract this DTO carries over
// the RabbitMQ hop. This mirrors the pre-existing Secret pointer pattern,
// extended to the other six mutable fields -- see
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md.
// SECURITY-CRITICAL: do not change Secret back to a plain string.
type V1DataMcpServersIDPut struct {
	Name         *string             `json:"name,omitempty"`
	Detail       *string             `json:"detail,omitempty"`
	URL          *string             `json:"url,omitempty"`
	Status       *mcpserver.Status   `json:"status,omitempty"`
	AuthType     *mcpserver.AuthType `json:"auth_type,omitempty"`
	APIKeyHeader *string             `json:"api_key_header,omitempty"`
	Secret       *string             `json:"secret,omitempty"`
}

// V1DataMcpServersOAuthStartPost is the v1 request struct for
// /v1/mcp_servers/oauth/start POST -- backs AIV1McpOAuthStart. See
// docs/plans/2026-09-12-mcp-server-oauth-support-design.md §10.
type V1DataMcpServersOAuthStartPost struct {
	CustomerID  uuid.UUID  `json:"customer_id,omitempty"`
	Vendor      string     `json:"vendor,omitempty"`
	McpServerID *uuid.UUID `json:"mcp_server_id,omitempty"`
}

// V1DataMcpServersOAuthCallbackGet is the v1 request struct for
// /v1/mcp_servers/oauth/callback GET -- backs the thin existence-check
// RPC behind the public callback relay (design §7a, §10).
type V1DataMcpServersOAuthCallbackGet struct {
	State string `json:"state,omitempty"`
}

// V1DataMcpServersOAuthCompletePost is the v1 request struct for
// /v1/mcp_servers/oauth/complete POST -- backs AIV1McpOAuthComplete.
type V1DataMcpServersOAuthCompletePost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	State      string    `json:"state,omitempty"`
	Code       string    `json:"code,omitempty"`
}
