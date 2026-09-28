package mcptoolhandler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

// MCP Streamable HTTP transport, as the reference server implements it.
// The requirement numbers below refer to section 4b of
// docs/plans/2026-09-27-mcp-phase2-tool-exposure-analysis-v2.md.
const (
	headerSessionID       = "Mcp-Session-Id"
	headerProtocolVersion = "MCP-Protocol-Version"

	// acceptHeader must list both types; a conformant server answers 406
	// otherwise (requirement 1).
	acceptHeader = "application/json, text/event-stream"

	// requestedProtocolVersion is what initialize asks for. The server
	// answers with the version it will actually speak, which is the only
	// value this client ever sends back (requirement 5).
	requestedProtocolVersion = "2025-11-25"

	clientName    = "voipbin-ai-manager"
	clientVersion = "1.0.0"

	// maxSessionIDBytes bounds the session id a server may hand back. The
	// id is echoed on every later request, so an unbounded one would be
	// reflected into our own request headers.
	maxSessionIDBytes = 1024

	// sessionCloseTimeout bounds the best-effort DELETE that ends a session.
	// It runs after the call's own deadline may already have passed, so it
	// has a deadline of its own.
	sessionCloseTimeout = 2 * time.Second
)

// handshakeProtocolVersions are the protocol versions negotiated through the
// initialize + session handshake this client implements. A server that
// negotiates anything else is on a different transport era: the reference
// server routes a request carrying such a version to a different handler that
// rejects this client's envelopes with 400, so driving it would fail every
// call and the resolver would silently swallow it as an empty tool list.
// Refusing it here turns that into one clear error (requirement 5).
var handshakeProtocolVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
	"2025-11-25": true,
}

// jsonRPCNotification is a JSON-RPC message with no id, which the server
// must not answer with a result.
type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// initializeParams always carries all three members (requirement 4).
type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      clientInfo     `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// httpStatusError is a non-2xx answer. Body is a bounded prefix of the
// response, kept because the 406 and 400 a misconfigured client receives are
// only diagnosable from it (requirement 10).
type httpStatusError struct {
	StatusCode int
	Body       []byte
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("unexpected status code %d, body: %s", e.StatusCode, truncateForError(e.Body))
}

// mcpSession is one initialize + session handshake. It lives for exactly one
// ListTools or CallTool call as a local variable and is never stored or
// cached: a cached session id would outlive the credential that opened it
// (requirement 7).
type mcpSession struct {
	client *http.Client
	server *mcpserver.McpServer

	authName  string
	authValue string

	// id is the Mcp-Session-Id the server issued. Empty means the server is
	// stateless; there is deliberately no separate code path for that
	// (requirement 6).
	id string

	// protocolVersion is the version the server negotiated, always one of
	// handshakeProtocolVersions once the session is open.
	protocolVersion string

	lastID int
}

// nextID allocates a monotonic JSON-RPC request id, so each response can be
// matched to the request that produced it (requirement 9).
func (s *mcpSession) nextID() int {
	s.lastID++
	return s.lastID
}

// openSession runs the handshake: initialize, check the JSON-RPC result
// rather than the HTTP status, adopt the negotiated protocol version and the
// session id, then send notifications/initialized.
func (h *mcpToolHandler) openSession(ctx context.Context, client *http.Client, m *mcpserver.McpServer) (*mcpSession, error) {
	authName, authValue, err := h.buildAuthHeader(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("could not build auth header: %w", err)
	}

	s := &mcpSession{
		client:    client,
		server:    m,
		authName:  authName,
		authValue: authValue,
	}

	params := initializeParams{
		ProtocolVersion: requestedProtocolVersion,
		Capabilities:    map[string]any{},
		ClientInfo:      clientInfo{Name: clientName, Version: clientVersion},
	}

	// A malformed initialize is answered with HTTP 200, a JSON-RPC error AND
	// a usable session id, so success is decided by the JSON-RPC result
	// alone (requirement 3). request() fails on any JSON-RPC error member.
	result, header, err := h.request(ctx, s, "initialize", params)

	// Adopt the session id before judging the result: a server that issued
	// one alongside an error still holds that session open, and it is closed
	// below rather than left to the server's idle timeout.
	if header != nil {
		sessionID := header.Get(headerSessionID)
		if errID := validateSessionID(sessionID); errID != nil {
			return nil, errID
		}
		s.id = sessionID
	}

	if err != nil {
		h.closeSession(s)
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	var res initializeResult
	if errUnmarshal := json.Unmarshal(result, &res); errUnmarshal != nil {
		h.closeSession(s)
		return nil, fmt.Errorf("could not parse initialize result: %w", errUnmarshal)
	}
	if !handshakeProtocolVersions[res.ProtocolVersion] {
		h.closeSession(s)
		return nil, fmt.Errorf("server negotiated protocol version %q, which is not a handshake-era version this client can drive", truncateUTF8(res.ProtocolVersion, 64))
	}

	s.protocolVersion = res.ProtocolVersion

	if err := h.notify(ctx, s, "notifications/initialized"); err != nil {
		h.closeSession(s)
		return nil, fmt.Errorf("initialized notification failed: %w", err)
	}

	return s, nil
}

// closeSession ends a stateful session with a best-effort DELETE, as the
// transport asks of a client that no longer needs one. Without it every call
// would leave a session open on the customer's server until its idle timeout,
// and the reference server refuses all new sessions, from every client, once
// its session limit is reached. A stateless session (no id) has nothing to
// close. Failure is logged and never fails the call; 405 means the server
// does not support explicit termination, which is allowed.
func (h *mcpToolHandler) closeSession(s *mcpSession) {
	if s == nil || s.id == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sessionCloseTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.server.URL, nil)
	if err != nil {
		return
	}
	h.setSessionHeaders(req, s)

	resp, err := s.client.Do(req)
	if err != nil {
		logrus.Debugf("Could not close mcp session. err: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBodyBytes))

	if resp.StatusCode != http.StatusMethodNotAllowed && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		logrus.Debugf("Mcp server did not close the session. status: %d", resp.StatusCode)
	}
}

// validateSessionID enforces the transport's rule that a session id is
// visible ASCII (0x21 to 0x7E), plus a length bound, before the id is echoed
// back in a request header.
func validateSessionID(id string) error {
	if len(id) > maxSessionIDBytes {
		return fmt.Errorf("server issued a session id of %d bytes, over the %d byte limit", len(id), maxSessionIDBytes)
	}
	for i := 0; i < len(id); i++ {
		if id[i] < 0x21 || id[i] > 0x7E {
			return fmt.Errorf("server issued a session id containing a byte outside visible ASCII")
		}
	}
	return nil
}

// request sends one JSON-RPC request on the session and returns the result
// of the response whose id matches it.
func (h *mcpToolHandler) request(ctx context.Context, s *mcpSession, method string, params any) (json.RawMessage, http.Header, error) {
	id := s.nextID()
	msg := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	resp, err := h.post(ctx, s, msg)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body := io.LimitReader(resp.Body, mcpserverhandler.McpHTTPResponseSizeCapBytes)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, statusError(resp.StatusCode, body)
	}

	rpcResp, err := readJSONRPCResponse(resp.Header.Get("Content-Type"), body, id)
	if err != nil {
		return nil, resp.Header, err
	}
	if rpcResp.Error != nil {
		return nil, resp.Header, fmt.Errorf("JSON-RPC error (code %d): %s", rpcResp.Error.Code, truncateForError([]byte(rpcResp.Error.Message)))
	}
	if len(rpcResp.Result) == 0 || bytes.Equal(bytes.TrimSpace(rpcResp.Result), []byte("null")) {
		return nil, resp.Header, fmt.Errorf("JSON-RPC response has no result")
	}

	return rpcResp.Result, resp.Header, nil
}

// notify sends a JSON-RPC notification. The server acknowledges it with an
// empty 2xx (202 per the transport); any body is ignored.
func (h *mcpToolHandler) notify(ctx context.Context, s *mcpSession, method string) error {
	resp, err := h.post(ctx, s, jsonRPCNotification{JSONRPC: "2.0", Method: method})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body := io.LimitReader(resp.Body, mcpserverhandler.McpHTTPResponseSizeCapBytes)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return statusError(resp.StatusCode, body)
	}
	_, _ = io.Copy(io.Discard, body)

	return nil
}

// post sends one message with the headers every request on the session
// carries. The session and protocol headers are absent on initialize because
// neither is known yet.
func (h *mcpToolHandler) post(ctx context.Context, s *mcpSession, msg any) (*http.Response, error) {
	bodyBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.server.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	h.setSessionHeaders(req, s)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// setSessionHeaders applies the headers every request on a session carries:
// the Accept pair, the session id and negotiated protocol version once known,
// and the credential. initialize carries neither of the first two because
// neither is known yet.
func (h *mcpToolHandler) setSessionHeaders(req *http.Request, s *mcpSession) {
	req.Header.Set("Accept", acceptHeader)
	if s.id != "" {
		req.Header.Set(headerSessionID, s.id)
	}
	if s.protocolVersion != "" {
		req.Header.Set(headerProtocolVersion, s.protocolVersion)
	}
	if s.authName != "" {
		req.Header.Set(s.authName, s.authValue)
	}
}

// statusError reads a bounded prefix of a non-2xx body into the error, so the
// reason the server gave survives into the log (requirement 10).
func statusError(code int, body io.Reader) error {
	prefix, _ := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	return &httpStatusError{StatusCode: code, Body: prefix}
}

// maxErrorBodyBytes bounds how much of a non-2xx body is read. It is larger
// than the truncation applied when rendering the error, so the rendered text
// is always cut on a rune boundary rather than by the read.
const maxErrorBodyBytes = 4096

// readJSONRPCResponse decodes the response to request wantID from either a
// bare JSON body or an SSE stream (requirement 8). On a stream it returns as
// soon as the matching event arrives instead of waiting for the server to
// close it, and it skips events that are notifications, server-to-client
// requests or responses to other ids.
func readJSONRPCResponse(contentType string, body io.Reader, wantID int) (*jsonRPCResponse, error) {
	mediaType, _, _ := mime.ParseMediaType(contentType)

	if mediaType == "text/event-stream" {
		return readSSEResponse(body, wantID)
	}

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	resp, err := decodeResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON-RPC response: %w", err)
	}
	if !answers(resp, wantID) {
		return nil, fmt.Errorf("JSON-RPC response id %s does not match request id %d", truncateForError(resp.ID), wantID)
	}

	return resp, nil
}

func readSSEResponse(body io.Reader, wantID int) (*jsonRPCResponse, error) {
	reader := bufio.NewReader(body)
	var data strings.Builder
	hasData := false

	dispatch := func() *jsonRPCResponse {
		defer func() {
			data.Reset()
			hasData = false
		}()
		if !hasData {
			return nil
		}
		resp, err := decodeResponse([]byte(data.String()))
		if err != nil || !answers(resp, wantID) {
			return nil
		}
		return resp
	}

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			switch {
			case line == "":
				if resp := dispatch(); resp != nil {
					return resp, nil
				}
			case line == "data" || strings.HasPrefix(line, "data:"):
				// A field line without a colon has an empty value, per the
				// SSE specification. Bare CR line endings are not handled:
				// the reader splits on LF, which covers LF and CRLF, the only
				// endings MCP servers emit.
				if hasData {
					data.WriteByte('\n')
				}
				data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
				hasData = true
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("could not read event stream: %w", err)
			}
			if resp := dispatch(); resp != nil {
				return resp, nil
			}
			return nil, fmt.Errorf("event stream ended without a response to request id %d", wantID)
		}
	}
}

// decodeResponse parses one JSON-RPC message and rejects anything that is not
// a response (a request or notification carries a method).
func decodeResponse(raw []byte) (*jsonRPCResponse, error) {
	var resp jsonRPCResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Method != "" {
		return nil, fmt.Errorf("message is a %q request, not a response", resp.Method)
	}
	return &resp, nil
}

// answers reports whether resp is the answer to request want. Besides a
// matching id, an error with a null id counts: JSON-RPC uses a null id when
// the server could not determine the request id, and dropping it would lose
// the only reason the server gave.
func answers(resp *jsonRPCResponse, want int) bool {
	id := string(bytes.TrimSpace(resp.ID))
	if id == strconv.Itoa(want) {
		return true
	}
	return resp.Error != nil && (id == "" || id == "null")
}

// truncateUTF8 cuts s to at most maxBytes without splitting a multi-byte
// character, so a truncated log line stays valid UTF-8 (requirement 12).
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
