package mcptoolhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

// jsonRPCRequest is the outbound MCP JSON-RPC 2.0 envelope.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// jsonRPCError is the JSON-RPC 2.0 error object.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// jsonRPCResponse is the generic inbound JSON-RPC 2.0 envelope; Result is
// decoded per-call into the shape the caller expects.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

// toolsListResult is the result payload of a tools/list call.
type toolsListResult struct {
	Tools []McpTool `json:"tools"`
}

// toolsCallParams is the params payload of a tools/call request.
type toolsCallParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// mcpContentItem is one element of a tools/call result's content array.
type mcpContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// toolsCallResult is the result payload of a tools/call call.
type toolsCallResult struct {
	Content []mcpContentItem `json:"content"`
	IsError bool             `json:"isError"`
}

// buildAuthHeader returns the header name/value to set on the outbound
// request per the server's AuthType, decrypting the stored secret via
// h.crypto.Decrypt. Returns ("", "", nil) for AuthTypeNone.
func (h *mcpToolHandler) buildAuthHeader(m *mcpserver.McpServer) (headerName string, headerValue string, err error) {
	switch m.AuthType {
	case mcpserver.AuthTypeNone:
		return "", "", nil

	case mcpserver.AuthTypeBearer:
		secret, err := h.crypto.Decrypt(m.SecretCiphertext, m.SecretNonce, m.KeyVersion)
		if err != nil {
			return "", "", fmt.Errorf("could not decrypt secret: %w", err)
		}
		return "Authorization", "Bearer " + secret, nil

	case mcpserver.AuthTypeAPIKey:
		secret, err := h.crypto.Decrypt(m.SecretCiphertext, m.SecretNonce, m.KeyVersion)
		if err != nil {
			return "", "", fmt.Errorf("could not decrypt secret: %w", err)
		}
		headerName := m.APIKeyHeader
		if headerName == "" {
			headerName = "X-API-Key"
		}
		return headerName, secret, nil

	default:
		return "", "", fmt.Errorf("unsupported auth type: %q", m.AuthType)
	}
}

// doJSONRPCRequest sends the JSON-RPC envelope to the server's URL using an
// SSRF-guarded client, applies auth per the server's AuthType, caps the
// response body, and decodes the outer JSON-RPC envelope. Returns the raw
// result payload on success.
func (h *mcpToolHandler) doJSONRPCRequest(ctx context.Context, m *mcpserver.McpServer, method string, params any) (json.RawMessage, error) {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not marshal request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	headerName, headerValue, err := h.buildAuthHeader(m)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not build auth header: %w", err)
	}
	if headerName != "" {
		httpReq.Header.Set(headerName, headerValue)
	}

	newClient := h.newClient
	if newClient == nil {
		newClient = mcpserverhandler.NewSSRFGuardedClient
	}
	client := newClient(h.timeout)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, mcpserverhandler.McpHTTPResponseSizeCapBytes)
	respBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcptoolhandler: unexpected status code %d, body: %s", resp.StatusCode, truncateForError(respBytes))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcptoolhandler: JSON-RPC error (code %d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return nil, fmt.Errorf("mcptoolhandler: JSON-RPC response has no result")
	}

	return rpcResp.Result, nil
}

// truncateForError caps error-message body echoes to avoid dumping
// unbounded remote content into logs/errors.
func truncateForError(b []byte) string {
	const cap = 512
	if len(b) > cap {
		return string(b[:cap]) + "...(truncated)"
	}
	return string(b)
}

// ListTools sends an MCP tools/list request to the server identified by
// serverID and returns its advertised tools.
func (h *mcpToolHandler) ListTools(ctx context.Context, serverID uuid.UUID) ([]McpTool, error) {
	m, err := h.db.McpServerGet(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: could not get mcp server: %w", err)
	}

	result, err := h.doJSONRPCRequest(ctx, m, "tools/list", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: %w", err)
	}

	var listResult toolsListResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: could not parse tools/list result: %w", err)
	}

	return listResult.Tools, nil
}

// CallTool sends an MCP tools/call request for toolName on the server
// identified by serverID, with argumentsJSON passed through unchanged as the
// JSON-RPC arguments object, and returns the joined text content of the
// result.
func (h *mcpToolHandler) CallTool(ctx context.Context, serverID uuid.UUID, toolName string, argumentsJSON string) (string, error) {
	m, err := h.db.McpServerGet(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("mcptoolhandler.CallTool: could not get mcp server: %w", err)
	}

	var args any
	if strings.TrimSpace(argumentsJSON) == "" {
		args = map[string]any{}
	} else if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("mcptoolhandler.CallTool: could not parse arguments JSON: %w", err)
	}

	params := toolsCallParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := h.doJSONRPCRequest(ctx, m, "tools/call", params)
	if err != nil {
		return "", fmt.Errorf("mcptoolhandler.CallTool: %w", err)
	}

	var callResult toolsCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return "", fmt.Errorf("mcptoolhandler.CallTool: could not parse tools/call result: %w", err)
	}

	texts := make([]string, 0, len(callResult.Content))
	for _, item := range callResult.Content {
		if item.Text != "" {
			texts = append(texts, item.Text)
		}
	}

	return strings.Join(texts, "\n"), nil
}
