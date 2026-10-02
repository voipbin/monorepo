package callhandler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jart/gosip/sip"
)

// Synthetic SIP captures shaped like a production call through Kamailio (double Record-Route) and the call Asterisk.
// No customer data. Line endings are added by sipMsg.

const (
	tCallID = "a84b4c76e66710@pc33.example.com"

	tRR1 = "<sip:199.127.61.42;r2=on;lr>"
	tRR2 = "<sip:172.24.0.246;transport=tcp;r2=on;lr>"
)

func sipMsg(t *testing.T, lines ...string) *sip.Msg {
	t.Helper()
	raw := strings.Join(lines, "\r\n") + "\r\n\r\n"
	m, err := sip.ParseMsg([]byte(raw))
	if err != nil {
		t.Fatalf("could not parse the test message. err: %v\n%s", err, raw)
	}
	return m
}

// outgoing call: Asterisk (tag as-1) calls the remote; the provider challenges the first INVITE.
func outgoingCapture(t *testing.T) map[string]*sip.Msg {
	from := `"Agent" <sip:+15550100@voipbin.net>;tag=as-1`
	to := `<sip:+15550199@carrier.example.com>`
	toTagged := to + ";tag=rem-1"
	via := "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-1"
	return map[string]*sip.Msg{
		"invite1": sipMsg(t, "INVITE sip:+15550199@carrier.example.com SIP/2.0", via, "From: "+from, "To: "+to,
			"Call-ID: "+tCallID, "CSeq: 101 INVITE", "Contact: <sip:as@172.24.0.244:5060>", "Content-Length: 0"),
		"407": sipMsg(t, "SIP/2.0 407 Proxy Authentication Required", via, "From: "+from, "To: "+to+";tag=chal-1",
			"Call-ID: "+tCallID, "CSeq: 101 INVITE", "Content-Length: 0"),
		"invite2": sipMsg(t, "INVITE sip:+15550199@carrier.example.com SIP/2.0", via, "From: "+from, "To: "+to,
			"Call-ID: "+tCallID, "CSeq: 102 INVITE", "Contact: <sip:as@172.24.0.244:5060>", "Content-Length: 0"),
		"200": sipMsg(t, "SIP/2.0 200 OK", via, "Record-Route: "+tRR1, "Record-Route: "+tRR2, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 102 INVITE", "Contact: <sip:remote@198.51.100.20:5070;transport=udp>", "Content-Length: 0"),
		"ack": sipMsg(t, "ACK sip:remote@198.51.100.20:5070;transport=udp SIP/2.0", via, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 102 ACK", "Content-Length: 0"),
		// Asterisk session refresh (re-INVITE with To tag) and its 2xx
		"reinvite": sipMsg(t, "INVITE sip:remote@198.51.100.20:5070;transport=udp SIP/2.0", via, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 103 INVITE", "Content-Length: 0"),
		"reinvite200": sipMsg(t, "SIP/2.0 200 OK", via, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 103 INVITE", "Contact: <sip:remote@198.51.100.20:5070;transport=udp>", "Content-Length: 0"),
		// a request from the remote: its own CSeq space, From carries the remote tag
		"remoteinfo": sipMsg(t, "INFO sip:as@172.24.0.244:5060 SIP/2.0", via, "From: "+toTagged, "To: "+from,
			"Call-ID: "+tCallID, "CSeq: 9001 INFO", "Content-Length: 0"),
	}
}

// incoming call: the remote (tag caller-1) calls Asterisk, which answers with tag as-9.
func incomingCapture(t *testing.T) map[string]*sip.Msg {
	from := `"Caller" <sip:alice@example.com>;tag=caller-1`
	to := `<sip:2000@abcd.reg.voipbin.net>`
	toTagged := to + ";tag=as-9"
	via := "Via: SIP/2.0/TCP 172.24.0.246;branch=z9hG4bK-9"
	return map[string]*sip.Msg{
		"invite": sipMsg(t, "INVITE sip:2000@172.24.0.101:5060 SIP/2.0", via, "Record-Route: "+tRR2, "Record-Route: "+tRR1,
			"From: "+from, "To: "+to, "Call-ID: "+tCallID, "CSeq: 7 INVITE", "Contact: <sip:alice@203.0.113.7:5060>", "Content-Length: 0"),
		"200": sipMsg(t, "SIP/2.0 200 OK", via, "Record-Route: "+tRR2, "Record-Route: "+tRR1, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 7 INVITE", "Contact: <sip:as@172.24.0.101:5060>", "Content-Length: 0"),
		"ack": sipMsg(t, "ACK sip:as@172.24.0.101:5060 SIP/2.0", via, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 7 ACK", "Content-Length: 0"),
		"remotebye200": sipMsg(t, "SIP/2.0 200 OK", via, "From: "+from, "To: "+toTagged,
			"Call-ID: "+tCallID, "CSeq: 8 INFO", "Content-Length: 0"),
	}
}

func pick(m map[string]*sip.Msg, names ...string) []*sip.Msg {
	res := []*sip.Msg{}
	for _, n := range names {
		res = append(res, m[n])
	}
	return res
}

// permutations returns every order of the given messages.
func permutations(msgs []*sip.Msg) [][]*sip.Msg {
	if len(msgs) <= 1 {
		return [][]*sip.Msg{append([]*sip.Msg{}, msgs...)}
	}
	res := [][]*sip.Msg{}
	for i := range msgs {
		rest := append(append([]*sip.Msg{}, msgs[:i]...), msgs[i+1:]...)
		for _, p := range permutations(rest) {
			res = append(res, append([]*sip.Msg{msgs[i]}, p...))
		}
	}
	return res
}

func Test_getRecoveryDetail(t *testing.T) {
	out := outgoingCapture(t)
	in := incomingCapture(t)

	outgoingExpect := recoveryDetail{
		CallID:       tCallID,
		FromDisplay:  "Agent",
		FromURI:      "sip:+15550100@voipbin.net",
		FromTag:      "as-1",
		ToURI:        "sip:+15550199@carrier.example.com",
		ToTag:        "rem-1",
		RequestURI:   "sip:remote@198.51.100.20:5070;transport=udp",
		Routes:       tRR2 + ", " + tRR1,
		RecordRoutes: tRR1 + ", " + tRR2,
		CSeq:         103 + recoveryCSeqMargin,
	}
	outgoingExpectLater := outgoingExpect
	outgoingExpectLater.CSeq = 110 + recoveryCSeqMargin

	incomingExpect := recoveryDetail{
		CallID:       tCallID,
		FromURI:      "sip:2000@abcd.reg.voipbin.net",
		FromTag:      "as-9",
		ToDisplay:    "Caller",
		ToURI:        "sip:alice@example.com",
		ToTag:        "caller-1",
		RequestURI:   "sip:alice@203.0.113.7:5060",
		Routes:       tRR2 + ", " + tRR1,
		RecordRoutes: tRR2 + ", " + tRR1,
		CSeq:         0,
	}

	tests := []struct {
		name     string
		messages []*sip.Msg
		role     asteriskRole
		expect   recoveryDetail
	}{
		{
			name:     "outgoing, challenged, with re-INVITE and a remote request",
			messages: pick(out, "invite1", "407", "invite2", "200", "ack", "reinvite", "reinvite200", "remoteinfo"),
			role:     asteriskRoleUAC,
			expect:   outgoingExpect,
		},
		{
			name:     "outgoing, duplicated hop copies",
			messages: pick(out, "200", "invite2", "200", "invite1", "invite2", "407", "reinvite200", "reinvite", "ack"),
			role:     asteriskRoleUAC,
			expect:   outgoingExpect,
		},
		{
			name:     "outgoing, a later To-tagless INVITE also answered: the lowest CSeq created the dialog",
			messages: append(pick(out, "invite2", "200", "reinvite"), laterTaglessInvite(t), laterTagless200(t)),
			role:     asteriskRoleUAC,
			expect:   outgoingExpectLater,
		},
		{
			name:     "outgoing, single Record-Route",
			messages: []*sip.Msg{out["invite2"], singleRR200(t)},
			role:     asteriskRoleUAC,
			expect: func() recoveryDetail {
				e := outgoingExpect
				e.Routes, e.RecordRoutes, e.CSeq = tRR1, tRR1, 102+recoveryCSeqMargin
				return e
			}(),
		},
		{
			name:     "outgoing, no Record-Route",
			messages: []*sip.Msg{out["invite2"], noRR200(t)},
			role:     asteriskRoleUAC,
			expect: func() recoveryDetail {
				e := outgoingExpect
				e.Routes, e.RecordRoutes, e.CSeq = "", "", 102+recoveryCSeqMargin
				return e
			}(),
		},
		{
			name:     "incoming, single Record-Route",
			messages: []*sip.Msg{singleRRInvite(t), in["200"]},
			role:     asteriskRoleUAS,
			expect: func() recoveryDetail {
				e := incomingExpect
				e.Routes, e.RecordRoutes = tRR2, tRR2
				return e
			}(),
		},
		{
			name:     "incoming, ACK before 200, no request from Asterisk",
			messages: pick(in, "ack", "200", "invite", "remotebye200"),
			role:     asteriskRoleUAS,
			expect:   incomingExpect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getRecoveryDetail(tt.messages, tt.role)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if *res != tt.expect {
				t.Errorf("Wrong match.\nexpect: %+v\ngot:    %+v", tt.expect, *res)
			}
		})
	}
}

// Test_getRecoveryDetail_order checks the result does not depend on the order Homer returns the rows in.
func Test_getRecoveryDetail_order(t *testing.T) {
	out := outgoingCapture(t)
	in := incomingCapture(t)

	tests := []struct {
		name     string
		messages []*sip.Msg
		role     asteriskRole
	}{
		{
			name:     "outgoing",
			messages: pick(out, "invite1", "407", "invite2", "200", "reinvite", "remoteinfo"),
			role:     asteriskRoleUAC,
		},
		{
			name:     "incoming",
			messages: pick(in, "invite", "200", "ack", "remotebye200"),
			role:     asteriskRoleUAS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect, err := getRecoveryDetail(tt.messages, tt.role)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			for i, p := range permutations(tt.messages) {
				res, err := getRecoveryDetail(p, tt.role)
				if err != nil {
					t.Fatalf("Wrong match. permutation: %d, expect: ok, got: %v", i, err)
				}
				if *res != *expect {
					t.Fatalf("Wrong match. permutation: %d\nexpect: %+v\ngot:    %+v", i, *expect, *res)
				}
			}
		})
	}
}

func Test_getRecoveryDetail_error(t *testing.T) {
	out := outgoingCapture(t)
	in := incomingCapture(t)

	via := "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-1"
	from := `"Agent" <sip:+15550100@voipbin.net>;tag=as-1`
	forked := sipMsg(t, "SIP/2.0 200 OK", via, "Record-Route: "+tRR1, "Record-Route: "+tRR2, "From: "+from,
		"To: <sip:+15550199@carrier.example.com>;tag=rem-2", "Call-ID: "+tCallID, "CSeq: 102 INVITE",
		"Contact: <sip:remote2@198.51.100.21:5070>", "Content-Length: 0")
	otherCopy := sipMsg(t, "SIP/2.0 200 OK", via, "Record-Route: "+tRR1, "From: "+from,
		"To: <sip:+15550199@carrier.example.com>;tag=rem-1", "Call-ID: "+tCallID, "CSeq: 102 INVITE",
		"Contact: <sip:remote@198.51.100.20:5070;transport=udp>", "Content-Length: 0")
	noFromTag := sipMsg(t, "INVITE sip:2000@172.24.0.101:5060 SIP/2.0", "Via: SIP/2.0/TCP 172.24.0.246;branch=z9hG4bK-9",
		"From: <sip:alice@example.com>", "To: <sip:2000@abcd.reg.voipbin.net>", "Call-ID: "+tCallID, "CSeq: 7 INVITE",
		"Contact: <sip:alice@203.0.113.7:5060>", "Content-Length: 0")
	noFromTag200 := sipMsg(t, "SIP/2.0 200 OK", "Via: SIP/2.0/TCP 172.24.0.246;branch=z9hG4bK-9",
		"From: <sip:alice@example.com>", "To: <sip:2000@abcd.reg.voipbin.net>;tag=as-9", "Call-ID: "+tCallID, "CSeq: 7 INVITE",
		"Contact: <sip:as@172.24.0.101:5060>", "Content-Length: 0")

	inviteOtherCopy := sipMsg(t, "INVITE sip:2000@172.24.0.101:5060 SIP/2.0", "Via: SIP/2.0/TCP 172.24.0.246;branch=z9hG4bK-9",
		"Record-Route: "+tRR1, `From: "Caller" <sip:alice@example.com>;tag=caller-1`, "To: <sip:2000@abcd.reg.voipbin.net>",
		"Call-ID: "+tCallID, "CSeq: 7 INVITE", "Contact: <sip:alice@203.0.113.7:5060>", "Content-Length: 0")

	tests := []struct {
		name     string
		messages []*sip.Msg
		role     asteriskRole
	}{
		{"no messages", []*sip.Msg{}, asteriskRoleUAC},
		{"differing copies of the initial INVITE", append(pick(in, "invite", "200"), inviteOtherCopy), asteriskRoleUAS},
		{"unknown role", pick(out, "invite2", "200"), asteriskRoleUnknown},
		{"no 2xx", pick(out, "invite1", "407"), asteriskRoleUAC},
		{"only a re-INVITE 2xx", pick(out, "invite1", "407", "invite2", "ack", "reinvite", "reinvite200"), asteriskRoleUAC},
		{"no initial INVITE", pick(out, "200", "ack", "reinvite"), asteriskRoleUAC},
		{"forked 2xx", append(pick(out, "invite2", "200"), forked), asteriskRoleUAC},
		{"differing copies of the 2xx", append(pick(out, "invite2", "200"), otherCopy), asteriskRoleUAC},
		{"missing remote tag", []*sip.Msg{noFromTag, noFromTag200}, asteriskRoleUAS},
		{"incoming INVITE without its 2xx", pick(in, "invite"), asteriskRoleUAS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getRecoveryDetail(tt.messages, tt.role)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: %+v", res)
			}
		})
	}
}

func Test_filterSIPMessages(t *testing.T) {
	out := outgoingCapture(t)

	rows := []HomerSIPMessageDetail{
		{ID: 1, CallID: tCallID, Raw: out["invite2"].String()},
		{ID: 2, CallID: tCallID, Raw: "not a sip message"},
		{ID: 3, CallID: "other-call-id", Raw: out["200"].String()},
		{ID: 4, CallID: tCallID, Raw: strings.Replace(out["ack"].String(), tCallID, "mismatch@x", 1)},
		{ID: 5, CallID: tCallID, Raw: out["200"].String()},
	}

	res := filterSIPMessages(rows, tCallID)
	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2 messages, got: %d", len(res))
	}
	got := fmt.Sprintf("%s/%d", res[0].Method, res[1].Status)
	if got != "INVITE/200" {
		t.Errorf("Wrong match. expect: INVITE/200, got: %s", got)
	}
}

// a second To-tagless INVITE from Asterisk in the same Call-ID (CSeq 110), answered with another To tag.
// It cannot have created the dialog the call runs on; the lowest-CSeq dialog is used.
func laterTaglessInvite(t *testing.T) *sip.Msg {
	return sipMsg(t, "INVITE sip:+15550199@carrier.example.com SIP/2.0", "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-7",
		`From: "Agent" <sip:+15550100@voipbin.net>;tag=as-1`, "To: <sip:+15550199@carrier.example.com>",
		"Call-ID: "+tCallID, "CSeq: 110 INVITE", "Contact: <sip:as@172.24.0.244:5060>", "Content-Length: 0")
}

func laterTagless200(t *testing.T) *sip.Msg {
	return sipMsg(t, "SIP/2.0 200 OK", "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-7",
		`From: "Agent" <sip:+15550100@voipbin.net>;tag=as-1`, "To: <sip:+15550199@carrier.example.com>;tag=rem-9",
		"Call-ID: "+tCallID, "CSeq: 110 INVITE", "Contact: <sip:other@198.51.100.99:5070>", "Content-Length: 0")
}

func singleRR200(t *testing.T) *sip.Msg {
	return sipMsg(t, "SIP/2.0 200 OK", "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-1", "Record-Route: "+tRR1,
		`From: "Agent" <sip:+15550100@voipbin.net>;tag=as-1`, "To: <sip:+15550199@carrier.example.com>;tag=rem-1",
		"Call-ID: "+tCallID, "CSeq: 102 INVITE", "Contact: <sip:remote@198.51.100.20:5070;transport=udp>", "Content-Length: 0")
}

func noRR200(t *testing.T) *sip.Msg {
	return sipMsg(t, "SIP/2.0 200 OK", "Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-1",
		`From: "Agent" <sip:+15550100@voipbin.net>;tag=as-1`, "To: <sip:+15550199@carrier.example.com>;tag=rem-1",
		"Call-ID: "+tCallID, "CSeq: 102 INVITE", "Contact: <sip:remote@198.51.100.20:5070;transport=udp>", "Content-Length: 0")
}

func singleRRInvite(t *testing.T) *sip.Msg {
	return sipMsg(t, "INVITE sip:2000@172.24.0.101:5060 SIP/2.0", "Via: SIP/2.0/TCP 172.24.0.246;branch=z9hG4bK-9", "Record-Route: "+tRR2,
		`From: "Caller" <sip:alice@example.com>;tag=caller-1`, "To: <sip:2000@abcd.reg.voipbin.net>",
		"Call-ID: "+tCallID, "CSeq: 7 INVITE", "Contact: <sip:alice@203.0.113.7:5060>", "Content-Length: 0")
}

// Test_filterSIPMessages_body checks a message whose SDP body the SIP parser rejects (a declined dynamic payload
// without rtpmap) is still used: only the headers are parsed.
func Test_filterSIPMessages_body(t *testing.T) {
	body := "v=0\r\no=- 1 1 IN IP4 198.51.100.20\r\ns=-\r\nc=IN IP4 198.51.100.20\r\nt=0 0\r\n" +
		"m=audio 4000 RTP/AVP 0\r\na=rtpmap:0 PCMU/8000\r\nm=video 0 RTP/AVP 96\r\n"
	raw := strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 172.24.0.244:5060;branch=z9hG4bK-1",
		`From: "Agent" <sip:+15550100@voipbin.net>;tag=as-1`,
		"To: <sip:+15550199@carrier.example.com>;tag=rem-1",
		"Call-ID: " + tCallID,
		"CSeq: 102 INVITE",
		"Contact: <sip:remote@198.51.100.20:5070;transport=udp>",
		"Content-Type: application/sdp",
		fmt.Sprintf("Content-Length: %d", len(body)),
	}, "\r\n") + "\r\n\r\n" + body

	if _, err := sip.ParseMsg([]byte(raw)); err == nil {
		t.Fatalf("the test body is expected to be rejected by the full parser")
	}

	res := filterSIPMessages([]HomerSIPMessageDetail{{ID: 1, CallID: tCallID, Raw: raw}}, tCallID)
	if len(res) != 1 {
		t.Fatalf("Wrong match. expect: 1 message, got: %d", len(res))
	}
	if res[0].Status != 200 || getTag(res[0].To) != "rem-1" {
		t.Errorf("Wrong match. expect: 200 with tag rem-1, got: %d %s", res[0].Status, getTag(res[0].To))
	}
}
