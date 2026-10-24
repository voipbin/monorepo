package requesthandler

import (
	"context"
	"testing"

	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

const aiQueue = "bin-manager.ai-manager.request"

// synthetic values, assigned apart from any address literal.
const (
	mcpTestSecret     = "test-secret-not-real"
	mcpTestLinkToken1 = "link-token-0001"
	mcpTestLinkToken2 = "link-token-0002"
)

func Test_AIMcpServerWrappers(t *testing.T) {
	if aiQueue != string(outline.QueueNameAIRequest) {
		t.Fatalf("Wrong match. unexpected ai queue constant: %s", outline.QueueNameAIRequest)
	}

	id := uuid.FromStringOrNil("8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11")
	customerID := uuid.FromStringOrNil("1d7e0a3c-5b94-4f2a-8c61-7e2b9d4a6f30")
	serverID := uuid.FromStringOrNil("c4f2d8a9-3e17-4b6a-a0d5-9b1e7c3f2a48")

	name := "synthetic name"
	empty := ""
	status := ammcpserver.StatusDisabled
	authType := ammcpserver.AuthTypeAPIKey

	cases := []rpcCase{
		{
			name:           "list",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"status":"active"}`,
			responseData:   `[{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11","name":"first"},{"id":"c4f2d8a9-3e17-4b6a-a0d5-9b1e7c3f2a48","has_secret":true}]`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpServerList(context.Background(), "2020-09-20T03:23:20.995000Z", 10, map[ammcpserver.Field]any{ammcpserver.FieldStatus: ammcpserver.StatusActive})
			},
			expectRes: []*ammcpserver.McpServer{
				{Identity: identity.Identity{ID: id}, Name: "first"},
				{Identity: identity.Identity{ID: serverID}, HasSecret: true},
			},
		},
		{
			name:           "get",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11","url":"https://mcp.example.invalid/v1","status":"active"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpServerGet(context.Background(), id)
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}, URL: "https://mcp.example.invalid/v1", Status: ammcpserver.StatusActive},
		},
		{
			name:           "create",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"1d7e0a3c-5b94-4f2a-8c61-7e2b9d4a6f30","name":"srv","detail":"d","url":"https://mcp.example.invalid/v1","auth_type":"api_key","api_key_header":"X-Test-Key","secret":"` + mcpTestSecret + `"}`,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11","name":"srv","auth_type":"api_key","api_key_header":"X-Test-Key","has_secret":true}`,
			call: func(h *requestHandler) (any, error) {
				secret := mcpTestSecret
				return h.AIV1McpServerCreate(context.Background(), customerID, "srv", "d", "https://mcp.example.invalid/v1", ammcpserver.AuthTypeAPIKey, "X-Test-Key", secret)
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}, Name: "srv", AuthType: ammcpserver.AuthTypeAPIKey, APIKeyHeader: "X-Test-Key", HasSecret: true},
		},
		{
			name:           "update_partial",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"name":"synthetic name","detail":"","status":"disabled","auth_type":"api_key"}`,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11","name":"synthetic name","status":"disabled"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpServerUpdate(context.Background(), id, &name, &empty, nil, &status, &authType, nil, nil)
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}, Name: "synthetic name", Status: ammcpserver.StatusDisabled},
		},
		{
			name:           "update_nothing",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{}`,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpServerUpdate(context.Background(), id, nil, nil, nil, nil, nil, nil, nil)
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}},
		},
		{
			name:           "delete",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpServerDelete(context.Background(), id)
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}},
		},
		{
			name:           "oauth_start_new_server",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/oauth/start",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"1d7e0a3c-5b94-4f2a-8c61-7e2b9d4a6f30","vendor":"vendor-a"}`,
			responseData:   `{"authorize_url":"https://auth.example.invalid/authorize?x=1","link_token":"` + mcpTestLinkToken1 + `"}`,
			call: func(h *requestHandler) (any, error) {
				authorizeURL, linkToken, err := h.AIV1McpOAuthStart(context.Background(), customerID, "vendor-a", nil)
				return [2]string{authorizeURL, linkToken}, err
			},
			expectRes: [2]string{"https://auth.example.invalid/authorize?x=1", "link-token-0001"},
		},
		{
			name:           "oauth_start_existing_server",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/oauth/start",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"1d7e0a3c-5b94-4f2a-8c61-7e2b9d4a6f30","vendor":"vendor-a","mcp_server_id":"c4f2d8a9-3e17-4b6a-a0d5-9b1e7c3f2a48"}`,
			responseData:   `{"authorize_url":"https://auth.example.invalid/authorize","link_token":"` + mcpTestLinkToken2 + `"}`,
			call: func(h *requestHandler) (any, error) {
				authorizeURL, linkToken, err := h.AIV1McpOAuthStart(context.Background(), customerID, "vendor-a", &serverID)
				return [2]string{authorizeURL, linkToken}, err
			},
			expectRes: [2]string{"https://auth.example.invalid/authorize", "link-token-0002"},
		},
		{
			name:           "oauth_callback_exists",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/oauth/callback?state=a+b%26c%3Dd",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"exists":true}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpOAuthCallback(context.Background(), "a b&c=d")
			},
			expectRes: true,
		},
		{
			name:           "oauth_complete",
			queue:          aiQueue,
			expectURI:      "/v1/mcp_servers/oauth/complete",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"1d7e0a3c-5b94-4f2a-8c61-7e2b9d4a6f30","state":"state-0001","code":"code-0001"}`,
			responseData:   `{"id":"8a5c6b1e-2f0b-4c8e-9d3a-0c9f5a1b7e11","oauth_vendor":"vendor-a"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1McpOAuthComplete(context.Background(), customerID, "state-0001", "code-0001")
			},
			expectRes: &ammcpserver.McpServer{Identity: identity.Identity{ID: id}, OAuthVendor: "vendor-a"},
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.AIV1McpServerList(ctx, "", 10, map[ammcpserver.Field]any{ammcpserver.FieldStatus: make(chan int)})
	})
}
