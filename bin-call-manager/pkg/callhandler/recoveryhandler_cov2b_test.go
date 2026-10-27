package callhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jart/gosip/sip"
)

func Test_NewRecoveryHandler_cov2b(t *testing.T) {
	res := NewRecoveryHandler(nil, "http://homer.example.invalid", "token-cov2b", []string{"203.0.113.1"})

	h, ok := res.(*recoveryHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.homerAPIAddress != "http://homer.example.invalid" || h.homerAuthToken != "token-cov2b" {
		t.Errorf("Wrong homer config. address: %s, token: %s", h.homerAPIAddress, h.homerAuthToken)
	}
	if len(h.loadBalancerIPs) != 1 || h.loadBalancerIPs[0] != "203.0.113.1" {
		t.Errorf("Wrong load balancer IPs. got: %v", h.loadBalancerIPs)
	}
	if h.httpClient == nil || h.httpClient.Timeout.Seconds() != 30 {
		t.Errorf("Wrong http client. got: %+v", h.httpClient)
	}
}

func Test_GetRecoveryDetail_cov2b_success(t *testing.T) {
	out := outgoingCapture(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rows := []string{}
		for _, name := range []string{"invite2", "200"} {
			raw := strings.ReplaceAll(out[name].String(), "\r\n", "\\r\\n")
			rows = append(rows, `{"callid":"`+tCallID+`","raw":"`+raw+`","id":1}`)
		}
		_, _ = w.Write([]byte(`{"data":{"messages":[` + strings.Join(rows, ",") + `]}}`))
	}))
	defer srv.Close()

	h := &recoveryHandler{
		httpClient:      srv.Client(),
		homerAPIAddress: srv.URL,
		homerAuthToken:  "token-cov2b",
	}

	res, err := h.GetRecoveryDetail(context.Background(), tCallID, asteriskRoleUAC)
	if err != nil {
		t.Fatalf("Wrong error. got: %v", err)
	}
	if res.CallID != tCallID || res.FromTag != "as-1" || res.ToTag != "rem-1" {
		t.Errorf("Wrong detail. got: %+v", res)
	}
	if !strings.Contains(res.RequestURI, "198.51.100.20") {
		t.Errorf("Wrong request URI. got: %s", res.RequestURI)
	}
}

func Test_getRecoveryDetail_cov2b_fallbacks(t *testing.T) {
	via := "Via: SIP/2.0/UDP 198.51.100.10:5060;branch=z9hG4bK-1"
	from := `"Agent" <sip:+15551230001@voipbin.example.com>;tag=as-1`
	to := "<sip:+15551230002@carrier.example.com>"

	tests := []struct {
		name     string
		messages []*sip.Msg
		role     asteriskRole

		expectRequestURI string
	}{
		{
			name: "UAC without a Contact in the 2xx uses the INVITE To URI",
			messages: []*sip.Msg{
				sipMsg(t, "INVITE sip:+15551230002@carrier.example.com SIP/2.0", via, "From: "+from, "To: "+to, "Call-ID: "+tCallID, "CSeq: 1 INVITE", "Content-Length: 0"),
				sipMsg(t, "SIP/2.0 200 OK", via, "From: "+from, "To: "+to+";tag=rem-1", "Call-ID: "+tCallID, "CSeq: 1 INVITE", "Content-Length: 0"),
			},
			role:             asteriskRoleUAC,
			expectRequestURI: "sip:+15551230002@carrier.example.com",
		},
		{
			name: "UAS without a Contact in the INVITE uses the INVITE From URI",
			messages: []*sip.Msg{
				sipMsg(t, "INVITE sip:2000@198.51.100.11:5060 SIP/2.0", via, `From: "Caller" <sip:alice@example.com>;tag=caller-1`, "To: <sip:2000@abcd.reg.voipbin.example.com>", "Call-ID: "+tCallID, "CSeq: 7 INVITE", "Content-Length: 0"),
				sipMsg(t, "SIP/2.0 200 OK", via, `From: "Caller" <sip:alice@example.com>;tag=caller-1`, "To: <sip:2000@abcd.reg.voipbin.example.com>;tag=as-9", "Call-ID: "+tCallID, "CSeq: 7 INVITE", "Content-Length: 0"),
			},
			role:             asteriskRoleUAS,
			expectRequestURI: "sip:alice@example.com",
		},
		{
			name: "the dialog with the lowest CSeq wins even when it appears later",
			messages: append(
				[]*sip.Msg{laterTaglessInvite(t), laterTagless200(t)},
				pick(outgoingCapture(t), "invite2", "200")...,
			),
			role:             asteriskRoleUAC,
			expectRequestURI: "sip:remote@198.51.100.20:5070;transport=udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getRecoveryDetail(tt.messages, tt.role)
			if err != nil {
				t.Fatalf("Wrong error. got: %v", err)
			}
			if res.RequestURI != tt.expectRequestURI {
				t.Errorf("Wrong request URI. expect: %s, got: %s", tt.expectRequestURI, res.RequestURI)
			}
		})
	}
}

func Test_findDialogCreation_cov2b_sameCSeqDifferentDialogs(t *testing.T) {
	via := "Via: SIP/2.0/UDP 198.51.100.10:5060;branch=z9hG4bK-1"
	mk := func(fromTag string) []*sip.Msg {
		from := "<sip:a@example.com>;tag=" + fromTag
		to := "<sip:b@example.com>"
		return []*sip.Msg{
			sipMsg(t, "INVITE sip:b@example.com SIP/2.0", via, "From: "+from, "To: "+to, "Call-ID: "+tCallID, "CSeq: 5 INVITE", "Content-Length: 0"),
			sipMsg(t, "SIP/2.0 200 OK", via, "From: "+from, "To: "+to+";tag=r", "Call-ID: "+tCallID, "CSeq: 5 INVITE", "Content-Length: 0"),
		}
	}

	_, _, err := findDialogCreation(append(mk("t1"), mk("t2")...))
	if err == nil || !strings.Contains(err.Error(), "same CSeq") {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_sipAddrHelpers_cov2b_nilAndTagless(t *testing.T) {
	tagless := sipMsg(t, "INVITE sip:b@example.com SIP/2.0", "Via: SIP/2.0/UDP 192.0.2.1:5060;branch=z9hG4bK-1",
		"From: <sip:a@example.com>", "To: <sip:b@example.com>", "Call-ID: "+tCallID, "CSeq: 1 INVITE", "Content-Length: 0")

	if got := getTag(nil); got != "" {
		t.Errorf("getTag(nil) = %q", got)
	}
	if got := getTag(tagless.From); got != "" {
		t.Errorf("getTag(tagless) = %q", got)
	}
	if got := addrURI(nil); got != "" {
		t.Errorf("addrURI(nil) = %q", got)
	}
	if got := addrURI(tagless.From); got != "sip:a@example.com" {
		t.Errorf("addrURI = %q", got)
	}
	if got := addrDisplay(nil); got != "" {
		t.Errorf("addrDisplay(nil) = %q", got)
	}
	if got := addrString(nil); got != "" {
		t.Errorf("addrString(nil) = %q", got)
	}
	if got := addrString(tagless.From); got != "<sip:a@example.com>" {
		t.Errorf("addrString = %q", got)
	}
}
