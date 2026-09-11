package response

// V1ResponseMcpServersOAuthStartPost is the v1 response type for
// /v1/mcp_servers/oauth/start POST. This is a genuinely wire-only shape
// (style B, no single domain type equals it): authorize_url and
// link_token are ephemeral flow values, not persisted fields of any
// domain model. See docs/plans/2026-09-12-mcp-server-oauth-support-design.md §10.
type V1ResponseMcpServersOAuthStartPost struct {
	AuthorizeURL string `json:"authorize_url"`
	LinkToken    string `json:"link_token"`
}

// V1ResponseMcpServersOAuthCallbackGet is the v1 response type for the
// thin existence-check RPC behind the public callback relay.
type V1ResponseMcpServersOAuthCallbackGet struct {
	Exists bool `json:"exists"`
}
