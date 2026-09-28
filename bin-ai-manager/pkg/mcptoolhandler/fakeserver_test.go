package mcptoolhandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// fakeMCPServer is a hermetic MCP Streamable HTTP server that rejects the way
// the reference server does, so a client that skips part of the handshake
// fails here instead of passing against a lenient handler:
//
//   - 406 unless Accept lists both application/json and text/event-stream
//   - 400 on any non-initialize request without the session id initialize issued
//   - 400 when MCP-Protocol-Version is present but is not the negotiated value
//   - tools/list and tools/call answered SSE-framed by default
//
// Each field below switches one behaviour so a single test can target one
// requirement.
type fakeMCPServer struct {
	t *testing.T

	mu sync.Mutex

	// stateless issues no session id and requires none, like the reference
	// server's stateless_http mode.
	stateless bool
	// jsonResponse answers with a bare JSON body instead of an SSE stream.
	jsonResponse bool
	// negotiate is the protocolVersion initialize answers with. Defaults to
	// 2025-11-25.
	negotiate string
	// initializeErrorWithSession reproduces the reference server's answer to
	// a malformed initialize: HTTP 200, a JSON-RPC error and a usable session
	// id all at once.
	initializeErrorWithSession bool
	// expireSessionsOnce answers the first post-initialize request with 404,
	// as a server that dropped the session does.
	expireSessionsOnce bool
	// expireAlways answers every post-initialize request with 404.
	expireAlways bool
	// sseNoise prepends a notification and an unrelated response to every
	// SSE answer, to prove the client picks the event matching its id.
	sseNoise bool
	// statusOnMethod answers the method call (not the handshake) with this
	// HTTP status and body.
	statusOnMethod int
	statusBody     string

	tools      string
	callResult string

	sessions int
	requests []recordedRequest
}

type recordedRequest struct {
	Method          string
	ID              json.RawMessage
	SessionID       string
	ProtocolVersion string
	Accept          string
	Authorization   string
	Params          json.RawMessage
}

func newFakeMCPServer(t *testing.T) *fakeMCPServer {
	return &fakeMCPServer{
		t:          t,
		negotiate:  "2025-11-25",
		tools:      `[{"name":"lookup_order","description":"Look up an order by id.","inputSchema":{"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"]}}]`,
		callResult: `{"content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]}`,
	}
}

func (f *fakeMCPServer) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.requests))
	for _, r := range f.requests {
		out = append(out, r.Method)
	}
	return out
}

func (f *fakeMCPServer) recorded() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

func (f *fakeMCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		Method:          msg.Method,
		ID:              msg.ID,
		SessionID:       r.Header.Get("Mcp-Session-Id"),
		ProtocolVersion: r.Header.Get("MCP-Protocol-Version"),
		Accept:          r.Header.Get("Accept"),
		Authorization:   r.Header.Get("Authorization"),
		Params:          msg.Params,
	})
	f.mu.Unlock()

	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
		http.Error(w, "Not Acceptable: Client must accept both application/json and text/event-stream", http.StatusNotAcceptable)
		return
	}

	if msg.Method == "initialize" {
		f.handleInitialize(w, msg.ID, msg.Params)
		return
	}

	if !f.stateless {
		got := r.Header.Get("Mcp-Session-Id")
		if got == "" {
			http.Error(w, "Bad Request: Missing session ID", http.StatusBadRequest)
			return
		}
		// Expiry applies to method calls only: the handshake's own
		// notification is answered normally, so a one-shot expiry must not be
		// consumed by it.
		f.mu.Lock()
		expire := false
		if msg.Method != "notifications/initialized" {
			expire = f.expireAlways || f.expireSessionsOnce
			f.expireSessionsOnce = false
		}
		known := got == fmt.Sprintf("session-%d", f.sessions)
		f.mu.Unlock()
		if expire {
			http.Error(w, "Not Found: Session not found", http.StatusNotFound)
			return
		}
		if !known {
			http.Error(w, "Bad Request: unknown session", http.StatusBadRequest)
			return
		}
	}

	if pv := r.Header.Get("MCP-Protocol-Version"); pv != "" && pv != f.negotiate {
		http.Error(w, "Bad Request: Unsupported protocol version: "+pv, http.StatusBadRequest)
		return
	}

	if msg.Method == "notifications/initialized" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if f.statusOnMethod != 0 {
		http.Error(w, f.statusBody, f.statusOnMethod)
		return
	}

	switch msg.Method {
	case "tools/list":
		f.reply(w, msg.ID, `{"tools":`+f.tools+`}`)
	case "tools/call":
		f.reply(w, msg.ID, f.callResult)
	default:
		f.replyRaw(w, fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"Method not found"}}`, msg.ID))
	}
}

func (f *fakeMCPServer) handleInitialize(w http.ResponseWriter, id, params json.RawMessage) {
	var p struct {
		ProtocolVersion string          `json:"protocolVersion"`
		Capabilities    json.RawMessage `json:"capabilities"`
		ClientInfo      json.RawMessage `json:"clientInfo"`
	}
	_ = json.Unmarshal(params, &p)

	f.mu.Lock()
	f.sessions++
	sid := fmt.Sprintf("session-%d", f.sessions)
	f.mu.Unlock()

	if !f.stateless {
		w.Header().Set("Mcp-Session-Id", sid)
	}

	missing := p.ProtocolVersion == "" || len(p.Capabilities) == 0 || len(p.ClientInfo) == 0
	if missing || f.initializeErrorWithSession {
		f.replyRaw(w, fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32602,"message":"Invalid request parameters"}}`, id))
		return
	}

	f.reply(w, id, fmt.Sprintf(`{"protocolVersion":%q,"capabilities":{"tools":{}},"serverInfo":{"name":"fake","version":"1"}}`, f.negotiate))
}

func (f *fakeMCPServer) reply(w http.ResponseWriter, id json.RawMessage, result string) {
	f.replyRaw(w, fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, id, result))
}

func (f *fakeMCPServer) replyRaw(w http.ResponseWriter, message string) {
	if f.jsonResponse {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(message))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	var b strings.Builder
	if f.sseNoise {
		b.WriteString("event: message\r\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/message\",\"params\":{\"level\":\"info\"}}\r\n\r\n")
		b.WriteString("event: message\r\ndata: {\"jsonrpc\":\"2.0\",\"id\":999,\"result\":{\"tools\":[]}}\r\n\r\n")
	}
	b.WriteString("event: message\r\ndata: " + message + "\r\n\r\n")
	_, _ = w.Write([]byte(b.String()))
}
