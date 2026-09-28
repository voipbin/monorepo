package mcptoolhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
// decoded per-call into the shape the caller expects. ID stays raw so it is
// compared exactly rather than coerced, and Method is decoded only to tell a
// server-sent request or notification apart from a response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
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
func (h *mcpToolHandler) buildAuthHeader(ctx context.Context, m *mcpserver.McpServer) (headerName string, headerValue string, err error) {
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

	case mcpserver.AuthTypeOAuth:
		// design docs/plans/2026-09-12-mcp-server-oauth-support-design.md
		// §8: resolves a currently-valid access token, transparently
		// refreshing it first via the vendor's token endpoint if needed.
		if h.oauthHandler == nil {
			return "", "", fmt.Errorf("mcp server uses oauth auth but no oauth handler is configured")
		}
		token, err := h.oauthHandler.GetValidAccessToken(ctx, m)
		if err != nil {
			return "", "", fmt.Errorf("could not get valid oauth access token: %w", err)
		}
		return "Authorization", "Bearer " + token, nil

	default:
		return "", "", fmt.Errorf("unsupported auth type: %q", m.AuthType)
	}
}

// doJSONRPCRequest runs one MCP method on a fresh session: initialize, the
// method itself, and a best-effort session close. The session is scoped to
// this call and dropped when it returns (requirement 7 in section 4b of
// docs/plans/2026-09-27-mcp-phase2-tool-exposure-analysis-v2.md).
//
// h.timeout bounds the whole call, not each request. A call is several
// requests, and a per-request timeout would let a slow server hold one call
// for a multiple of the configured limit on the session-start path. The only
// work that may run past it is the session close's short floor
// (sessionCloseFloor). An OAuth refresh in progress keeps running after the
// call gives up, so a rotated token is still stored, but the call itself
// does not wait for it; see mcpoauthhandler.GetValidAccessToken.
//
// A 404 on the method means the server discarded the session. That is retried
// exactly once with a new session (requirement 11): the retry is straight-line
// code, not a loop, and its own failure is returned. A 404 on initialize is
// not retried: it means the URL is wrong, not that a session expired.
func (h *mcpToolHandler) doJSONRPCRequest(ctx context.Context, m *mcpserver.McpServer, method string, params any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	newClient := h.newClient
	if newClient == nil {
		newClient = mcpserverhandler.NewSSRFGuardedClient
	}
	client := newClient(h.timeout)
	// Each call builds its own client and transport, so its idle
	// connections would otherwise outlive it indefinitely.
	defer client.CloseIdleConnections()

	// Resolved once and reused by the replacement session below; see
	// openSession.
	authName, authValue, err := h.buildAuthHeader(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: could not build auth header: %w", err)
	}

	session, err := h.openSession(ctx, client, m, authName, authValue)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: %w", err)
	}

	result, _, err := h.request(ctx, session, method, params)

	if sessionGone(session, err) {
		// The server already discarded this session, so it is not closed.
		session, err = h.openSession(ctx, client, m, authName, authValue)
		if err != nil {
			return nil, fmt.Errorf("mcptoolhandler: could not re-initialize after the session expired: %w", err)
		}
		result, _, err = h.request(ctx, session, method, params)
	}

	if !sessionGone(session, err) {
		h.closeSession(ctx, session)
	}

	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler: %s failed: %w", method, err)
	}

	return result, nil
}

// sessionGone reports whether err is the server saying it no longer knows the
// session: a 404 on a request that carried a session id. A stateless server
// has no session, so its 404 means something else and is final.
func sessionGone(s *mcpSession, err error) bool {
	var statusErr *httpStatusError
	return err != nil && s.id != "" && errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound
}

// truncateForError caps error-message body echoes to avoid dumping
// unbounded remote content into logs/errors. It cuts on a rune boundary so
// the result stays valid UTF-8.
func truncateForError(b []byte) string {
	const maxBytes = 512
	s := string(b)
	if len(s) > maxBytes {
		return truncateUTF8(s, maxBytes) + "...(truncated)"
	}
	return s
}

// refuseDeleted stops a transport call to a soft-deleted MCP server.
//
// This is the last line of defence, not the primary one: resolveTools and
// toolHandleMcpCall already refuse deleted servers with the AI's customer in
// hand, which this layer does not have. It exists because McpServerGet returns
// soft-deleted rows on purpose, so without it any present or future caller
// that reaches the transport directly would happily open a connection to a
// server the customer has revoked -- and the delete path has already zeroed
// the credentials, so the attempt would authenticate as nobody.
func refuseDeleted(op string, m *mcpserver.McpServer) error {
	if m == nil {
		return fmt.Errorf("mcptoolhandler.%s: mcp server not found", op)
	}
	if m.TMDelete != nil {
		return fmt.Errorf("mcptoolhandler.%s: mcp server is deleted", op)
	}

	return nil
}

// ListTools sends an MCP tools/list request to the server identified by
// serverID and returns its advertised tools.
func (h *mcpToolHandler) ListTools(ctx context.Context, serverID uuid.UUID) ([]McpTool, error) {
	m, err := h.db.McpServerGet(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: could not get mcp server: %w", err)
	}
	if err := refuseDeleted("ListTools", m); err != nil {
		return nil, err
	}

	result, err := h.doJSONRPCRequest(ctx, m, "tools/list", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: %w", err)
	}

	tools, truncated, err := decodeToolsList(result, MaxToolsPerServer)
	if err != nil {
		return nil, fmt.Errorf("mcptoolhandler.ListTools: could not parse tools/list result: %w", err)
	}
	if truncated {
		logrus.WithField("mcp_server_id", serverID).Warnf("Mcp server listed more than %d tools; keeping the first %d.", MaxToolsPerServer, MaxToolsPerServer)
	}

	return tools, nil
}

// decodeToolsList reads the tools array of a tools/list result one element
// at a time and stops after max, reporting whether more followed. Decoding
// the whole array first and truncating afterwards would not bound anything:
// a body of empty objects inside the 1 MiB cap is several hundred thousand
// elements, each allocated before the cap could apply. Members other than
// tools are skipped without being decoded into values.
func decodeToolsList(result json.RawMessage, max int) ([]McpTool, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(result))

	if err := expectDelim(dec, '{'); err != nil {
		return nil, false, err
	}

	tools := []McpTool{}
	truncated := false
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, false, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, false, fmt.Errorf("unexpected object key %v", keyTok)
		}

		if key != "tools" {
			if err := skipValue(dec); err != nil {
				return nil, false, err
			}
			continue
		}

		if err := expectDelim(dec, '['); err != nil {
			return nil, false, fmt.Errorf("tools: %w", err)
		}
		for dec.More() {
			if len(tools) >= max {
				truncated = true
				// Stop reading. The rest of the body is not needed and
				// reading it would cost what the cap exists to avoid.
				return tools, truncated, nil
			}
			var t McpTool
			if err := dec.Decode(&t); err != nil {
				return nil, false, fmt.Errorf("tools[%d]: %w", len(tools), err)
			}
			tools = append(tools, t)
		}
		if err := expectDelim(dec, ']'); err != nil {
			return nil, false, fmt.Errorf("tools: %w", err)
		}
	}

	return tools, truncated, nil
}

// expectDelim reads the next token and fails unless it is the delimiter d.
func expectDelim(dec *json.Decoder, d json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if got, ok := tok.(json.Delim); !ok || got != d {
		return fmt.Errorf("expected %q, got %v", d, tok)
	}
	return nil
}

// skipValue consumes one JSON value of any kind by walking its tokens, so a
// large unrelated member is passed over without being built in memory.
func skipValue(dec *json.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		if depth == 0 {
			return nil
		}
	}
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
	if err := refuseDeleted("CallTool", m); err != nil {
		return "", err
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
