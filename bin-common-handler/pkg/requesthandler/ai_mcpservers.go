package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
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
