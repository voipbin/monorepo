package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	amresponse "monorepo/bin-ai-manager/pkg/listenhandler/models/response"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIV1McpServerList sends a request to ai-manager to get a paginated list of
// customer-registered MCP servers.
func (r *requestHandler) AIV1McpServerList(ctx context.Context, pageToken string, pageSize uint64, filters map[ammcpserver.Field]any) ([]*ammcpserver.McpServer, error) {
	uri := fmt.Sprintf("/v1/mcp_servers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/mcp_servers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1McpServerGet sends a request to ai-manager to get a single MCP server.
func (r *requestHandler) AIV1McpServerGet(ctx context.Context, id uuid.UUID) (*ammcpserver.McpServer, error) {
	uri := fmt.Sprintf("/v1/mcp_servers/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/mcp_servers/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1McpServerCreate sends a request to ai-manager to register a new MCP server.
func (r *requestHandler) AIV1McpServerCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	url string,
	authType ammcpserver.AuthType,
	apiKeyHeader string,
	secret string,
) (*ammcpserver.McpServer, error) {
	uri := "/v1/mcp_servers"

	data := &amrequest.V1DataMcpServersPost{
		CustomerID:   customerID,
		Name:         name,
		Detail:       detail,
		URL:          url,
		AuthType:     authType,
		APIKeyHeader: apiKeyHeader,
		Secret:       secret,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/mcp_servers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1McpServerUpdate sends a request to ai-manager to update an MCP server.
//
// Every field is a pointer per design §10's PUT semantics (extended to
// all mutable fields in
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md): nil
// means "leave the existing value untouched", a non-nil pointer to ""
// means "explicitly clear the secret" (secret only) or "set to the empty
// string" (every other field), and a non-nil pointer to a value means
// "set to this value". SECURITY-CRITICAL: do not change secret back to a
// plain string.
func (r *requestHandler) AIV1McpServerUpdate(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	detail *string,
	url *string,
	status *ammcpserver.Status,
	authType *ammcpserver.AuthType,
	apiKeyHeader *string,
	secret *string,
) (*ammcpserver.McpServer, error) {
	uri := fmt.Sprintf("/v1/mcp_servers/%s", id.String())

	data := &amrequest.V1DataMcpServersIDPut{
		Name:         name,
		Detail:       detail,
		URL:          url,
		Status:       status,
		AuthType:     authType,
		APIKeyHeader: apiKeyHeader,
		Secret:       secret,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPut, "ai/mcp_servers/<id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1McpServerDelete sends a request to ai-manager to soft-delete an MCP server.
func (r *requestHandler) AIV1McpServerDelete(ctx context.Context, id uuid.UUID) (*ammcpserver.McpServer, error) {
	uri := fmt.Sprintf("/v1/mcp_servers/%s", id.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/mcp_servers/<id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1McpOAuthStart sends a request to ai-manager to start the vendor
// OAuth 2.1 authorization-code + PKCE flow for a customer-registered
// McpServer (design docs/plans/2026-09-12-mcp-server-oauth-support-design.md §7).
func (r *requestHandler) AIV1McpOAuthStart(ctx context.Context, customerID uuid.UUID, vendor string, mcpServerID *uuid.UUID) (string, string, error) {
	uri := "/v1/mcp_servers/oauth/start"

	data := &amrequest.V1DataMcpServersOAuthStartPost{
		CustomerID:  customerID,
		Vendor:      vendor,
		McpServerID: mcpServerID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return "", "", err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/mcp_servers/oauth/start", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return "", "", err
	}

	var res amresponse.V1ResponseMcpServersOAuthStartPost
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return "", "", errParse
	}

	return res.AuthorizeURL, res.LinkToken, nil
}

// AIV1McpOAuthCallback sends a request to ai-manager backing the PUBLIC
// GET /mcpservers/oauth/callback relay's thin, no-mutation existence
// check (design §7a Layer 1, §10).
func (r *requestHandler) AIV1McpOAuthCallback(ctx context.Context, state string) (bool, error) {
	uri := fmt.Sprintf("/v1/mcp_servers/oauth/callback?state=%s", url.QueryEscape(state))

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/mcp_servers/oauth/callback", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return false, err
	}

	var res amresponse.V1ResponseMcpServersOAuthCallbackGet
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return false, errParse
	}

	return res.Exists, nil
}

// AIV1McpOAuthComplete sends a request to ai-manager to complete the
// OAuth flow: verify state ownership, exchange the authorization code
// for tokens, and create/update the McpServer row (design §7 steps 9-12).
func (r *requestHandler) AIV1McpOAuthComplete(ctx context.Context, customerID uuid.UUID, state string, code string) (*ammcpserver.McpServer, error) {
	uri := "/v1/mcp_servers/oauth/complete"

	data := &amrequest.V1DataMcpServersOAuthCompletePost{
		CustomerID: customerID,
		State:      state,
		Code:       code,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/mcp_servers/oauth/complete", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res ammcpserver.McpServer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
