package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// mcpServerGet returns the McpServer info without a permission check.
func (h *serviceHandler) mcpServerGet(ctx context.Context, id uuid.UUID) (*ammcpserver.McpServer, error) {
	res, err := h.reqHandler.AIV1McpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get mcp server info")
	}

	return res, nil
}

// McpServerCreate registers a new customer-owned MCP server.
func (h *serviceHandler) McpServerCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	detail string,
	url string,
	authType ammcpserver.AuthType,
	apiKeyHeader string,
	secret string,
) (*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.AIV1McpServerCreate(ctx, a.CustomerID, name, detail, url, authType, apiKeyHeader, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create mcp server")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// McpServerGetsByCustomerID returns a paginated list of MCP servers for the authenticated customer.
func (h *serviceHandler) McpServerGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	filters := map[string]string{
		"deleted":     "false",
		"customer_id": a.CustomerID.String(),
	}

	typedFilters, err := h.convertMcpServerFilters(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert mcp server filters")
	}

	tmps, err := h.reqHandler.AIV1McpServerList(ctx, token, size, typedFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get mcp servers info")
	}

	res := make([]*ammcpserver.WebhookMessage, 0, len(tmps))
	for _, t := range tmps {
		res = append(res, t.ConvertWebhookMessage())
	}

	return res, nil
}

// convertMcpServerFilters converts map[string]string to map[ammcpserver.Field]any.
func (h *serviceHandler) convertMcpServerFilters(filters map[string]string) (map[ammcpserver.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, ammcpserver.McpServer{})
	if err != nil {
		return nil, err
	}

	result := make(map[ammcpserver.Field]any, len(typed))
	for k, v := range typed {
		result[ammcpserver.Field(k)] = v
	}

	return result, nil
}

// McpServerGet returns a single MCP server by ID after checking ownership.
func (h *serviceHandler) McpServerGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.mcpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get mcp server info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return tmp.ConvertWebhookMessage(), nil
}

// McpServerUpdate updates an MCP server after checking ownership.
//
// Every field is a pointer per design §10's PUT semantics (extended to
// all mutable fields in
// docs/plans/2026-09-12-mcp-server-put-partial-update-design.md): nil
// means "leave the existing value untouched", a non-nil pointer to "" on
// secret means "explicitly clear the secret", and a non-nil pointer to a
// value means "set to this value". SECURITY-CRITICAL: do not change
// secret back to a plain string.
func (h *serviceHandler) McpServerUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
	name *string,
	detail *string,
	url *string,
	status *ammcpserver.Status,
	authType *ammcpserver.AuthType,
	apiKeyHeader *string,
	secret *string,
) (*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.mcpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get mcp server info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1McpServerUpdate(ctx, id, name, detail, url, status, authType, apiKeyHeader, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update mcp server")
	}

	return res.ConvertWebhookMessage(), nil
}

// McpServerDelete soft-deletes an MCP server after checking ownership.
func (h *serviceHandler) McpServerDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.mcpServerGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get mcp server info")
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1McpServerDelete(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete mcp server")
	}

	return res.ConvertWebhookMessage(), nil
}

// McpOAuthStart starts the vendor OAuth flow for the authenticated
// customer. If mcpServerID is non-nil (reconnect), ownership is verified
// at the ai-manager layer (design §9); a caller-side ownership check is
// deliberately not duplicated here.
func (h *serviceHandler) McpOAuthStart(ctx context.Context, a *auth.AuthIdentity, vendor string, mcpServerID *uuid.UUID) (string, string, error) {
	if a.IsDirect() {
		return "", "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return "", "", serviceerrors.ErrPermissionDenied
	}

	authorizeURL, linkToken, err := h.reqHandler.AIV1McpOAuthStart(ctx, a.CustomerID, vendor, mcpServerID)
	if err != nil {
		return "", "", errors.Wrapf(err, "could not start mcp oauth flow")
	}

	return authorizeURL, linkToken, nil
}

// McpOAuthCallback is the PUBLIC (unauthenticated) thin existence check
// backing GET /mcpservers/oauth/callback's redirect relay (design §7a
// Layer 1). It never mutates anything.
func (h *serviceHandler) McpOAuthCallback(ctx context.Context, state string) (bool, error) {
	exists, err := h.reqHandler.AIV1McpOAuthCallback(ctx, state)
	if err != nil {
		return false, errors.Wrapf(err, "could not check mcp oauth callback state")
	}

	return exists, nil
}

// McpOAuthComplete completes the OAuth flow for the authenticated
// customer. The real security boundary (state row's customer_id must
// match the JWT-authenticated caller) is enforced at the ai-manager
// layer (design §7a); this layer only needs the standard
// permission/direct-access checks.
func (h *serviceHandler) McpOAuthComplete(ctx context.Context, a *auth.AuthIdentity, state string, code string) (*ammcpserver.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.AIV1McpOAuthComplete(ctx, a.CustomerID, state, code)
	if err != nil {
		return nil, errors.Wrapf(err, "could not complete mcp oauth flow")
	}

	return res.ConvertWebhookMessage(), nil
}
