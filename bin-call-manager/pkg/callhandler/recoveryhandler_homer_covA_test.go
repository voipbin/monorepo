package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

const covASIPInvite = "INVITE sip:bob@example.com SIP/2.0\r\n" +
	"Via: SIP/2.0/UDP 192.0.2.1:5060;branch=z9hG4bK1\r\n" +
	"From: <sip:alice@example.com>;tag=a1\r\n" +
	"To: <sip:bob@example.com>\r\n" +
	"Call-ID: covA-call-id\r\n" +
	"CSeq: 1 INVITE\r\n" +
	"Content-Length: 0\r\n\r\n"

func Test_getSIPMessages_errors_covA(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		token   string
		callID  string
		handler http.HandlerFunc
	}{
		{name: "missing address", addr: "", token: "t", callID: "c"},
		{name: "missing token", addr: "http://x", token: "", callID: "c"},
		{name: "empty call id", addr: "http://x", token: "t", callID: ""},
		{
			name: "status error", token: "t", callID: "c",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		},
		{
			name: "invalid json", token: "t", callID: "c",
			handler: func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("not json")) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := tt.addr
			client := http.DefaultClient
			if tt.handler != nil {
				srv := httptest.NewServer(tt.handler)
				defer srv.Close()
				addr = srv.URL
				client = srv.Client()
			}
			h := &recoveryHandler{httpClient: client, homerAPIAddress: addr, homerAuthToken: tt.token}

			res, err := h.getSIPMessages(context.Background(), tt.callID)
			if err == nil {
				t.Errorf("expected error, got res: %v", res)
			}
		})
	}
}

func Test_getSIPMessages_requestError_covA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // connection refused

	h := &recoveryHandler{httpClient: http.DefaultClient, homerAPIAddress: addr, homerAuthToken: "t"}
	if _, err := h.getSIPMessages(context.Background(), "c"); err == nil {
		t.Errorf("expected error")
	}

	// invalid URL -> request creation error
	h = &recoveryHandler{httpClient: http.DefaultClient, homerAPIAddress: "http://[::1", homerAuthToken: "t"}
	if _, err := h.getSIPMessages(context.Background(), "c"); err == nil {
		t.Errorf("expected error on invalid url")
	}
}

func Test_getSIPMessages_success_covA(t *testing.T) {
	var gotToken, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("Auth-Token")
		gotPath = r.URL.Path
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])

		raw := strings.ReplaceAll(covASIPInvite, "\r\n", "\\r\\n")
		body := `{"data":{"messages":[` +
			`{"callid":"covA-call-id","raw":"` + raw + `","id":1},` +
			`{"callid":"other","raw":"` + raw + `","id":2},` +
			`{"callid":"covA-call-id","raw":"garbage","id":3}]}}`
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	h := &recoveryHandler{
		httpClient:      srv.Client(),
		homerAPIAddress: srv.URL,
		homerAuthToken:  "token-covA",
		loadBalancerIPs: []string{"203.0.113.1"},
	}

	res, err := h.getSIPMessages(context.Background(), "covA-call-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].CallID != "covA-call-id" {
		t.Errorf("unexpected result: %v", res)
	}
	if gotToken != "token-covA" {
		t.Errorf("wrong token: %s", gotToken)
	}
	if gotPath != "/api/v3/call/transaction" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if !strings.Contains(gotBody, "covA-call-id") || !strings.Contains(gotBody, "203.0.113.1") {
		t.Errorf("wrong body: %s", gotBody)
	}
}

func covASIPInviteWith(callID string, cseq int) string {
	return "INVITE sip:bob@example.com SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP 192.0.2.1:5060;branch=z9hG4bK1\r\n" +
		"From: <sip:alice@example.com>;tag=a1\r\n" +
		"To: <sip:bob@example.com>\r\n" +
		"Call-ID: " + callID + "\r\n" +
		fmt.Sprintf("CSeq: %d INVITE\r\n", cseq) +
		"Content-Length: 0\r\n\r\n"
}

func Test_getSIPMessages_status300_covA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultipleChoices)
		_, _ = w.Write([]byte(`{"data":{"messages":[]}}`))
	}))
	defer srv.Close()

	h := &recoveryHandler{httpClient: srv.Client(), homerAPIAddress: srv.URL, homerAuthToken: "t"}
	if res, err := h.getSIPMessages(context.Background(), "c"); err == nil {
		t.Errorf("Wrong match. expect: error, got: %v", res)
	}
}

func Test_getSIPMessages_emptyCallID_covA(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"data":{"messages":[]}}`))
	}))
	defer srv.Close()

	h := &recoveryHandler{httpClient: srv.Client(), homerAPIAddress: srv.URL, homerAuthToken: "t"}
	_, err := h.getSIPMessages(context.Background(), "")
	if err == nil || err.Error() != "call ID cannot be empty" {
		t.Errorf("Wrong match. expect: call ID cannot be empty, got: %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Errorf("Wrong match. expect: no request, got: %d", n)
	}
}

func Test_getSIPMessages_requestPayload_covA(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"data":{"messages":[]}}`))
	}))
	defer srv.Close()

	h := &recoveryHandler{httpClient: srv.Client(), homerAPIAddress: srv.URL, homerAuthToken: "t", loadBalancerIPs: []string{"203.0.113.9"}}
	if _, err := h.getSIPMessages(context.Background(), "covA-call-id"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got homerRequestPayload
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("could not parse the request body. body: %s, err: %v", gotBody, err)
	}
	if got.Param.Limit != 1 {
		t.Errorf("Wrong match. expect limit: 1, got: %d", got.Param.Limit)
	}
	if !reflect.DeepEqual(got.Param.Whitelist, []string{"203.0.113.9"}) {
		t.Errorf("Wrong match. whitelist: %v", got.Param.Whitelist)
	}
	if got.Timestamp.From >= got.Timestamp.To {
		t.Errorf("Wrong match. expect: from < to, got: %d, %d", got.Timestamp.From, got.Timestamp.To)
	}
}

func Test_filterSIPMessages_rowCallID_covA(t *testing.T) {
	// the second row belongs to another Call-ID even if its raw message carries the requested one.
	rows := []HomerSIPMessageDetail{
		{ID: 1, CallID: "call-a", Raw: covASIPInviteWith("call-a", 1)},
		{ID: 2, CallID: "call-b", Raw: covASIPInviteWith("call-a", 2)},
	}

	res := filterSIPMessages(rows, "call-a")
	if len(res) != 1 || res[0].CSeq != 1 {
		t.Errorf("Wrong match. expect: only the first row, got: %v", res)
	}
}
