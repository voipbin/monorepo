package homerhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHomer is a recording httptest server standing in for the Homer API.
type fakeHomer struct {
	srv  *httptest.Server
	hits atomic.Int32

	mu      sync.Mutex
	method  string
	path    string
	headers http.Header
	body    []byte
}

func newFakeHomer(t *testing.T, status int, respBody string) *fakeHomer {
	t.Helper()

	f := &fakeHomer{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		b, _ := io.ReadAll(r.Body)

		f.mu.Lock()
		f.method = r.Method
		f.path = r.URL.Path
		f.headers = r.Header.Clone()
		f.body = b
		f.mu.Unlock()

		w.WriteHeader(status)
		if respBody != "" {
			_, _ = w.Write([]byte(respBody))
		}
	}))
	t.Cleanup(f.srv.Close)

	return f
}

func (f *fakeHomer) lastRequest() (method, path string, headers http.Header, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.method, f.path, f.headers, f.body
}

// homerCall is one of the three Homer handler methods under a common signature.
type homerCall struct {
	name string
	call func(h HomerHandler, ctx context.Context, callID string, from, to time.Time) (any, error)
	// body returned by the server for a success case.
	okBody string
	path   string
	// expected request payload parts.
	searchKey string
	limit     float64
	extra     map[string]any
}

func homerCalls() []homerCall {
	return []homerCall{
		{
			name: "GetSIPMessages",
			call: func(h HomerHandler, ctx context.Context, callID string, from, to time.Time) (any, error) {
				return h.GetSIPMessages(ctx, callID, from, to)
			},
			okBody:    `{"data":{"messages":[]}}`,
			path:      "/api/v3/call/transaction",
			searchKey: "1_call",
			limit:     200,
			extra:     map[string]any{},
		},
		{
			name: "GetPcap",
			call: func(h HomerHandler, ctx context.Context, callID string, from, to time.Time) (any, error) {
				return h.GetPcap(ctx, callID, from, to)
			},
			okBody:    "pcap-bytes",
			path:      "/api/v3/export/call/messages/pcap",
			searchKey: "1_call",
			limit:     1000,
			extra:     map[string]any{"type": "string", "hepid": float64(1)},
		},
		{
			name: "GetRTCPPcap",
			call: func(h HomerHandler, ctx context.Context, callID string, from, to time.Time) (any, error) {
				return h.GetRTCPPcap(ctx, callID, from, to)
			},
			okBody:    "rtcp-pcap-bytes",
			path:      "/api/v3/export/call/messages/pcap",
			searchKey: "5_default",
			limit:     1000,
			extra:     map[string]any{"type": "string", "hepid": float64(5)},
		},
	}
}

var (
	testFrom = time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)
	testTo   = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
)

func Test_homer_input_validation(t *testing.T) {
	tests := []struct {
		name   string
		addr   string
		token  string
		callID string

		expectErr string
	}{
		{"missing address", "", "test-token", "call-1", "missing Homer API address or auth token"},
		{"missing token", "http://placeholder.invalid", "", "call-1", "missing Homer API address or auth token"},
		{"empty address and call id", "", "test-token", "", "missing Homer API address or auth token"},
		{"empty call id with config", "SERVER", "test-token", "", "sip call ID cannot be empty"},
	}

	for _, hc := range homerCalls() {
		for _, tt := range tests {
			t.Run(hc.name+"/"+tt.name, func(t *testing.T) {
				f := newFakeHomer(t, http.StatusOK, hc.okBody)
				addr := tt.addr
				if addr == "SERVER" {
					addr = f.srv.URL
				}
				h := NewHomerHandler(addr, tt.token)

				_, err := hc.call(h, context.Background(), tt.callID, testFrom, testTo)
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if f.hits.Load() != 0 {
					t.Errorf("Wrong match. expect no request to reach the server, got: %d", f.hits.Load())
				}
			})
		}
	}
}

func Test_homer_request(t *testing.T) {
	for _, hc := range homerCalls() {
		t.Run(hc.name, func(t *testing.T) {
			f := newFakeHomer(t, http.StatusOK, hc.okBody)
			h := NewHomerHandler(f.srv.URL, "test-token")

			if _, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if f.hits.Load() != 1 {
				t.Fatalf("Wrong match. expect: 1 hit, got: %d", f.hits.Load())
			}

			method, path, headers, body := f.lastRequest()
			if method != http.MethodPost {
				t.Errorf("Wrong match. expect: %s, got: %s", http.MethodPost, method)
			}
			if path != hc.path {
				t.Errorf("Wrong match. expect: %s, got: %s", hc.path, path)
			}
			if got := headers.Get("Auth-Token"); got != "test-token" {
				t.Errorf("Wrong match. expect: test-token, got: %s", got)
			}
			if got := headers.Get("Content-Type"); got != "application/json" {
				t.Errorf("Wrong match. expect: application/json, got: %s", got)
			}

			search := map[string]any{"callid": []any{"call-abc"}}
			for k, v := range hc.extra {
				search[k] = v
			}
			expect := map[string]any{
				// from is 10 minutes before testFrom, to is 10 minutes after testTo.
				"timestamp": map[string]any{
					"from": float64(1767322200000),
					"to":   float64(1767323645000),
				},
				"param": map[string]any{
					"limit":       hc.limit,
					"search":      map[string]any{hc.searchKey: search},
					"transaction": map[string]any{},
				},
			}

			var got map[string]any
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("Wrong match. could not decode the request body: %v", err)
			}
			if !reflect.DeepEqual(got, expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, got)
			}
		})
	}
}

func Test_homer_status(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expectOK bool
	}{
		{"299 success", 299, true},
		{"300 failure", 300, false},
		{"500 failure", 500, false},
	}

	for _, hc := range homerCalls() {
		for _, tt := range tests {
			t.Run(hc.name+"/"+tt.name, func(t *testing.T) {
				body := hc.okBody
				if !tt.expectOK && hc.name == "GetSIPMessages" {
					// a valid body, so only the status check can reject it.
					body = `{"data":{"messages":[]}}`
				}
				f := newFakeHomer(t, tt.status, body)
				h := NewHomerHandler(f.srv.URL, "test-token")

				_, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo)
				if tt.expectOK {
					if err != nil {
						t.Errorf("Wrong match. expect: ok, got: %v", err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), "non-success status") {
					t.Errorf("Wrong match. expect error containing: non-success status, got: %v", err)
				}
			})
		}
	}
}

func Test_homer_pcap_204(t *testing.T) {
	for _, hc := range homerCalls() {
		if hc.name == "GetSIPMessages" {
			continue
		}
		t.Run(hc.name, func(t *testing.T) {
			f := newFakeHomer(t, http.StatusNoContent, "")
			h := NewHomerHandler(f.srv.URL, "test-token")

			res, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if b, _ := res.([]byte); len(b) != 0 {
				t.Errorf("Wrong match. expect: empty body, got: %d bytes", len(b))
			}
		})
	}
}

func Test_GetSIPMessages_204_empty_body(t *testing.T) {
	f := newFakeHomer(t, http.StatusNoContent, "")
	h := NewHomerHandler(f.srv.URL, "test-token")

	res, err := h.GetSIPMessages(context.Background(), "call-abc", testFrom, testTo)
	if err == nil || !strings.Contains(err.Error(), "could not unmarshal Homer API response") {
		t.Errorf("Wrong match. expect: unmarshal error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_homer_transport_error(t *testing.T) {
	for _, hc := range homerCalls() {
		t.Run(hc.name, func(t *testing.T) {
			f := newFakeHomer(t, http.StatusOK, hc.okBody)
			addr := f.srv.URL
			f.srv.Close()
			h := NewHomerHandler(addr, "test-token")

			_, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo)
			if err == nil || !strings.Contains(err.Error(), "could not send HTTP request") {
				t.Errorf("Wrong match. expect error containing: could not send HTTP request, got: %v", err)
			}
		})
	}
}

func Test_homer_cancelled_context(t *testing.T) {
	for _, hc := range homerCalls() {
		t.Run(hc.name, func(t *testing.T) {
			f := newFakeHomer(t, http.StatusOK, hc.okBody)
			h := NewHomerHandler(f.srv.URL, "test-token")

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := hc.call(h, ctx, "call-abc", testFrom, testTo)
			if err == nil || !strings.Contains(err.Error(), "could not send HTTP request") {
				t.Errorf("Wrong match. expect error containing: could not send HTTP request, got: %v", err)
			}
			if f.hits.Load() != 0 {
				t.Errorf("Wrong match. expect no request to reach the server, got: %d", f.hits.Load())
			}
		})
	}
}

func Test_homer_invalid_address(t *testing.T) {
	for _, hc := range homerCalls() {
		t.Run(hc.name, func(t *testing.T) {
			h := NewHomerHandler("http://\x7f", "test-token")

			_, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo)
			if err == nil || !strings.Contains(err.Error(), "could not create HTTP request") {
				t.Errorf("Wrong match. expect error containing: could not create HTTP request, got: %v", err)
			}
		})
	}
}

func Test_GetSIPMessages_invalid_json(t *testing.T) {
	f := newFakeHomer(t, http.StatusOK, `{"data":`)
	h := NewHomerHandler(f.srv.URL, "test-token")

	res, err := h.GetSIPMessages(context.Background(), "call-abc", testFrom, testTo)
	if err == nil || !strings.Contains(err.Error(), "could not unmarshal Homer API response") {
		t.Errorf("Wrong match. expect: unmarshal error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_GetSIPMessages_empty_is_non_nil(t *testing.T) {
	f := newFakeHomer(t, http.StatusOK, `{"data":{"messages":[]}}`)
	h := NewHomerHandler(f.srv.URL, "test-token")

	res, err := h.GetSIPMessages(context.Background(), "call-abc", testFrom, testTo)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("Wrong match. expect: non-nil empty slice, got: %v", res)
	}
}

func Test_GetSIPMessages_mapping(t *testing.T) {
	body := `{"data":{"messages":[{"callid":"call-abc","raw":"INVITE sip:bob@192.0.2.10 SIP/2.0","micro_ts":1767322800123,"method":"INVITE","srcIp":"192.0.2.1","dstIp":"192.0.2.2","srcPort":5060,"dstPort":5061}]}}`
	f := newFakeHomer(t, http.StatusOK, body)
	h := NewHomerHandler(f.srv.URL, "test-token")

	res, err := h.GetSIPMessages(context.Background(), "call-abc", testFrom, testTo)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("Wrong match. expect: 1 message, got: %d", len(res))
	}

	m := res[0]
	if m.Timestamp != "2026-01-02T03:00:00.123Z" {
		t.Errorf("Wrong match. expect: 2026-01-02T03:00:00.123Z, got: %s", m.Timestamp)
	}
	if m.Method != "INVITE" || m.SrcIP != "192.0.2.1" || m.SrcPort != 5060 || m.DstIP != "192.0.2.2" || m.DstPort != 5061 {
		t.Errorf("Wrong match. unexpected fields: %+v", m)
	}
	if m.Raw != "INVITE sip:bob@192.0.2.10 SIP/2.0" {
		t.Errorf("Wrong match. unexpected raw: %s", m.Raw)
	}
}

// buildMessagesBody builds a Homer response with n messages in descending time
// order. Every message carries a non-zero millisecond part and the same number
// of fractional digits, so the string timestamps order like the real times.
func buildMessagesBody(n int) string {
	parts := make([]string, 0, n)
	for i := n - 1; i >= 0; i-- {
		parts = append(parts, fmt.Sprintf(`{"micro_ts":%d,"method":"M%d"}`, int64(1767322800123)+int64(i)*1000, i))
	}
	return `{"data":{"messages":[` + strings.Join(parts, ",") + `]}}`
}

func Test_GetSIPMessages_sort_and_cap(t *testing.T) {
	tests := []struct {
		name  string
		count int

		expectCount int
	}{
		{"3 messages are sorted ascending", 3, 3},
		{"exactly 50 are all returned", 50, 50},
		{"51 are cut to the earliest 50", 51, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeHomer(t, http.StatusOK, buildMessagesBody(tt.count))
			h := NewHomerHandler(f.srv.URL, "test-token")

			res, err := h.GetSIPMessages(context.Background(), "call-abc", testFrom, testTo)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != tt.expectCount {
				t.Fatalf("Wrong match. expect: %d messages, got: %d", tt.expectCount, len(res))
			}

			// the input is in descending order; the output must start with M0 and ascend.
			for i, m := range res {
				if expect := fmt.Sprintf("M%d", i); m.Method != expect {
					t.Errorf("Wrong match at %d. expect: %s, got: %s", i, expect, m.Method)
				}
			}
		})
	}
}

func Test_GetPcap_returns_body_as_is(t *testing.T) {
	f := newFakeHomer(t, http.StatusOK, "\x00\x01pcap\xff")
	h := NewHomerHandler(f.srv.URL, "test-token")

	res, err := h.GetPcap(context.Background(), "call-abc", testFrom, testTo)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if string(res) != "\x00\x01pcap\xff" {
		t.Errorf("Wrong match. expect: the raw bytes, got: %q", res)
	}

	res, err = h.GetRTCPPcap(context.Background(), "call-abc", testFrom, testTo)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if string(res) != "\x00\x01pcap\xff" {
		t.Errorf("Wrong match. expect: the raw bytes, got: %q", res)
	}
}
