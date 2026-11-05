package carriertrackhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/pkg/mcpschema"
)

const (
	fxID     = "SENTINEL-FEDEX-ID"
	fxSecret = "SENTINEL-FEDEX-SECRET"
	fxKey    = fxID + ":" + fxSecret
	fxNumber = "ABCD12345678"
	fxToken  = "STUB-ACCESS-TOKEN"

	fxFAD      = "TRACKING.AUTHENTICATEDDELIVERY.ERROR"
	fxNotFound = "TRACKING.DATA.NOTFOUND"
	fxEmpty    = "TRACKING.TRACKINGNUMBER.EMPTY"
)

// fxPickedUp is a plain tracking result of the stub carrier.
const fxPickedUp = `{"trackingNumberInfo":{"trackingNumber":"ABCD12345678"},"latestStatusDetail":{"code":"PU","statusByLocale":"Picked up","description":"Picked up"},"scanEvents":[{"date":"2026-01-01T10:00:00-05:00","eventDescription":"Picked up","scanLocation":{"city":"MEMPHIS","stateOrProvinceCode":"TN","countryCode":"US"}}]}`

func fxBody(results ...string) string {
	return `{"output":{"completeTrackResults":[{"trackResults":[` + strings.Join(results, ",") + `]}]}}`
}

func fxErrBody(codes ...string) string {
	items := make([]string, 0, len(codes))
	for _, c := range codes {
		items = append(items, fmt.Sprintf(`{"code":%q,"message":"echo `+fxSecret+`"}`, c))
	}
	return `{"errors":[` + strings.Join(items, ",") + `]}`
}

// fxReq is one request the stub received.
type fxReq struct {
	method, host, path, rawQuery            string
	contentType, accept, auth, locale, body string
}

// fedexStub is a TLS stub of the carrier. Its host is what the adapter under
// test is told to use; a request to any other host is refused by the client.
type fedexStub struct {
	t       *testing.T
	srv     *httptest.Server
	host    string
	blocked atomic.Int32

	mu        sync.Mutex
	reqs      []fxReq
	tokenHits int
	trackHits int
	token     func(w http.ResponseWriter, r *http.Request)
	track     func(w http.ResponseWriter, r *http.Request)
}

func newFedExStub(t *testing.T) *fedexStub {
	t.Helper()
	s := &fedexStub{t: t}
	s.token = func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"`+fxToken+`","token_type":"bearer","expires_in":3600,"scope":"CXS"}`)
	}
	s.setTrack(http.StatusOK, fxBody(fxPickedUp))

	s.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.reqs = append(s.reqs, fxReq{
			method: r.Method, host: r.Host, path: r.URL.Path, rawQuery: r.URL.RawQuery,
			contentType: r.Header.Get("Content-Type"), accept: r.Header.Get("Accept"),
			auth: r.Header.Get("Authorization"), locale: r.Header.Get("X-locale"), body: string(b),
		})
		var h func(http.ResponseWriter, *http.Request)
		switch r.URL.Path {
		case fedexTokenPath:
			s.tokenHits++
			h = s.token
		case fedexTrackPath:
			s.trackHits++
			h = s.track
		}
		s.mu.Unlock()
		if h == nil {
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}))
	s.host = strings.TrimPrefix(s.srv.URL, "https://")
	t.Cleanup(func() {
		s.srv.Close()
		if n := s.blocked.Load(); n != 0 {
			t.Errorf("%d requests to a host other than the stub were attempted", n)
		}
	})
	return s
}

func (s *fedexStub) setTrack(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.track = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func (s *fedexStub) setToken(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func (s *fedexStub) hits() (token, track int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokenHits, s.trackHits
}

func (s *fedexStub) requests() []fxReq {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]fxReq(nil), s.reqs...)
}

type hostGuard struct {
	host    string
	blocked *atomic.Int32
	next    http.RoundTripper
}

func (g hostGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != g.host {
		g.blocked.Add(1)
		return nil, errors.New("blocked: not the stub host")
	}
	return g.next.RoundTrip(req)
}

// client is the client factory of the handler under test: it trusts the stub,
// follows no redirect (as the guarded production client does) and refuses any
// host but the stub's.
func (s *fedexStub) client(timeout time.Duration) *http.Client {
	base := s.srv.Client()
	return &http.Client{
		Transport:     hostGuard{host: s.host, blocked: &s.blocked, next: base.Transport},
		Timeout:       timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// closedClient fails every request. It is the client of every FedEx test that
// has no stub, so that no test can reach a carrier.
func closedClient(time.Duration) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("blocked: no network in tests")
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newFedExTestHandler(t *testing.T, st *fedexStub, o ...func(*options)) (*Handler, *fakeClock) {
	t.Helper()
	opt := options{newClient: closedClient}
	host := fedexHost
	if st != nil {
		opt.newClient = st.client
		host = st.host
	}
	for _, f := range o {
		f(&opt)
	}
	return newTestHandler(t, opt, newFedExAdapterAt(host))
}

// newFedExAdapterAt makes an adapter for a stub host. It exists only in tests,
// so that the production binary has no constructor that takes a host.
func newFedExAdapterAt(host string) *fedexAdapter { return &fedexAdapter{host: host} }

func fxCall(h *Handler, number string) (string, bool, error) {
	return h.Call(context.Background(), makeCall("fedex", number, fxKey))
}

func fxDecode(t *testing.T, h *Handler, number string) Result {
	t.Helper()
	text, isError, err := fxCall(h, number)
	if err != nil || isError {
		t.Fatalf("expected a result, got %q %v %v", text, isError, err)
	}
	return decode(t, text)
}

func fxExpectCode(t *testing.T, h *Handler, number string, code ErrCode) {
	t.Helper()
	text, isError, err := fxCall(h, number)
	expectError(t, text, isError, err, code, "fedex")
}

func fxExpectSecured(t *testing.T, h *Handler) {
	t.Helper()
	r := fxDecode(t, h, fxNumber)
	if r.Status != StatusUnknown || r.StatusText != fedexSecuredText || len(r.Events) != 0 || !r.TimeZoneKnown {
		t.Errorf("expected the secured delivery result, got %+v", r)
	}
}

func Test_fedex_tool(t *testing.T) {
	a := newFedExAdapter()
	if a.Carrier() != "fedex" {
		t.Fatalf("carrier = %q", a.Carrier())
	}
	tool := a.Tool()
	if tool.Name != "track_fedex" {
		t.Fatalf("tool name = %q", tool.Name)
	}
	// the rule of aicallhandler.validMcpToolName: [A-Za-z0-9_-], at most 64-len("mcp_")-8-1 bytes
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,51}$`).MatchString(tool.Name) {
		t.Errorf("tool name %q is not accepted by the tool name rule", tool.Name)
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	norm, rep := mcpschema.Normalize(schema, 16<<10)
	if rep.ToolDropped || rep.DroppedKeys != 0 || rep.Rewrites != 0 || rep.DroppedPropsN != 0 {
		t.Errorf("the schema must pass the normalizer unchanged: %+v", rep)
	}
	got, _ := json.Marshal(norm)
	want, _ := json.Marshal(schema)
	if string(got) != string(want) {
		t.Errorf("normalized schema differs:\n%s\n%s", got, want)
	}
	if !strings.Contains(tool.Description, "FedEx") || len(tool.Description) > 600 {
		t.Errorf("unexpected description %q", tool.Description)
	}
}

func Test_fedex_secured_text(t *testing.T) {
	const want = "Secured delivery shipment: FedEx does not provide its status or scan events through this lookup. The caller must check it on the carrier website or with the carrier."
	if fedexSecuredText != want {
		t.Errorf("unexpected secured text %q", fedexSecuredText)
	}
	if len([]rune(fedexSecuredText)) > maxTextLen {
		t.Errorf("the secured text is cut by finalize: %d runes", len([]rune(fedexSecuredText)))
	}
	if sanitizeText(fedexSecuredText, maxTextLen) != fedexSecuredText {
		t.Errorf("the secured text is changed by sanitizeText")
	}
}

func Test_fedex_NormalizeNumber(t *testing.T) {
	a := newFedExAdapter()
	tests := []struct {
		raw    string
		expect string
		ok     bool
	}{
		{"ABCD12345678", "ABCD12345678", true},
		{" abcd-1234 5678 ", "ABCD12345678", true},
		{"1234567", "", false},         // 7
		{"12345678", "12345678", true}, // 8
		{strings.Repeat("1", 40), strings.Repeat("1", 40), true},
		{strings.Repeat("1", 41), "", false},
		{"", "", false},
		{"ABCD1234567\u00e9", "", false},
		{"ABCD1234\x005678", "", false},
		{"ABCD1234567;DROP", "", false},
		{"ABCD12345678\n", "ABCD12345678", true}, // trimmed
		{"ABC D!1234567", "", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.raw), func(t *testing.T) {
			n, ok := a.NormalizeNumber(tt.raw)
			if ok != tt.ok || (ok && n != tt.expect) {
				t.Errorf("NormalizeNumber(%q) = (%q, %v), expected (%q, %v)", tt.raw, n, ok, tt.expect, tt.ok)
			}
		})
	}
}

func Test_fedex_productionConstants(t *testing.T) {
	a := newFedExAdapter()
	if a.host != "apis.fedex.com" || fedexHost != "apis.fedex.com" {
		t.Fatalf("host = %q", a.host)
	}
	if got, want := a.url(fedexTokenPath), "https://"+fedexHost+fedexTokenPath; got != want {
		t.Errorf("token url = %q", got)
	}
	if got, want := a.url(fedexTrackPath), "https://"+fedexHost+fedexTrackPath; got != want {
		t.Errorf("track url = %q", got)
	}
	if !a.checkURL(a.url(fedexTokenPath), fedexTokenPath) || !a.checkURL(a.url(fedexTrackPath), fedexTrackPath) {
		t.Errorf("the production urls must pass the check")
	}
	if fedexTokenPath != "/oauth/token" || fedexTrackPath != "/track/v1/trackingnumbers" {
		t.Errorf("unexpected paths %q %q", fedexTokenPath, fedexTrackPath)
	}
	if a.MaxBody() != 0 {
		t.Errorf("MaxBody = %d", a.MaxBody())
	}
}

func Test_fedex_checkURL(t *testing.T) {
	a := newFedExAdapterAt("stub.test")
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{"good", "https://stub.test/oauth/token", true},
		{"http", "http://stub.test/oauth/token", false},
		{"other host", "https://other.test/oauth/token", false},
		{"host with a suffix", "https://stub.test.other.test/oauth/token", false},
		{"user info", "https://user@stub.test/oauth/token", false},
		{"other path", "https://stub.test/other", false},
		{"query", "https://stub.test/oauth/token?x=1", false},
		{"not a url", "://", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.checkURL(tt.raw, "/oauth/token"); got != tt.ok {
				t.Errorf("checkURL(%q) = %v", tt.raw, got)
			}
		})
	}
}

func Test_fedex_requestShape(t *testing.T) {
	st := newFedExStub(t)
	h, _ := newFedExTestHandler(t, st)

	r := fxDecode(t, h, " abcd-1234 5678 ")
	if r.Carrier != "fedex" || r.TrackingNumber != fxNumber || r.Status != StatusInTransit {
		t.Fatalf("unexpected result %+v", r)
	}

	reqs := st.requests()
	if len(reqs) != 2 {
		t.Fatalf("expected a token and a track request, got %d", len(reqs))
	}
	tok, trk := reqs[0], reqs[1]

	if tok.method != http.MethodPost || tok.path != fedexTokenPath || tok.rawQuery != "" || tok.host != st.host {
		t.Errorf("unexpected token request %+v", tok)
	}
	if tok.contentType != "application/x-www-form-urlencoded" || tok.auth != "" {
		t.Errorf("the token request must be a form without an Authorization header: %+v", tok)
	}
	form, err := url.ParseQuery(tok.body)
	if err != nil {
		t.Fatal(err)
	}
	if form.Get("grant_type") != "client_credentials" || form.Get("client_id") != fxID || form.Get("client_secret") != fxSecret {
		t.Errorf("unexpected token form %v", form)
	}

	if trk.method != http.MethodPost || trk.path != fedexTrackPath || trk.rawQuery != "" || trk.host != st.host {
		t.Errorf("unexpected track request %+v", trk)
	}
	if trk.contentType != "application/json" || trk.accept != "application/json" || trk.auth != "Bearer "+fxToken || trk.locale != "en_US" {
		t.Errorf("unexpected track headers %+v", trk)
	}
	if strings.Contains(trk.body, fxID) || strings.Contains(trk.body, fxSecret) {
		t.Errorf("the key is in the track body")
	}
	var body struct {
		IncludeDetailedScans bool `json:"includeDetailedScans"`
		TrackingInfo         []struct {
			TrackingNumberInfo struct {
				TrackingNumber string `json:"trackingNumber"`
			} `json:"trackingNumberInfo"`
		} `json:"trackingInfo"`
	}
	if err := json.Unmarshal([]byte(trk.body), &body); err != nil {
		t.Fatal(err)
	}
	if !body.IncludeDetailedScans || len(body.TrackingInfo) != 1 || body.TrackingInfo[0].TrackingNumberInfo.TrackingNumber != fxNumber {
		t.Errorf("unexpected track body %s", trk.body)
	}
	for _, rq := range reqs {
		if strings.Contains(rq.path+rq.rawQuery, fxID) || strings.Contains(rq.path+rq.rawQuery, fxSecret) || strings.Contains(rq.path+rq.rawQuery, fxNumber) {
			t.Errorf("the key or the number is in a url: %+v", rq)
		}
	}
}

func Test_fedex_tokenErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		code     ErrCode
		cooldown time.Duration
	}{
		{"401 is a rejected key", 401, `{"errors":[{"code":"NOT.AUTHORIZED.ERROR","message":"` + fxSecret + `"}]}`, CodeCredentialInvalid, 10 * time.Minute},
		{"403 is a short block", 403, `{}`, CodeCarrierUnavailable, time.Minute},
		{"429 is busy without a cooldown", 429, `{}`, CodeServiceBusy, 0},
		{"400", 400, `{}`, CodeCarrierUnavailable, 0},
		{"500", 500, `{}`, CodeCarrierUnavailable, 0},
		{"503", 503, `{}`, CodeCarrierUnavailable, 0},
		{"302", 302, ``, CodeCarrierUnavailable, 0},
		{"broken json", 200, `{"access_token":`, CodeCarrierUnavailable, 0},
		{"wrong token type", 200, `{"access_token":"x","token_type":"mac","expires_in":3600}`, CodeCarrierUnavailable, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setToken(tt.status, tt.body)
			h, clk := newFedExTestHandler(t, st)

			fxExpectCode(t, h, fxNumber, tt.code)
			if th, kh := st.hits(); th != 1 || kh != 0 {
				t.Fatalf("token hits %d, track hits %d", th, kh)
			}

			// inside the cooldown no request is made; outside it the carrier is asked again
			if tt.cooldown > 0 {
				clk.add(tt.cooldown - time.Second)
				fxExpectCode(t, h, fxNumber, tt.code)
				if th, _ := st.hits(); th != 1 {
					t.Errorf("a blocked key reached the carrier: %d token hits", th)
				}
				clk.add(2 * time.Second)
			}
			fxExpectCode(t, h, fxNumber, tt.code)
			if th, _ := st.hits(); th != 2 {
				t.Errorf("expected the second token request, got %d", th)
			}
		})
	}
}

func Test_fedex_tokenRejectedKeyIsTriedAgainWhenReentered(t *testing.T) {
	st := newFedExStub(t)
	st.setToken(401, `{}`)
	h, _ := newFedExTestHandler(t, st)

	fxExpectCode(t, h, fxNumber, CodeCredentialInvalid)

	st.setToken(200, `{"access_token":"`+fxToken+`","token_type":"bearer","expires_in":3600}`)
	text, isError, err := h.Call(context.Background(), makeCall("fedex", fxNumber, "OTHER-ID:OTHER-SECRET"))
	if err != nil || isError {
		t.Fatalf("a re-entered key must be tried at once, got %q %v %v", text, isError, err)
	}
}

func Test_fedex_lookupStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		code     ErrCode // empty: the secured delivery result
		cooldown time.Duration
		cached   bool // the answer is kept for 120 seconds
		dropTok  bool
	}{
		{"400 without a code", 400, `{}`, CodeInvalidFormat, 0, true, false},
		{"400 with a broken body", 400, `<html>`, CodeInvalidFormat, 0, true, false},
		{"400 for an empty tracking number is our fault", 400, fxErrBody(fxEmpty), CodeCarrierUnavailable, 0, false, false},
		{"400 secured delivery", 400, fxErrBody(fxFAD), "", 0, true, false},
		{"400 secured delivery wins over the empty number code", 400, fxErrBody(fxEmpty, fxFAD), "", 0, true, false},
		{"401 drops the token", 401, `{}`, CodeCarrierUnavailable, 0, false, true},
		{"403 blocks the key for a minute", 403, `{}`, CodeCarrierUnavailable, time.Minute, false, false},
		{"404 without a code is not a missing number", 404, `{}`, CodeCarrierUnavailable, 0, false, false},
		{"404 resource not available", 404, fxErrBody("NOT.FOUND.ERROR"), CodeCarrierUnavailable, 0, false, false},
		{"404 no data", 404, fxErrBody(fxNotFound), CodeNotFound, 0, true, false},
		{"404 secured delivery", 404, fxErrBody(fxFAD), "", 0, true, false},
		{"429 blocks the key for a minute", 429, `{}`, CodeRateLimited, time.Minute, false, false},
		{"500", 500, `{}`, CodeCarrierUnavailable, 0, false, false},
		{"503", 503, `{}`, CodeCarrierUnavailable, 0, false, false},
		{"302", 302, ``, CodeCarrierUnavailable, 0, false, false},
		{"418", 418, `{}`, CodeCarrierUnavailable, 0, false, false},
		{"200 no data in the result", 200, fxBody(`{"error":{"code":"` + fxNotFound + `"}}`), CodeNotFound, 0, true, false},
		{"200 no data in the alerts", 200, `{"output":{"completeTrackResults":[],"alerts":[{"code":"` + fxNotFound + `"}]}}`, CodeNotFound, 0, true, false},
		{"200 secured delivery in the result", 200, fxBody(`{"error":{"code":"` + fxFAD + `"}}`), "", 0, true, false},
		{"200 secured delivery in the alerts", 200, `{"output":{"alerts":[{"code":"` + fxFAD + `"}]}}`, "", 0, true, false},
		{"200 another error code", 200, fxBody(`{"error":{"code":"SYSTEM.UNAVAILABLE.EXCEPTION"}}`), CodeCarrierUnavailable, 0, false, false},
		{"200 an error object without a code", 200, fxBody(`{"error":{},"latestStatusDetail":{"code":"PU","description":"Picked up"}}`), CodeCarrierUnavailable, 0, false, false},
		{"200 no result and no alert", 200, `{"output":{"completeTrackResults":[]}}`, CodeCarrierUnavailable, 0, false, false},
		{"200 an empty shell of a result", 200, fxBody(`{}`), CodeCarrierUnavailable, 0, false, false},
		{"200 a result without usable information", 200, fxBody(`{"trackingNumberInfo":{"trackingNumber":"ABCD12345678"},"scanEvents":[{"date":"  "}]}`), CodeCarrierUnavailable, 0, false, false},
		{"200 another alert does not hide a result", 200, strings.Replace(fxBody(fxPickedUp), `"completeTrackResults"`, `"alerts":[{"code":"SOME.WARNING"}],"completeTrackResults"`, 1), "ok", 0, true, false},
		{"200 a no data alert does not hide a result", 200, strings.Replace(fxBody(fxPickedUp), `"completeTrackResults"`, `"alerts":[{"code":"`+fxNotFound+`"}],"completeTrackResults"`, 1), "ok", 0, true, false},
		{"200 body over the limit", 200, `{"output":{"pad":"` + strings.Repeat("a", 1<<20+10) + `"}}`, CodeCarrierUnavailable, 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setTrack(tt.status, tt.body)
			h, clk := newFedExTestHandler(t, st)

			check := func() {
				t.Helper()
				switch tt.code {
				case "":
					fxExpectSecured(t, h)
				case "ok":
					if r := fxDecode(t, h, fxNumber); r.Status != StatusInTransit {
						t.Errorf("unexpected result %+v", r)
					}
				default:
					fxExpectCode(t, h, fxNumber, tt.code)
				}
			}

			check()
			if th, kh := st.hits(); th != 1 || kh != 1 {
				t.Fatalf("token hits %d, track hits %d", th, kh)
			}

			// the second call: kept answer, blocked key, or a new request
			check()
			th, kh := st.hits()
			switch {
			case tt.cached || tt.cooldown > 0:
				if kh != 1 {
					t.Errorf("the second call reached the carrier: %d track hits", kh)
				}
			default:
				if kh != 2 {
					t.Errorf("expected a new request, got %d track hits", kh)
				}
			}
			if tt.dropTok && th != 2 {
				t.Errorf("a 401 must drop the token, got %d token hits", th)
			}
			if !tt.dropTok && th != 1 {
				t.Errorf("unexpected token hits %d", th)
			}

			if tt.cooldown > 0 {
				clk.add(tt.cooldown - time.Second)
				check()
				if _, kh2 := st.hits(); kh2 != kh {
					t.Errorf("a blocked key reached the carrier")
				}
				clk.add(2 * time.Second)
				check()
				if _, kh2 := st.hits(); kh2 != kh+1 {
					t.Errorf("the key must be tried again after the cooldown, got %d track hits", kh2)
				}
			}
			if tt.cached {
				clk.add(resultTTL + time.Second)
				check()
				if _, kh2 := st.hits(); kh2 != kh+1 {
					t.Errorf("the answer must expire after %v, got %d track hits", resultTTL, kh2)
				}
			}
		})
	}
}

func Test_fedex_timeout(t *testing.T) {
	st := newFedExStub(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	st.mu.Lock()
	st.track = func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		case <-time.After(10 * time.Second):
		}
	}
	st.mu.Unlock()

	h, _ := newFedExTestHandler(t, st, func(o *options) {
		o.reserve = time.Millisecond
		o.minBudget = time.Millisecond
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	text, isError, err := h.Call(ctx, makeCall("fedex", fxNumber, fxKey))
	expectError(t, text, isError, err, CodeCarrierUnavailable, "fedex")
	if n := len(h.state("fedex").slots); n != 0 {
		t.Errorf("%d slots leaked", n)
	}
}

func Test_fedex_redirectIsNotFollowed(t *testing.T) {
	var otherHits atomic.Int32
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { otherHits.Add(1) }))
	defer other.Close()

	st := newFedExStub(t)
	st.mu.Lock()
	st.track = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/x", http.StatusFound)
	}
	st.mu.Unlock()
	h, _ := newFedExTestHandler(t, st)

	fxExpectCode(t, h, fxNumber, CodeCarrierUnavailable)
	if otherHits.Load() != 0 || st.blocked.Load() != 0 {
		t.Errorf("the redirect was followed: %d, %d", otherHits.Load(), st.blocked.Load())
	}
}

// Test_fedex_noNetworkWithoutAStub shows what stops a test without a stub: the
// production host is only ever reached through a client that refuses it.
func Test_fedex_noNetworkWithoutAStub(t *testing.T) {
	h, _ := newFedExTestHandler(t, nil)
	fxExpectCode(t, h, fxNumber, CodeCarrierUnavailable)
}

func Test_fedex_allowList(t *testing.T) {
	const forbidden = `"recipientInformation":{"contact":{"personName":"SENTINEL-RECIPIENT"},"address":{"streetLines":["SENTINEL-STREET"],"postalCode":"SENTINEL-ZIP"}},` +
		`"shipperInformation":{"contact":{"personName":"SENTINEL-SHIPPER","phoneNumber":"SENTINEL-PHONE"}},` +
		`"deliveryDetails":{"receivedByName":"SENTINEL-RECEIVED","signedByName":"SENTINEL-SIGNED","actualDeliveryAddress":{"city":"SENTINEL-ACTUAL"}},` +
		`"lastUpdatedDestinationAddress":{"city":"SENTINEL-LASTADDR"},"destinationLocation":{"locationId":"SENTINEL-DEST"},"holdAtLocation":{"locationId":"SENTINEL-HOLD"},` +
		`"availableImages":[{"type":"SENTINEL-IMAGE"}],"packageDetails":{"packagingDescription":{"description":"SENTINEL-PACKAGE"}},"customDeliveryOptions":[{"description":"SENTINEL-OPTION"}],` +
		`"distanceToDestination":{"value":"SENTINEL-DISTANCE"},`

	events := make([]string, 0, 12)
	for i := 12; i >= 1; i-- {
		events = append(events, fmt.Sprintf(`{"date":"2026-01-%02dT10:00:00-05:00","eventDescription":"Event %d mail a@example.invalid or call 555-010-0123","scanLocation":{"city":"MEMPHIS","stateOrProvinceCode":"TN","countryCode":"US","postalCode":"SENTINEL-EVENT-ZIP","streetLines":["SENTINEL-EVENT-STREET"]}}`, i, i))
	}
	result := `{"trackingNumberInfo":{"trackingNumber":"ABCD12345678"},` + forbidden +
		`"latestStatusDetail":{"code":"PU","statusByLocale":"Picked up","description":"Picked up","scanLocation":{"city":"SENTINEL-LATEST-CITY"}},` +
		`"scanEvents":[` + strings.Join(events, ",") + `]}`

	st := newFedExStub(t)
	st.setTrack(200, fxBody(result))
	h, _ := newFedExTestHandler(t, st)

	text, isError, err := fxCall(h, fxNumber)
	if err != nil || isError {
		t.Fatalf("%q %v %v", text, isError, err)
	}
	if strings.Contains(text, "SENTINEL") {
		t.Errorf("a field outside the allow list reached the output: %s", text)
	}
	if strings.Contains(text, "a@example.invalid") || strings.Contains(text, "555-010-0123") {
		t.Errorf("an e-mail address or a phone number was not masked: %s", text)
	}

	r := decode(t, text)
	if len(r.Events) != maxEvents || !r.EventsTruncated {
		t.Errorf("expected %d events and a truncation flag, got %d %v", maxEvents, len(r.Events), r.EventsTruncated)
	}
	if len(r.Display.Fields) > maxDisplayFields {
		t.Errorf("%d display fields", len(r.Display.Fields))
	}
	if r.Events[0].Location == nil || r.Events[0].Location.Text != "MEMPHIS, TN, US" {
		t.Errorf("unexpected location %+v", r.Events[0].Location)
	}
}

func Test_fedex_status(t *testing.T) {
	status := func(code string) string {
		return fmt.Sprintf(`{"trackingNumberInfo":{"trackingNumber":"ABCD12345678"},"latestStatusDetail":{"code":%q,"description":"text"}}`, code)
	}
	tests := []struct {
		name   string
		result string
		status Status
		code   string
	}{
		{"OC", status("OC"), StatusLabelCreated, "OC"},
		{"PU", status("PU"), StatusInTransit, "PU"},
		{"AR", status("AR"), StatusInTransit, "AR"},
		{"DP", status("DP"), StatusInTransit, "DP"},
		{"OD", status("OD"), StatusOutForDelivery, "OD"},
		{"DL", status("DL"), StatusDelivered, "DL"},
		{"an unmapped code is unknown and keeps its code", status("HL"), StatusUnknown, "HL"},
		{"DE is not mapped either", status("DE"), StatusUnknown, "DE"},
		{"no code", `{"latestStatusDetail":{"description":"text"}}`, StatusUnknown, ""},
		{"an actual delivery date wins over the code", `{"latestStatusDetail":{"code":"PU","description":"text"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":"2026-01-02T10:00:00-05:00"}]}`, StatusDelivered, "PU"},
		{"an actual delivery without a date does not count", `{"latestStatusDetail":{"code":"PU","description":"text"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":""}]}`, StatusInTransit, "PU"},
		{"an actual delivery with a blank date does not count", `{"latestStatusDetail":{"code":"PU","description":"text"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":"  "}]}`, StatusInTransit, "PU"},
		{"another date type does not count", `{"latestStatusDetail":{"code":"PU","description":"text"},"dateAndTimes":[{"type":"SHIP","dateTime":"2026-01-02T10:00:00-05:00"}]}`, StatusInTransit, "PU"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setTrack(200, fxBody(tt.result))
			h, _ := newFedExTestHandler(t, st)

			r := fxDecode(t, h, fxNumber)
			if r.Status != tt.status {
				t.Errorf("status = %s, expected %s", r.Status, tt.status)
			}
			got := ""
			if r.CarrierStatusCode != nil {
				got = *r.CarrierStatusCode
			}
			if got != tt.code {
				t.Errorf("carrier_status_code = %q, expected %q", got, tt.code)
			}
		})
	}
}

func Test_fedex_fields(t *testing.T) {
	tests := []struct {
		name   string
		result string
		check  func(t *testing.T, r Result)
	}{
		{
			"status text, reason and service",
			`{"latestStatusDetail":{"code":"DP","statusByLocale":"Departed FedEx location","description":"Departed","ancillaryDetails":[{"reasonDescription":"Weather delay"}]},"serviceDetail":{"description":"FedEx Ground","shortDescription":"FG"}}`,
			func(t *testing.T, r Result) {
				if r.StatusText != "Departed FedEx location; Weather delay" {
					t.Errorf("status_text = %q", r.StatusText)
				}
				if r.Service == nil || *r.Service != "FedEx Ground" {
					t.Errorf("service = %v", r.Service)
				}
			},
		},
		{
			"description when there is no locale text, short service name",
			`{"latestStatusDetail":{"code":"DP","description":"Departed"},"serviceDetail":{"shortDescription":"FG"}}`,
			func(t *testing.T, r Result) {
				if r.StatusText != "Departed" || r.Service == nil || *r.Service != "FG" {
					t.Errorf("unexpected %q %v", r.StatusText, r.Service)
				}
			},
		},
		{
			"events are sorted newest first and the last event is the newest",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"scanEvents":[` +
				`{"date":"2026-01-01T10:00:00-05:00","eventDescription":"first"},` +
				`{"date":"2026-01-03T10:00:00-05:00","eventDescription":"third","exceptionDescription":"late truck"},` +
				`{"date":"2026-01-02T10:00:00-05:00","eventDescription":"second"}]}`,
			func(t *testing.T, r Result) {
				if len(r.Events) != 3 || r.Events[0].Description != "third - late truck" || r.Events[1].Description != "second" || r.Events[2].Description != "first" {
					t.Errorf("unexpected events %+v", r.Events)
				}
				if r.LastEvent == nil || r.LastEvent.Description != "third - late truck" {
					t.Errorf("unexpected last event %+v", r.LastEvent)
				}
				if !r.TimeZoneKnown {
					t.Errorf("every time has an offset")
				}
			},
		},
		{
			"an event time that cannot be compared goes after the others and is a local time",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"scanEvents":[` +
				`{"date":"2026-01-05T10:00:00","eventDescription":"local"},` +
				`{"date":"2026-01-01T10:00:00-05:00","eventDescription":"old"},` +
				`{"date":"2026-01-02T10:00:00-05:00","eventDescription":"newer"}]}`,
			func(t *testing.T, r Result) {
				if len(r.Events) != 3 || r.Events[0].Description != "newer" || r.Events[2].Description != "local" {
					t.Fatalf("unexpected events %+v", r.Events)
				}
				if r.Events[2].Time != nil || r.Events[2].TimeLocal == nil || *r.Events[2].TimeLocal != "2026-01-05T10:00:00" {
					t.Errorf("unexpected local time %+v", r.Events[2])
				}
				if *r.Events[0].Time != "2026-01-02T10:00:00-05:00" {
					t.Errorf("the time must be the original text, got %q", *r.Events[0].Time)
				}
				if r.TimeZoneKnown {
					t.Errorf("a local time is in the output")
				}
			},
		},
		{
			"empty events are dropped",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"scanEvents":[null,{},{"date":" ","eventDescription":""},{"eventDescription":"real"}]}`,
			func(t *testing.T, r Result) {
				if len(r.Events) != 1 || r.Events[0].Description != "real" {
					t.Errorf("unexpected events %+v", r.Events)
				}
			},
		},
		{
			"an event without a time is kept",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"scanEvents":[{"eventDescription":"no time"}]}`,
			func(t *testing.T, r Result) {
				if len(r.Events) != 1 || r.Events[0].Time != nil || r.Events[0].TimeLocal != nil || !r.TimeZoneKnown {
					t.Errorf("unexpected events %+v", r.Events)
				}
			},
		},
		{
			"delivery date with an offset",
			`{"latestStatusDetail":{"code":"DL","description":"Delivered"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":"2026-01-02T10:00:00-05:00"}]}`,
			func(t *testing.T, r Result) {
				if r.DeliveredAt == nil || *r.DeliveredAt != "2026-01-02T10:00:00-05:00" || !r.TimeZoneKnown {
					t.Errorf("unexpected %v %v", r.DeliveredAt, r.TimeZoneKnown)
				}
			},
		},
		{
			"delivery date without an offset",
			`{"latestStatusDetail":{"code":"DL","description":"Delivered"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":"2026-01-02T10:00:00"}]}`,
			func(t *testing.T, r Result) {
				if r.DeliveredAt == nil || *r.DeliveredAt != "2026-01-02T10:00:00" || r.TimeZoneKnown {
					t.Errorf("unexpected %v %v", r.DeliveredAt, r.TimeZoneKnown)
				}
			},
		},
		{
			"delivery date with a date only",
			`{"latestStatusDetail":{"code":"DL","description":"Delivered"},"dateAndTimes":[{"type":"ACTUAL_DELIVERY","dateTime":"2026-01-02"}]}`,
			func(t *testing.T, r Result) {
				if r.DeliveredAt == nil || *r.DeliveredAt != "2026-01-02" || r.TimeZoneKnown {
					t.Errorf("unexpected %v %v", r.DeliveredAt, r.TimeZoneKnown)
				}
			},
		},
		{
			"estimated delivery as a window",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"estimatedDeliveryTimeWindow":{"window":{"begins":"2026-01-10T08:00:00-06:00","ends":"2026-01-10T18:00:00-06:00"}}}`,
			func(t *testing.T, r Result) {
				e := r.EstimatedDelivery
				if e.From == nil || *e.From != "2026-01-10T08:00:00-06:00" || e.To == nil || *e.To != "2026-01-10T18:00:00-06:00" || e.Date != nil {
					t.Errorf("unexpected %+v", e)
				}
			},
		},
		{
			"a window whose begin has no offset falls back to the estimated date",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"estimatedDeliveryTimeWindow":{"window":{"begins":"2026-01-10T08:00:00","ends":"2026-01-10T18:00:00-06:00"}},"dateAndTimes":[{"type":"ESTIMATED_DELIVERY","dateTime":"2026-01-11T00:00:00"}]}`,
			func(t *testing.T, r Result) {
				e := r.EstimatedDelivery
				if e.From != nil || e.To != nil || e.Date == nil || *e.Date != "2026-01-11" || !r.TimeZoneKnown {
					t.Errorf("unexpected %+v %v", e, r.TimeZoneKnown)
				}
			},
		},
		{
			"a window whose end has no offset falls back to the estimated date",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"estimatedDeliveryTimeWindow":{"window":{"begins":"2026-01-10T08:00:00-06:00","ends":"2026-01-10T18:00:00"}},"dateAndTimes":[{"type":"ESTIMATED_DELIVERY","dateTime":"2026-01-11T00:00:00"}]}`,
			func(t *testing.T, r Result) {
				e := r.EstimatedDelivery
				if e.From != nil || e.To != nil || e.Date == nil || *e.Date != "2026-01-11" {
					t.Errorf("unexpected %+v", e)
				}
			},
		},
		{
			"an estimated date that is not a date is dropped",
			`{"latestStatusDetail":{"code":"PU","description":"x"},"dateAndTimes":[{"type":"ESTIMATED_DELIVERY","dateTime":"soon, maybe"}]}`,
			func(t *testing.T, r Result) {
				e := r.EstimatedDelivery
				if e.From != nil || e.To != nil || e.Date != nil {
					t.Errorf("unexpected %+v", e)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setTrack(200, fxBody(tt.result))
			h, _ := newFedExTestHandler(t, st)
			tt.check(t, fxDecode(t, h, fxNumber))
		})
	}
}

func Test_fedex_timeZoneKnownCountsOnlyWhatIsOutput(t *testing.T) {
	ev := func(day int, zone string) string {
		return fmt.Sprintf(`{"date":"2026-01-%02dT10:00:00%s","eventDescription":"e%d"}`, day, zone, day)
	}
	// the oldest event has no offset and is cut away by the limit of eight
	events := []string{`{"date":"2025-12-01T10:00:00","eventDescription":"cut"}`}
	for d := 1; d <= 8; d++ {
		events = append(events, ev(d, "-05:00"))
	}
	st := newFedExStub(t)
	st.setTrack(200, fxBody(`{"latestStatusDetail":{"code":"PU","description":"x"},"scanEvents":[`+strings.Join(events, ",")+`]}`))
	h, _ := newFedExTestHandler(t, st)

	r := fxDecode(t, h, fxNumber)
	if len(r.Events) != 8 || !r.EventsTruncated || !r.TimeZoneKnown {
		t.Errorf("a time that is cut away must not set the flag: %d %v %v", len(r.Events), r.EventsTruncated, r.TimeZoneKnown)
	}
}

func Test_fedex_resultSelection(t *testing.T) {
	other := `{"trackingNumberInfo":{"trackingNumber":"ZZZZ99999999"},"latestStatusDetail":{"code":"DL","description":"other package"}}`
	mine := `{"trackingNumberInfo":{"trackingNumber":"abcd-1234 5678"},"latestStatusDetail":{"code":"OD","description":"mine"}}`
	tests := []struct {
		name   string
		body   string
		expect string // the status text, or an error code
	}{
		{"the matching result is chosen", fxBody(other, mine), "mine"},
		{"the first result is used when none matches", fxBody(other, `{"trackingNumberInfo":{"trackingNumber":"QQQQ11111111"},"latestStatusDetail":{"code":"PU","description":"third"}}`), "other package"},
		{"null results are skipped", fxBody(`null`, other), "other package"},
		{"the matching result is an error", fxBody(other, `{"trackingNumberInfo":{"trackingNumber":"ABCD12345678"},"error":{"code":"`+fxNotFound+`"}}`), string(CodeNotFound)},
		{"only null results", fxBody(`null`), string(CodeCarrierUnavailable)},
		{"null track groups", `{"output":{"completeTrackResults":[null]}}`, string(CodeCarrierUnavailable)},
		{"results in several groups", `{"output":{"completeTrackResults":[{"trackResults":[` + other + `]},{"trackResults":[` + mine + `]}]}}`, "mine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setTrack(200, tt.body)
			h, _ := newFedExTestHandler(t, st)

			text, isError, err := fxCall(h, fxNumber)
			if err != nil {
				t.Fatal(err)
			}
			if isError {
				expectError(t, text, isError, err, ErrCode(tt.expect), "fedex")
				return
			}
			if r := decode(t, text); r.StatusText != tt.expect {
				t.Errorf("status_text = %q, expected %q", r.StatusText, tt.expect)
			}
		})
	}
}

func Test_fedex_oddShapes(t *testing.T) {
	status := `"latestStatusDetail":{"code":"PU","description":"Picked up"}`
	tests := []struct {
		name   string
		status int
		body   string
		code   ErrCode // empty: a result
	}{
		{"null", 200, `null`, CodeCarrierUnavailable},
		{"an array", 200, `[]`, CodeCarrierUnavailable},
		{"empty object", 200, `{}`, CodeCarrierUnavailable},
		{"output null", 200, `{"output":null}`, CodeCarrierUnavailable},
		{"wrong type of results", 200, `{"output":{"completeTrackResults":"x"}}`, CodeCarrierUnavailable},
		{"null group", 200, `{"output":{"completeTrackResults":[null]}}`, CodeCarrierUnavailable},
		{"null track result", 200, fxBody(`null`), CodeCarrierUnavailable},
		{"null alerts only", 200, `{"output":{"alerts":[null]}}`, CodeCarrierUnavailable},
		{"null errors on a 400", 400, `{"errors":[null]}`, CodeInvalidFormat},
		{"null errors on a 404", 404, `{"errors":[null]}`, CodeCarrierUnavailable},
		{"null events with a status", 200, fxBody(`{` + status + `,"scanEvents":[null]}`), ""},
		{"null events without a status", 200, fxBody(`{"scanEvents":[null]}`), CodeCarrierUnavailable},
		{"a number where a text is expected", 200, fxBody(`{` + status + `,"scanEvents":[{"date":123}]}`), CodeCarrierUnavailable},
		{"an object where a list is expected", 200, fxBody(`{` + status + `,"scanEvents":{"a":1}}`), CodeCarrierUnavailable},
		{"a text where an object is expected", 200, fxBody(`{` + status + `,"serviceDetail":"x"}`), CodeCarrierUnavailable},
		{"deep nesting", 200, strings.Repeat("[", 20000), CodeCarrierUnavailable},
		{"many null events", 200, fxBody(`{` + status + `,"scanEvents":[` + strings.TrimSuffix(strings.Repeat("null,", 20000), ",") + `]}`), ""},
		{"unknown fields everywhere", 200, fxBody(`{` + status + `,"surprise":{"a":[1,2,3]},"output":1}`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFedExStub(t)
			st.setTrack(tt.status, tt.body)
			h, _ := newFedExTestHandler(t, st)

			text, isError, err := fxCall(h, fxNumber)
			if err != nil {
				t.Fatalf("a system error: %v", err)
			}
			if tt.code == "" {
				if isError {
					t.Fatalf("expected a result, got %q", text)
				}
				decode(t, text)
				return
			}
			expectError(t, text, isError, err, tt.code, "fedex")
		})
	}
}

func Test_fedex_tokenIsCached(t *testing.T) {
	st := newFedExStub(t)
	h, clk := newFedExTestHandler(t, st)

	fxDecode(t, h, fxNumber)
	fxDecode(t, h, "EFGH12345678")
	if th, kh := st.hits(); th != 1 || kh != 2 {
		t.Fatalf("one token for two numbers expected, got %d token hits and %d track hits", th, kh)
	}

	clk.add(58 * time.Minute)
	fxDecode(t, h, "IJKL12345678")
	if th, _ := st.hits(); th != 1 {
		t.Errorf("the token is valid for 59 minutes, got %d token hits", th)
	}

	clk.add(2 * time.Minute)
	fxDecode(t, h, "MNOP12345678")
	if th, _ := st.hits(); th != 2 {
		t.Errorf("the token must be exchanged again after it expired, got %d token hits", th)
	}
}

func Test_fedex_resultIsCached(t *testing.T) {
	st := newFedExStub(t)
	h, clk := newFedExTestHandler(t, st)

	first := fxDecode(t, h, fxNumber)
	second := fxDecode(t, h, fxNumber)
	if first.Cached || !second.Cached {
		t.Errorf("cached flags: %v %v", first.Cached, second.Cached)
	}
	if _, kh := st.hits(); kh != 1 {
		t.Errorf("track hits = %d", kh)
	}
	clk.add(resultTTL + time.Second)
	fxDecode(t, h, fxNumber)
	if _, kh := st.hits(); kh != 2 {
		t.Errorf("track hits = %d", kh)
	}
}

func Test_fedex_keyAndNumberNeverAppear(t *testing.T) {
	scenarios := []struct {
		name        string
		tokenStatus int
		trackStatus int
		trackBody   string
	}{
		{"success", 200, 200, fxBody(fxPickedUp)},
		{"token rejected", 401, 200, fxBody(fxPickedUp)},
		{"token blocked", 403, 200, fxBody(fxPickedUp)},
		{"track 500", 200, 500, `{"errors":[{"code":"X","message":"echo ` + fxSecret + ` ` + fxNumber + `"}]}`},
		{"track 401", 200, 401, fxErrBody("NOT.AUTHORIZED.ERROR")},
		{"track 429", 200, 429, fxErrBody("RATE")},
		{"track 400", 200, 400, fxErrBody(fxEmpty)},
		{"secured delivery with a message that echoes the key", 200, 400, fxErrBody(fxFAD)},
		{"no data", 200, 404, fxErrBody(fxNotFound)},
		{"another error code in a 200", 200, 200, fxBody(`{"error":{"code":"SYSTEM","message":"echo ` + fxSecret + `"}}`)},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			st := newFedExStub(t)
			if sc.tokenStatus != 200 {
				st.setToken(sc.tokenStatus, fxErrBody("NOT.AUTHORIZED.ERROR"))
			}
			st.setTrack(sc.trackStatus, sc.trackBody)
			h, _ := newFedExTestHandler(t, st)
			logs := logtest.New(t)

			var outputs []string
			for i := 0; i < 2; i++ {
				text, _, err := fxCall(h, fxNumber)
				if err != nil {
					outputs = append(outputs, err.Error())
				}
				outputs = append(outputs, text)
			}
			for _, o := range outputs {
				if strings.Contains(o, fxSecret) || strings.Contains(o, fxID) || strings.Contains(o, fxToken) {
					t.Errorf("a key or a token leaked into %q", o)
				}
			}
			logs.AssertAbsent(t, fxKey, fxID, fxSecret, fxToken, fxNumber, customerA.String())

			for _, key := range h.debugKeys("fedex") {
				for _, secret := range []string{fxID, fxSecret, fxToken, fxNumber, customerA.String()} {
					if strings.Contains(key, secret) {
						t.Errorf("a cache key contains %q", secret)
					}
				}
			}
		})
	}
}

func Test_fedex_errorTextsAreFixed(t *testing.T) {
	st := newFedExStub(t)
	st.setTrack(400, fxErrBody(fxEmpty))
	h, _ := newFedExTestHandler(t, st)

	text, isError, err := fxCall(h, fxNumber)
	if err != nil || !isError {
		t.Fatalf("%q %v %v", text, isError, err)
	}
	if want := Message(CodeCarrierUnavailable, "fedex", "client_id_secret"); text != want || len(text) > 200 {
		t.Errorf("unexpected text %q", text)
	}
}

func Test_fedex_invalidNumberIsRefusedBeforeTheKey(t *testing.T) {
	st := newFedExStub(t)
	h, _ := newFedExTestHandler(t, st)

	secretRead := false
	call := makeCall("fedex", "short", fxKey)
	call.Secret = func() (string, bool, error) { secretRead = true; return fxKey, true, nil }
	text, isError, err := h.Call(context.Background(), call)
	expectError(t, text, isError, err, CodeInvalidFormat, "fedex")
	if secretRead {
		t.Errorf("the key was read for an invalid number")
	}
	if th, kh := st.hits(); th != 0 || kh != 0 {
		t.Errorf("the carrier was reached")
	}
}

func Test_fedex_race(t *testing.T) {
	st := newFedExStub(t)
	h, _ := newFedExTestHandler(t, st)

	var wg sync.WaitGroup
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				number := fmt.Sprintf("RACE%08d", (g*7+i)%5)
				secret := fxKey
				if g%2 == 1 {
					secret = fmt.Sprintf("ID%d:SECRET%d", g%4, g%4)
				}
				_, _, _ = h.Call(context.Background(), makeCall("fedex", number, secret))
			}
		}(g)
	}
	wg.Wait()
	if n := len(h.state("fedex").slots); n != 0 {
		t.Errorf("%d slots leaked", n)
	}
}
