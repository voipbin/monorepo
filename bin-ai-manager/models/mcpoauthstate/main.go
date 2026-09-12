// Package mcpoauthstate holds the short-lived CSRF/PKCE state for an
// in-flight MCP server OAuth authorization request. See
// docs/plans/2026-09-12-mcp-server-oauth-support-design.md §5.
package mcpoauthstate

import (
	"time"

	"github.com/gofrs/uuid"
)

// McpOAuthState represents a single row of ai_mcp_oauth_states -- an
// opaque, single-use, time-bounded token binding an OAuth authorization
// attempt to the customer that started it (design §7a Layer 1).
type McpOAuthState struct {
	// State is the opaque random token used as both the primary key and
	// the OAuth `state` query parameter. Also returned to the caller of
	// AIV1McpOAuthStart as `link_token` (design §7a Layer 2).
	State string `json:"state" db:"state"`

	// CustomerID is the customer that started this flow -- the value
	// AIV1McpOAuthComplete's ownership check compares against the
	// JWT-authenticated caller's customer_id (design §7a's real security
	// boundary).
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	// McpServerID is nil for "create a new server on callback", or the
	// id of an existing row for "attach to this existing server"
	// (reconnect, design §9). A nil pointer here means uuid.Nil is
	// substituted in the DB row (see dbhandler for the exact encoding).
	McpServerID *uuid.UUID `json:"mcp_server_id,omitempty" db:"mcp_server_id,uuid"`

	Vendor string `json:"vendor" db:"vendor"`

	// PKCEVerifier is the PKCE code_verifier, stored in plaintext
	// (single-use, short-lived -- see design §12).
	PKCEVerifier string `json:"-" db:"pkce_verifier"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMExpire *time.Time `json:"tm_expire" db:"tm_expire"`
}

// IsExpired reports whether this state row is past its tm_expire, given
// the current time. A nil TMExpire is treated as already expired (fail
// closed on a malformed row rather than treating it as eternally valid).
func (s *McpOAuthState) IsExpired(now time.Time) bool {
	if s.TMExpire == nil {
		return true
	}
	return !now.Before(*s.TMExpire)
}
