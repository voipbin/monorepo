package carriertrackhandler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

const (
	upsNumber      = "1Z999AA10123456784"
	upsToken       = "SENTINEL-TOKEN-VALUE"
	upsTrackStub   = "/track/"
	upsTokenStub   = "/oauth/token"
	upsActivityDay = "20260102"
)

// upsStub is a stub of the carrier: a token endpoint and a track endpoint.
type upsStub struct {
	srv *httptest.Server

	mu          sync.Mutex
	tokenStatus int
	tokenBody   string
	trackStatus int
	trackBody   string

	tokenCalls atomic.Int32
	trackCalls atomic.Int32

	tokenReq seenRequest
	trackReq seenRequest
}

type seenRequest struct {
	method, path, query, auth, transID, transSrc, accept, body string
}

func newUPSStub(t *testing.T) *upsStub {
	t.Helper()
	s := &upsStub{
		tokenStatus: 200,
		tokenBody:   `{"access_token":"` + upsToken + `","token_type":"Bearer","expires_in":"3600"}`,
		trackStatus: 200,
	}
	s.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen := seenRequest{
			method: r.Method, path: r.URL.Path, query: r.URL.RawQuery,
			auth: r.Header.Get("Authorization"), transID: r.Header.Get("transId"),
			transSrc: r.Header.Get("transactionSrc"), accept: r.Header.Get("Accept"), body: string(b),
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case strings.HasPrefix(r.URL.Path, upsTokenStub):
			s.tokenCalls.Add(1)
			s.tokenReq = seen
			w.WriteHeader(s.tokenStatus)
			_, _ = io.WriteString(w, s.tokenBody)
		case strings.HasPrefix(r.URL.Path, upsTrackStub):
			s.trackCalls.Add(1)
			s.trackReq = seen
			trackBody := s.trackBody
			if trackBody == "" {
				trackBody = answerFor(strings.TrimPrefix(r.URL.Path, upsTrackStub))
			}
			if s.trackStatus >= 300 && s.trackStatus < 400 {
				w.Header().Set("Location", "https://elsewhere.example.com/x")
			}
			w.WriteHeader(s.trackStatus)
			_, _ = io.WriteString(w, trackBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// answerFor is the default answer of the stub: an in-transit package of the number asked for.
func answerFor(number string) string {
	return fmt.Sprintf(`{"trackResponse":{"shipment":[{"inquiryNumber":%q,"package":[%s]}]}}`,
		number, pkgJSON(number, "I", "", actJSON(upsActivityDay, "101500", upsActivityDay, "151500", "-05:00", "Departed", "I")))
}

func (s *upsStub) set(f func(*upsStub)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

func (s *upsStub) adapter() *upsAdapter {
	return newUPSAdapter(s.srv.URL+upsTrackStub, s.srv.URL+upsTokenStub)
}

func (s *upsStub) client() *http.Client { return noRedirectClient(s.srv) }

func (s *upsStub) handler(t *testing.T) (*Handler, *fakeClock) {
	t.Helper()
	return newTestHandler(t, options{newClient: func(time.Duration) *http.Client { return s.client() }}, s.adapter())
}

func (s *upsStub) deps(clk *fakeClock) Deps {
	return Deps{Carrier: "ups", Client: s.client(), Now: clk.now, Tokens: newTokenCache(128, clk.now), TokenKey: "token-key", MaxBody: 1 << 20}
}

// ---- fixtures

func actJSON(date, tm, gmtDate, gmtTime, offset, desc, typ string) string {
	return fmt.Sprintf(`{"date":%q,"time":%q,"gmtDate":%q,"gmtTime":%q,"gmtOffset":%q,"location":{"address":{"city":"Atlanta","stateProvince":"GA","country":"US","countryCode":"US"}},"status":{"description":%q,"type":%q}}`,
		date, tm, gmtDate, gmtTime, offset, desc, typ)
}

// upsPackage builds a package; extra is raw json members (with a leading comma) for the package.
func pkgJSON(number, statusType, extra string, activities ...string) string {
	return fmt.Sprintf(`{"trackingNumber":%q,"currentStatus":{"code":"KB","description":"Status text","simplifiedTextDescription":"Simple","type":%q},"service":{"description":"UPS Ground"},"activity":[%s]%s}`,
		number, statusType, strings.Join(activities, ","), extra)
}

func answerJSON(packages ...string) string {
	return fmt.Sprintf(`{"trackResponse":{"shipment":[{"inquiryNumber":%q,"package":[%s]}]}}`, upsNumber, strings.Join(packages, ","))
}

func mustMap(t *testing.T, body string) *Result {
	t.Helper()
	res, cerr := mapUPS([]byte(body), upsNumber)
	if cerr != nil || res == nil {
		t.Fatalf("expected a result, got %+v %+v", res, cerr)
	}
	return res
}

func expectMapError(t *testing.T, body string, code ErrCode) {
	t.Helper()
	res, cerr := mapUPS([]byte(body), upsNumber)
	if res != nil || cerr == nil || cerr.Code != code {
		t.Fatalf("expected %s, got %+v %+v", code, res, cerr)
	}
}

func strVal(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

// ---- number and tool

func Test_UPS_NormalizeNumber(t *testing.T) {
	a := NewUPS()
	tests := []struct {
		name   string
		in     string
		expect string
		ok     bool
	}{
		{"plain", upsNumber, upsNumber, true},
		{"lower case", strings.ToLower(upsNumber), upsNumber, true},
		{"spaces and hyphens", "1Z 999-AA1 0123 456784", upsNumber, true},
		{"17 characters", upsNumber[:17], upsNumber[:17], false},
		{"19 characters", upsNumber + "5", upsNumber + "5", false},
		{"no 1Z prefix", "2Z999AA10123456784", "2Z999AA10123456784", false},
		{"only a 1Z in the middle", "9999AA1Z123456784", "9999AA1Z123456784", false},
		{"punctuation", "1Z999AA1012345678!", "1Z999AA1012345678!", false},
		{"control character", "1Z999AA1012345678\n4", "1Z999AA1012345678\n4", false},
		{"cyrillic letter", "1Z999\u0410A10123456784", "1Z999\u0410A10123456784", false},
		{"dotless i upper-cases to an ASCII letter", "1Z999\u0131A10123456784", "1Z999IA10123456784", true},
		{"long s upper-cases to an ASCII letter", "1Z999\u017fA10123456784", "1Z999SA10123456784", true},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := a.NormalizeNumber(tt.in)
			if ok != tt.ok || got != tt.expect {
				t.Errorf("got (%q, %v), expected (%q, %v)", got, ok, tt.expect, tt.ok)
			}
			if ok && !regexp.MustCompile(`^[0-9A-Z]{18}$`).MatchString(got) {
				t.Errorf("an accepted number must be 18 ASCII letters or digits: %q", got)
			}
		})
	}
}

func Test_UPS_Tool(t *testing.T) {
	a := NewUPS()
	if a.Carrier() != "ups" || a.MaxBody() != 0 {
		t.Errorf("carrier %q max body %d", a.Carrier(), a.MaxBody())
	}
	def := a.Tool()
	if def.Name != "track_ups" || !strings.Contains(def.Description, "1Z") || !strings.Contains(def.Description, "Read-only") {
		t.Errorf("tool = %+v", def)
	}
	var schema struct {
		Type       string `json:"type"`
		Required   []string
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Type != "object" || !reflect.DeepEqual(schema.Required, []string{"tracking_number"}) || schema.Properties["tracking_number"].Type != "string" {
		t.Errorf("schema = %s", def.InputSchema)
	}
}

// ---- mapping

func Test_mapUPS_status(t *testing.T) {
	del := `,"deliveryDate":[{"date":"20260102","type":"DEL"}]`
	tests := []struct {
		name       string
		statusType string
		extra      string
		activities []string
		expect     Status
	}{
		{"X is an exception", "X", "", nil, StatusException},
		{"lower case and spaces are tolerated", " x ", "", nil, StatusException},
		{"I is in transit", "I", "", nil, StatusInTransit},
		{"M is a created label", "M", "", nil, StatusLabelCreated},
		{"RS is returned", "RS", "", nil, StatusReturned},
		{"D with a DEL date is delivered", "D", del, nil, StatusDelivered},
		{"D without a DEL date is unknown", "D", "", nil, StatusUnknown},
		{"D with only an SDD date is unknown", "D", `,"deliveryDate":[{"date":"20260102","type":"SDD"}]`, nil, StatusUnknown},
		{"I stays in transit with a DEL date", "I", del, nil, StatusInTransit},
		{"an unknown letter", "Z", "", nil, StatusUnknown},
		{"an unknown letter with a DEL date is not delivered", "Z", del, nil, StatusUnknown},
		{"out for delivery text is not a state", "I", "", []string{actJSON(upsActivityDay, "101500", "", "", "", "Out For Delivery Today", "I")}, StatusInTransit},
		{"no current type: the newest activity decides", "", "", []string{actJSON(upsActivityDay, "101500", "", "", "", "Exception", "X"), actJSON(upsActivityDay, "091500", "", "", "", "Scan", "I")}, StatusException},
		{"no current type and the newest activity is empty", "", "", []string{`{}`, actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I")}, StatusInTransit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := pkgJSON(upsNumber, tt.statusType, tt.extra, tt.activities...)
			if len(tt.activities) == 0 {
				pkg = pkgJSON(upsNumber, tt.statusType, tt.extra, actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I"))
			}
			if res := mustMap(t, answerJSON(pkg)); res.Status != tt.expect {
				t.Errorf("status = %s, expected %s", res.Status, tt.expect)
			}
		})
	}
}

func Test_mapUPS_fields(t *testing.T) {
	body := answerJSON(`{"trackingNumber":"` + upsNumber + `","currentStatus":{"code":"KB","description":"Departed from facility","simplifiedTextDescription":"On the way","type":"I"},"statusCode":"005","statusDescription":"Package status","service":{"description":"UPS Ground"},
	"activity":[{"date":"20260102","time":"101500","status":{"description":"","simplifiedTextDescription":"Simple scan","type":"I"},"location":{"address":{"city":"Atlanta","stateProvince":"GA","country":"","countryCode":"CA"}}},
	{"date":"20260101","time":"081500","status":{"description":"Origin scan","type":"I"},"location":{"address":{"city":"","stateProvince":"","country":"","countryCode":""}}}]}`)
	res := mustMap(t, body)
	if res.StatusText != "Departed from facility" || strVal(res.CarrierStatusCode) != "KB" || strVal(res.Service) != "UPS Ground" {
		t.Errorf("fields = %q %s %s", res.StatusText, strVal(res.CarrierStatusCode), strVal(res.Service))
	}
	if len(res.Events) != 2 || res.Events[0].Description != "Simple scan" || res.Events[1].Description != "Origin scan" {
		t.Fatalf("events = %+v", res.Events)
	}
	if res.Events[0].Location == nil || res.Events[0].Location.Text != "Atlanta, GA, CA" {
		t.Errorf("location = %+v", res.Events[0].Location)
	}
	if res.Events[1].Location != nil {
		t.Errorf("an empty address has no location: %+v", res.Events[1].Location)
	}

	// fallbacks: the package status description, then the simplified text; the package status code.
	res = mustMap(t, answerJSON(`{"trackingNumber":"`+upsNumber+`","currentStatus":{"type":"I"},"statusCode":"005","statusDescription":"Package status","activity":[{"date":"20260102","status":{"description":"x"}}]}`))
	if res.StatusText != "Package status" || strVal(res.CarrierStatusCode) != "005" || res.Service != nil {
		t.Errorf("fallbacks = %q %s %s", res.StatusText, strVal(res.CarrierStatusCode), strVal(res.Service))
	}
	res = mustMap(t, answerJSON(`{"trackingNumber":"`+upsNumber+`","currentStatus":{"type":"I","simplifiedTextDescription":"Only simple"}}`))
	if res.StatusText != "Only simple" {
		t.Errorf("status text = %q", res.StatusText)
	}
}

func Test_mapUPS_events(t *testing.T) {
	act := func(i int) string {
		return actJSON(upsActivityDay, fmt.Sprintf("1%05d", i), upsActivityDay, fmt.Sprintf("2%05d", i), "-05:00", fmt.Sprintf("Scan %d", i), "I")
	}
	build := func(n int) *Result {
		acts := make([]string, 0, n)
		for i := 0; i < n; i++ {
			acts = append(acts, act(i))
		}
		res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", acts...)))
		res.finalize("ups", upsNumber, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		return res
	}

	if res := build(8); len(res.Events) != 8 || res.EventsTruncated {
		t.Errorf("8 activities: %d events, truncated %v", len(res.Events), res.EventsTruncated)
	}
	if res := build(9); len(res.Events) != 8 || !res.EventsTruncated {
		t.Errorf("9 activities: %d events, truncated %v", len(res.Events), res.EventsTruncated)
	}
	if res := build(30); len(res.Events) != 8 || !res.EventsTruncated || res.Events[0].Description != "Scan 0" {
		t.Errorf("30 activities: %d events, truncated %v", len(res.Events), res.EventsTruncated)
	}

	// the mapping keeps one more than the result shows, so that the cut can be seen.
	many := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		many = append(many, act(i))
	}
	if res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", many...))); len(res.Events) != 9 {
		t.Errorf("12 activities: %d events before the cut, expected 9", len(res.Events))
	}

	// empty activities and nulls are skipped and cannot become the last event.
	res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", `{}`, `null`, `{"location":{"address":{"city":"Atlanta"}}}`, act(5))))
	res.finalize("ups", upsNumber, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(res.Events) != 1 || res.Events[0].Description != "Scan 5" || res.LastEvent == nil || res.LastEvent.Description != "Scan 5" {
		t.Errorf("events = %+v last = %+v", res.Events, res.LastEvent)
	}
}

func Test_mapUPS_eventTime(t *testing.T) {
	tests := []struct {
		name                               string
		date, tm, gmtDate, gmtTime, offset string
		expectTime, expectLocal            string
	}{
		{"consistent, negative offset", "20260102", "101500", "20260102", "151500", "-05:00", "2026-01-02T15:15:00Z", "<nil>"},
		{"consistent, positive offset", "20260102", "101500", "20260102", "044500", "+05:30", "2026-01-02T04:45:00Z", "<nil>"},
		{"consistent across midnight", "20260102", "233000", "20260103", "043000", "-05:00", "2026-01-03T04:30:00Z", "<nil>"},
		{"the date part is compared as well", "20260102", "233000", "20260102", "043000", "-05:00", "<nil>", "2026-01-02T23:30:00"},
		{"flipped sign", "20260102", "101500", "20260102", "151500", "+05:00", "<nil>", "2026-01-02T10:15:00"},
		{"wrong gmt value", "20260102", "101500", "20260102", "161500", "-05:00", "<nil>", "2026-01-02T10:15:00"},
		{"five digit gmt time is padded", "20260102", "094500", "20260102", "44500", "+05:00", "2026-01-02T04:45:00Z", "<nil>"},
		{"seven digit gmt time is invalid", "20260102", "101500", "20260102", "1151500", "-05:00", "<nil>", "2026-01-02T10:15:00"},
		{"gmt time with colons is invalid", "20260102", "101500", "20260102", "15:15:00", "-05:00", "<nil>", "2026-01-02T10:15:00"},
		{"offset of 14 hours is the largest", "20260102", "101500", "20260101", "201500", "+14:00", "2026-01-01T20:15:00Z", "<nil>"},
		{"only the hours are bounded", "20260102", "101500", "20260101", "201400", "+14:01", "2026-01-01T20:14:00Z", "<nil>"},
		{"offset of 15 hours", "20260102", "101500", "20260101", "191500", "+15:00", "<nil>", "2026-01-02T10:15:00"},
		{"offset with minutes 59", "20260102", "101500", "20260102", "161400", "-05:59", "2026-01-02T16:14:00Z", "<nil>"},
		{"offset with minutes 60", "20260102", "101500", "20260102", "161500", "-05:60", "<nil>", "2026-01-02T10:15:00"},
		{"text before the offset", "20260102", "101500", "20260102", "151500", "x-05:00", "<nil>", "2026-01-02T10:15:00"},
		{"text after the offset", "20260102", "101500", "20260102", "151500", "-05:00x", "<nil>", "2026-01-02T10:15:00"},
		{"empty gmt time is not midnight", "20260102", "050000", "20260102", "", "+05:00", "<nil>", "2026-01-02T05:00:00"},
		{"offset without a colon", "20260102", "101500", "20260102", "151500", "-0500", "<nil>", "2026-01-02T10:15:00"},
		{"no offset", "20260102", "101500", "20260102", "151500", "", "<nil>", "2026-01-02T10:15:00"},
		{"no offset and a gmt time equal to the local time", "20260102", "101500", "20260102", "101500", "", "<nil>", "2026-01-02T10:15:00"},
		{"offset without a colon and a gmt time equal to the local time", "20260102", "101500", "20260102", "101500", "-0500", "<nil>", "2026-01-02T10:15:00"},
		{"offset over the limit and a gmt time equal to the local time", "20260102", "101500", "20260102", "101500", "+15:00", "<nil>", "2026-01-02T10:15:00"},
		{"offset of 60 minutes and a gmt time equal to the local time", "20260102", "101500", "20260102", "101500", "-05:60", "<nil>", "2026-01-02T10:15:00"},
		{"an offset of zero is valid", "20260102", "101500", "20260102", "101500", "+00:00", "2026-01-02T10:15:00Z", "<nil>"},
		{"no gmt part", "20260102", "101500", "", "", "-05:00", "<nil>", "2026-01-02T10:15:00"},
		{"missing time keeps the local date", "20260102", "", "20260102", "151500", "-05:00", "<nil>", "2026-01-02"},
		{"invalid time keeps the local date", "20260102", "2515", "20260102", "151500", "-05:00", "<nil>", "2026-01-02"},
		{"invalid calendar date reports nothing", "20260231", "101500", "20260102", "151500", "-05:00", "<nil>", "<nil>"},
		{"short date reports nothing", "2026102", "101500", "", "", "", "<nil>", "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", actJSON(tt.date, tt.tm, tt.gmtDate, tt.gmtTime, tt.offset, "Scan", "I"))))
			if len(res.Events) != 1 {
				t.Fatalf("events = %+v", res.Events)
			}
			e := res.Events[0]
			if strVal(e.Time) != tt.expectTime || strVal(e.TimeLocal) != tt.expectLocal {
				t.Errorf("time = %s local = %s, expected %s and %s", strVal(e.Time), strVal(e.TimeLocal), tt.expectTime, tt.expectLocal)
			}
			if res.TimeZoneKnown != (tt.expectTime != "<nil>") {
				t.Errorf("time_zone_known = %v", res.TimeZoneKnown)
			}
		})
	}

	// a number where a text is expected is read as its text; any other type is an empty text.
	numeric := `{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":"I","description":"s"},"activity":[{"date":20260102,"time":101500,"gmtDate":20260102,"gmtTime":151500,"gmtOffset":"-05:00","status":{"description":"Scan"}}]}`
	if e := mustMap(t, answerJSON(numeric)).Events[0]; strVal(e.Time) != "2026-01-02T15:15:00Z" {
		t.Errorf("numbers as texts: %s %s", strVal(e.Time), strVal(e.TimeLocal))
	}
	wrongType := `{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":"I","description":"s"},"activity":[{"date":"20260102","time":"101500","gmtDate":"20260102","gmtTime":{"a":1},"gmtOffset":"-05:00","status":{"description":"Scan"}}]}`
	if e := mustMap(t, answerJSON(wrongType)).Events[0]; e.Time != nil || strVal(e.TimeLocal) != "2026-01-02T10:15:00" {
		t.Errorf("a wrong type in one time field: %s %s", strVal(e.Time), strVal(e.TimeLocal))
	}
}

func Test_mapUPS_activityFilter(t *testing.T) {
	dateOnly := `{"date":"20260102","time":"101500"}`
	textOnly := `{"status":{"description":"Only text"}}`
	simpleOnly := `{"status":{"simplifiedTextDescription":"Only simple text"}}`
	res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", dateOnly, textOnly, simpleOnly)))
	if len(res.Events) != 3 {
		t.Fatalf("an activity with a date or a text is kept: %+v", res.Events)
	}
	if res.Events[0].Description != "" || strVal(res.Events[0].TimeLocal) != "2026-01-02T10:15:00" {
		t.Errorf("date only: %+v", res.Events[0])
	}
	if res.Events[1].Description != "Only text" || res.Events[2].Description != "Only simple text" {
		t.Errorf("text only: %+v %+v", res.Events[1], res.Events[2])
	}
}

func Test_mapUPS_fieldPriority(t *testing.T) {
	body := answerJSON(`{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":" i "},"deliveryDate":[{"date":"20260104","type":" sdd "},{"date":"20260105","type":" rdd "}],
	"activity":[{"date":"20260102","status":{"description":"Primary","simplifiedTextDescription":"Secondary"},"location":{"address":{"city":"C","country":"Country","countryCode":"CC"}}}]}`)
	res := mustMap(t, body)
	if res.Events[0].Description != "Primary" || res.Events[0].Location == nil || res.Events[0].Location.Text != "C, Country" {
		t.Errorf("priorities: %+v", res.Events[0])
	}
	if res.Status != StatusInTransit || strVal(res.EstimatedDelivery.Date) != "2026-01-05" {
		t.Errorf("types are compared without case and spaces: %s %s", res.Status, strVal(res.EstimatedDelivery.Date))
	}

	// a delivery date type in another case still counts as delivered
	del := answerJSON(`{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":"d"},"deliveryDate":[{"date":"20260102","type":" del "}],"activity":[{"date":"20260102","status":{"description":"x","type":" d "}}]}`)
	if res := mustMap(t, del); res.Status != StatusDelivered {
		t.Errorf("status = %s", res.Status)
	}
}

func Test_mapUPS_scalarTypes(t *testing.T) {
	tests := []struct {
		name, raw, expect string
	}{
		{"string", `"UPS Ground"`, "UPS Ground"},
		{"number", `1.5`, "1.5"},
		{"zero", `0`, "0"},
		{"digit nine", `9`, "9"},
		{"negative number", `-3`, "-3"},
		{"boolean", `true`, "<nil>"},
		{"null", `null`, "<nil>"},
		{"array", `["a"]`, "<nil>"},
		{"object", `{"a":1}`, "<nil>"},
		{"empty string", `""`, "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := answerJSON(`{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":"I","description":"s"},"service":{"description":` + tt.raw + `}}`)
			if got := strVal(mustMap(t, body).Service); got != tt.expect {
				t.Errorf("service = %s, expected %s", got, tt.expect)
			}
		})
	}
}

func Test_mapUPS_emptyPackage(t *testing.T) {
	tests := []struct {
		name  string
		pkg   string
		empty bool
	}{
		{"nothing", `{"trackingNumber":"` + upsNumber + `"}`, true},
		{"a type only", `{"trackingNumber":"` + upsNumber + `","currentStatus":{"type":"I"}}`, false},
		{"a text only", `{"trackingNumber":"` + upsNumber + `","currentStatus":{"description":"x"}}`, false},
		{"an activity only", `{"trackingNumber":"` + upsNumber + `","activity":[{"date":"20260102"}]}`, false},
		{"the package status text only", `{"trackingNumber":"` + upsNumber + `","statusDescription":"x"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, cerr := mapUPS([]byte(answerJSON(tt.pkg)), upsNumber)
			if tt.empty != (cerr != nil) || tt.empty == (res != nil) {
				t.Errorf("got %+v %+v", res, cerr)
			}
		})
	}
}

func Test_mapUPS_timeZoneKnownFollowsTheNewestEvent(t *testing.T) {
	known := actJSON("20260102", "101500", "20260102", "151500", "-05:00", "New", "I")
	unknown := actJSON("20260101", "101500", "", "", "", "Old", "I")
	if res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", known, unknown))); !res.TimeZoneKnown {
		t.Errorf("the newest event is known")
	}
	if res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", unknown, known))); res.TimeZoneKnown {
		t.Errorf("the newest event is not known")
	}
}

func Test_mapUPS_deliveredAt(t *testing.T) {
	del := `,"deliveryDate":[{"date":"20260102","type":"DEL"}]`
	knownD := actJSON("20260102", "101500", "20260102", "151500", "-05:00", "Delivered", "D")
	unknownD := actJSON("20260102", "101500", "", "", "", "Delivered", "D")
	transit := actJSON("20260101", "081500", "20260101", "131500", "-05:00", "Departed", "I")
	tests := []struct {
		name   string
		status string
		extra  string
		acts   []string
		expect string
	}{
		{"delivered with a known D activity", "D", del, []string{knownD, transit}, "2026-01-02T15:15:00Z"},
		{"the D activity is not the first", "D", del, []string{transit, knownD}, "2026-01-02T15:15:00Z"},
		{"only the newest D activity counts", "D", del, []string{unknownD, knownD}, "<nil>"},
		{"the D activity has no known zone", "D", del, []string{unknownD}, "<nil>"},
		{"delivered without any D activity", "D", del, []string{transit}, "<nil>"},
		{"not delivered: no delivered_at", "D", "", []string{knownD}, "<nil>"},
		{"in transit: no delivered_at", "I", del, []string{knownD}, "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mustMap(t, answerJSON(pkgJSON(upsNumber, tt.status, tt.extra, tt.acts...)))
			if strVal(res.DeliveredAt) != tt.expect {
				t.Errorf("delivered_at = %s, expected %s", strVal(res.DeliveredAt), tt.expect)
			}
		})
	}
}

func Test_mapUPS_estimatedDelivery(t *testing.T) {
	act := actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I")
	tests := []struct {
		name   string
		status string
		extra  string
		expect string
	}{
		{"RDD wins over SDD", "I", `,"deliveryDate":[{"date":"20260104","type":"SDD"},{"date":"20260105","type":"RDD"}]`, "2026-01-05"},
		{"RDD wins whatever the order", "I", `,"deliveryDate":[{"date":"20260105","type":"RDD"},{"date":"20260104","type":"SDD"}]`, "2026-01-05"},
		{"an invalid RDD falls back to SDD", "I", `,"deliveryDate":[{"date":"20260230","type":"RDD"},{"date":"20260104","type":"SDD"}]`, "2026-01-04"},
		{"only SDD", "I", `,"deliveryDate":[{"date":"20260104","type":"SDD"}]`, "2026-01-04"},
		{"a short date is invalid", "I", `,"deliveryDate":[{"date":"2026014","type":"SDD"}]`, "<nil>"},
		{"no dates", "I", "", "<nil>"},
		{"an exception keeps the date", "X", `,"deliveryDate":[{"date":"20260104","type":"SDD"}]`, "2026-01-04"},
		{"a created label keeps the date", "M", `,"deliveryDate":[{"date":"20260104","type":"SDD"}]`, "2026-01-04"},
		{"a null entry", "I", `,"deliveryDate":[null,{"date":"20260104","type":"SDD"}]`, "2026-01-04"},
		{"not for a delivered package", "D", `,"deliveryDate":[{"date":"20260104","type":"SDD"},{"date":"20260102","type":"DEL"}]`, "<nil>"},
		{"not when a DEL date exists", "I", `,"deliveryDate":[{"date":"20260104","type":"SDD"},{"date":"20260102","type":"DEL"}]`, "<nil>"},
		{"not for a returned package", "RS", `,"deliveryDate":[{"date":"20260104","type":"SDD"}]`, "<nil>"},
		{"the window of deliveryTime is not forwarded", "I", `,"deliveryDate":[{"date":"20260104","type":"SDD"}],"deliveryTime":{"startTime":"090000","endTime":"120000","type":"EDW"}`, "2026-01-04"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mustMap(t, answerJSON(pkgJSON(upsNumber, tt.status, tt.extra, act)))
			if strVal(res.EstimatedDelivery.Date) != tt.expect || res.EstimatedDelivery.From != nil || res.EstimatedDelivery.To != nil {
				t.Errorf("estimate = %+v, expected %s", res.EstimatedDelivery, tt.expect)
			}
		})
	}
}

func Test_mapUPS_pickPackage(t *testing.T) {
	act := actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I")
	pkg := func(number, text string) string {
		return fmt.Sprintf(`{"trackingNumber":%q,"currentStatus":{"type":"I","description":%q},"activity":[%s]}`, number, text, act)
	}
	shipment := func(inquiry string, packages ...string) string {
		return fmt.Sprintf(`{"inquiryNumber":%q,"package":[%s]}`, inquiry, strings.Join(packages, ","))
	}
	wrap := func(shipments ...string) string {
		return `{"trackResponse":{"shipment":[` + strings.Join(shipments, ",") + `]}}`
	}
	other := "1Z999AA10123456785"

	tests := []struct {
		name   string
		body   string
		expect string // status text of the picked package; empty: none
	}{
		{"the package number matches", wrap(shipment(upsNumber, pkg(other, "other"), pkg(upsNumber, "mine"))), "mine"},
		{"the package number matches in a later shipment", wrap(shipment(other, pkg(other, "other")), shipment(upsNumber, pkg(upsNumber, "mine"))), "mine"},
		{"the match is made on the cleaned package number", wrap(shipment(upsNumber, pkg(other, "other"), pkg("1z999aa1-0123456784", "mine"))), "mine"},
		{"the match is made on the cleaned inquiry number", wrap(shipment(" 1z999aa1-0123456784 ", pkg(other, "only"))), "only"},
		{"the inquiry number matches and there is one package", wrap(shipment(upsNumber, pkg(other, "only"))), "only"},
		{"the inquiry number matches and there are two packages", wrap(shipment(upsNumber, pkg(other, "a"), pkg("1Z999AA10123456786", "b"))), ""},
		{"only other numbers", wrap(shipment(other, pkg(other, "other"))), ""},
		{"two shipments echo the same inquiry number", wrap(shipment(upsNumber, pkg(other, "a")), shipment(upsNumber, pkg("1Z999AA10123456786", "b"))), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, cerr := mapUPS([]byte(tt.body), upsNumber)
			if tt.expect == "" {
				if res != nil || cerr == nil || cerr.Code != CodeCarrierUnavailable {
					t.Fatalf("expected no package, got %+v %+v", res, cerr)
				}
				return
			}
			if cerr != nil || res.StatusText != tt.expect {
				t.Errorf("got %+v %+v, expected %s", res, cerr, tt.expect)
			}
		})
	}
}

func Test_mapUPS_noResult(t *testing.T) {
	warn := `{"code":"TW0001","message":"SECRET-WARNING-TEXT"}`
	pkg := pkgJSON(upsNumber, "I", "", actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I"))
	tests := []struct {
		name   string
		body   string
		expect ErrCode
	}{
		{"warnings and no package", `{"trackResponse":{"shipment":[{"inquiryNumber":"x","warnings":[` + warn + `]}]}}`, CodeNotFound},
		{"warnings in the second of two shipments", `{"trackResponse":{"shipment":[{"inquiryNumber":"x"},{"inquiryNumber":"y","warnings":[` + warn + `]}]}}`, CodeNotFound},
		{"warnings with an empty package list", `{"trackResponse":{"shipment":[{"inquiryNumber":"x","package":[],"warnings":[` + warn + `]}]}}`, CodeNotFound},
		{"no package and no warning", `{"trackResponse":{"shipment":[{"inquiryNumber":"x"}]}}`, CodeCarrierUnavailable},
		{"empty shipment list", `{"trackResponse":{"shipment":[]}}`, CodeCarrierUnavailable},
		{"null shipment list", `{"trackResponse":{"shipment":null}}`, CodeCarrierUnavailable},
		{"null shipment", `{"trackResponse":{"shipment":[null]}}`, CodeCarrierUnavailable},
		{"no track response", `{}`, CodeCarrierUnavailable},
		{"null track response", `{"trackResponse":null}`, CodeCarrierUnavailable},
		{"a package with nothing in it", answerJSON(`{}`), CodeCarrierUnavailable},
		{"a null package", answerJSON(`null`), CodeCarrierUnavailable},
		{"a package with a number only", answerJSON(`{"trackingNumber":"` + upsNumber + `"}`), CodeCarrierUnavailable},
		{"null activity and null delivery date", answerJSON(`{"trackingNumber":"` + upsNumber + `","activity":[null],"deliveryDate":[null]}`), CodeCarrierUnavailable},
		{"an object where the package list is expected", `{"trackResponse":{"shipment":[{"inquiryNumber":"x","package":{"a":1}}]}}`, CodeCarrierUnavailable},
		{"an object where the warnings list is expected", `{"trackResponse":{"shipment":[{"inquiryNumber":"x","warnings":{"a":1}}]}}`, CodeCarrierUnavailable},
		{"a string where an object is expected", `{"trackResponse":"x"}`, CodeCarrierUnavailable},
		{"not json", `<html>`, CodeCarrierUnavailable},
		{"empty body", ``, CodeCarrierUnavailable},
		{"a json null", `null`, CodeCarrierUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { expectMapError(t, tt.body, tt.expect) })
	}

	// a result with wrong scalar types is still a result, with the bad fields empty.
	weird := `{"trackResponse":{"shipment":[{"inquiryNumber":"1Z999AA10123456784","package":[{"trackingNumber":true,"currentStatus":{"type":["I"],"description":{"a":1},"code":null},"service":{"description":1.5},"activity":[{"date":null,"status":{"description":"Scan"}}]}]}]}}`
	res, cerr := mapUPS([]byte(weird), upsNumber)
	if cerr != nil || res == nil {
		t.Fatalf("got %+v %+v", res, cerr)
	}
	if res.Status != StatusUnknown || res.StatusText != "" || strVal(res.Service) != "1.5" || len(res.Events) != 1 {
		t.Errorf("weird scalars: %+v", res)
	}
	_ = pkg
}

func Test_mapUPS_allowList(t *testing.T) {
	sentinels := []string{"SENTINEL-NAME", "SENTINEL-RECV", "SENTINEL-DROP", "SENTINEL-STREET", "SENTINEL-POSTAL", "SENTINEL-SLIC", "SENTINEL-SIGNATURE", "SENTINEL-PAYMENT", "SENTINEL-REF", "SENTINEL-ALT", "SENTINEL-ACCESS", "SENTINEL-WARNING", "SENTINEL-PHOTO"}
	body := fmt.Sprintf(`{"trackResponse":{"shipment":[{"inquiryNumber":%q,"warnings":[{"code":"W1","message":"SENTINEL-WARNING"}],"package":[{
		"trackingNumber":%q,
		"currentStatus":{"code":"KB","description":"Departed","type":"I"},
		"packageAddress":[{"type":"DEST","name":"SENTINEL-NAME","attentionName":"SENTINEL-NAME","address":{"addressLine1":"SENTINEL-STREET","postalCode":"SENTINEL-POSTAL","city":"Atlanta"}}],
		"deliveryInformation":{"receivedBy":"SENTINEL-RECV","location":"SENTINEL-DROP","signature":{"image":"SENTINEL-SIGNATURE"},"deliveryPhoto":{"photo":"SENTINEL-PHOTO"}},
		"paymentInformation":[{"amount":"1","id":"SENTINEL-PAYMENT"}],
		"referenceNumber":[{"number":"SENTINEL-REF","type":"SHIPMENT"}],
		"alternateTrackingNumber":[{"number":"SENTINEL-ALT"}],
		"accessPointInformation":{"pickupByDate":"SENTINEL-ACCESS"},
		"activity":[{"date":"20260102","time":"101500","location":{"slic":"SENTINEL-SLIC","address":{"addressLine1":"SENTINEL-STREET","addressLine2":"SENTINEL-STREET","postalCode":"SENTINEL-POSTAL","city":"Atlanta","stateProvince":"GA","country":"US"}},"status":{"description":"Departed","type":"I"}}]
	}]}]}}`, upsNumber, upsNumber)

	res := mustMap(t, body)
	res.finalize("ups", upsNumber, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	text, err := res.marshal()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sentinels {
		if strings.Contains(text, s) {
			t.Errorf("%s reached the result: %s", s, text)
		}
	}
	if !strings.Contains(text, "Atlanta, GA, US") {
		t.Errorf("the city, state and country must be in the result: %s", text)
	}
}

func Test_mapUPS_freeTextIsMasked(t *testing.T) {
	act := actJSON(upsActivityDay, "101500", "", "", "", "Call +1 404 555 0100 or mail someone@example.com", "I")
	res := mustMap(t, answerJSON(pkgJSON(upsNumber, "I", "", act)))
	res.finalize("ups", upsNumber, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if strings.Contains(res.Events[0].Description, "someone@example.com") || strings.Contains(res.Events[0].Description, "555") {
		t.Errorf("not masked: %q", res.Events[0].Description)
	}
}

// ---- requests and answers of the carrier

func Test_UPS_requestShape(t *testing.T) {
	stub := newUPSStub(t)
	d := stub.deps(newFakeClock())
	res, cerr := stub.adapter().Track(context.Background(), d, testCredential(), upsNumber)
	if cerr != nil || res == nil {
		t.Fatalf("got %+v %+v", res, cerr)
	}

	tr := stub.tokenReq
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("test-client-id:test-client-secret"))
	if tr.method != "POST" || tr.path != upsTokenStub || tr.auth != wantAuth || !strings.Contains(tr.body, "grant_type=client_credentials") || strings.Contains(tr.body, "test-client") {
		t.Errorf("token request = %+v", tr)
	}

	q := stub.trackReq
	if q.method != "GET" || q.path != upsTrackStub+upsNumber || q.auth != "Bearer "+upsToken || q.transSrc != "voipbin" || q.accept != "application/json" {
		t.Errorf("track request = %+v", q)
	}
	values, err := url.ParseQuery(q.query)
	if err != nil || len(values) != 4 || values.Get("locale") != "en_US" || values.Get("returnSignature") != "false" || values.Get("returnMilestones") != "false" || values.Get("returnPOD") != "false" {
		t.Errorf("query = %q", q.query)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(q.transID) {
		t.Errorf("transId = %q", q.transID)
	}
	for _, secret := range []string{"test-client-id", "test-client-secret", upsToken} {
		if strings.Contains(q.path+"?"+q.query, secret) {
			t.Errorf("%s is in the url", secret)
		}
	}

	if _, cerr := stub.adapter().Track(context.Background(), d, testCredential(), upsNumber); cerr != nil {
		t.Fatal(cerr)
	}
	if first := q.transID; stub.trackReq.transID == first {
		t.Errorf("transId must differ between requests")
	}
	if stub.tokenCalls.Load() != 1 {
		t.Errorf("the quoted expires_in must let the token be reused: %d exchanges", stub.tokenCalls.Load())
	}
}

func Test_UPS_trackStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		expectCode ErrCode
		cooldown   time.Duration
		dropToken  bool
	}{
		{"400 is a bad number", 400, `{"response":{"errors":[{"code":"1","message":"SECRET-TEXT"}]}}`, CodeInvalidFormat, 0, false},
		{"401 drops the token", 401, `{}`, CodeCarrierUnavailable, 0, true},
		{"403 is a short block", 403, `{}`, CodeCarrierUnavailable, time.Minute, false},
		{"404 is not found", 404, `{}`, CodeNotFound, 0, false},
		{"429 is a limit", 429, `{}`, CodeRateLimited, 0, false},
		{"500", 500, `{}`, CodeCarrierUnavailable, 0, false},
		{"201 is not an answer", 201, answerFor(upsNumber), CodeCarrierUnavailable, 0, false},
		{"202 is not an answer", 202, answerFor(upsNumber), CodeCarrierUnavailable, 0, false},
		{"203 is not an answer", 203, answerFor(upsNumber), CodeCarrierUnavailable, 0, false},
		{"502", 502, `{}`, CodeCarrierUnavailable, 0, false},
		{"503", 503, `{}`, CodeCarrierUnavailable, 0, false},
		{"a redirect", 302, ``, CodeCarrierUnavailable, 0, false},
		{"another 4xx", 418, `{}`, CodeCarrierUnavailable, 0, false},
		{"200 that is not json", 200, `<html>`, CodeCarrierUnavailable, 0, false},
		{"200 with nothing in it", 200, `{}`, CodeCarrierUnavailable, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUPSStub(t)
			stub.set(func(s *upsStub) { s.trackStatus, s.trackBody = tt.status, tt.body })
			res, cerr := stub.adapter().Track(context.Background(), stub.deps(newFakeClock()), testCredential(), upsNumber)
			if res != nil || cerr == nil || cerr.Code != tt.expectCode || cerr.Cooldown != tt.cooldown || cerr.DropToken != tt.dropToken {
				t.Errorf("got %+v %+v", res, cerr)
			}
		})
	}
}

func Test_UPS_trackTransportAndSize(t *testing.T) {
	stub := newUPSStub(t)

	d := stub.deps(newFakeClock())
	d.MaxBody = 100
	stub.set(func(s *upsStub) {
		s.trackBody = answerJSON(pkgJSON(upsNumber, "I", "", actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I")))
	})
	if res, cerr := stub.adapter().Track(context.Background(), d, testCredential(), upsNumber); res != nil || cerr == nil || cerr.Code != CodeCarrierUnavailable {
		t.Errorf("a body over the limit: %+v %+v", res, cerr)
	}

	d = stub.deps(newFakeClock())
	d.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("SECRET-TRANSPORT-TEXT") })}
	if res, cerr := stub.adapter().Track(context.Background(), d, testCredential(), upsNumber); res != nil || cerr == nil || cerr.Code != CodeCarrierUnavailable {
		t.Errorf("a transport error: %+v %+v", res, cerr)
	}
}

func Test_UPS_redirectsAreRefused(t *testing.T) {
	for name, client := range map[string]func(*upsStub) *http.Client{
		"leave the redirect response": func(s *upsStub) *http.Client { return noRedirectClient(s.srv) },
		"production policy": func(s *upsStub) *http.Client {
			c := s.srv.Client()
			c.CheckRedirect = mcpserverhandler.RefuseRedirects
			return c
		},
	} {
		t.Run(name, func(t *testing.T) {
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Errorf("the redirect target must not be called") }))
			defer target.Close()

			stub := newUPSStub(t)
			stub.set(func(s *upsStub) { s.trackStatus = http.StatusFound })
			srv2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, upsTokenStub) {
					_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
					return
				}
				w.Header().Set("Location", target.URL+"/x")
				w.WriteHeader(http.StatusFound)
			}))
			defer srv2.Close()

			d := stub.deps(newFakeClock())
			d.Client = client(&upsStub{srv: srv2})
			a := newUPSAdapter(srv2.URL+upsTrackStub, srv2.URL+upsTokenStub)
			if res, cerr := a.Track(context.Background(), d, testCredential(), upsNumber); res != nil || cerr == nil || cerr.Code != CodeCarrierUnavailable {
				t.Errorf("got %+v %+v", res, cerr)
			}
		})
	}
}

// ---- through the handler

func Test_UPS_handlerSuccess(t *testing.T) {
	stub := newUPSStub(t)
	h, _ := stub.handler(t)

	text, isError, err := h.Call(context.Background(), makeCall("ups", upsNumber, testSecret))
	if err != nil || isError {
		t.Fatalf("got %q %v %v", text, isError, err)
	}
	r := decode(t, text)
	if r.Carrier != "ups" || r.TrackingNumber != upsNumber || r.Status != StatusInTransit || r.Cached || len(r.Events) != 1 {
		t.Errorf("result = %+v", r)
	}

	// the second call is served from the cache
	text, _, _ = h.Call(context.Background(), makeCall("ups", upsNumber, testSecret))
	if r := decode(t, text); !r.Cached || stub.trackCalls.Load() != 1 {
		t.Errorf("cached = %v, track calls %d", r.Cached, stub.trackCalls.Load())
	}

	// a different number reuses the token
	other := "1Z999AA10123456785"
	if _, isError, err := h.Call(context.Background(), makeCall("ups", other, testSecret)); err != nil || isError {
		t.Fatal(isError, err)
	}
	if stub.tokenCalls.Load() != 1 || stub.trackCalls.Load() != 2 {
		t.Errorf("token exchanges %d, track calls %d", stub.tokenCalls.Load(), stub.trackCalls.Load())
	}

	// a wrong number never reaches the carrier or the key
	called := false
	bad := makeCall("ups", "1Z-short", testSecret, func(c *mcpserver.CarrierCall) {
		c.Secret = func() (string, bool, error) { called = true; return testSecret, true, nil }
	})
	text, isError, err = h.Call(context.Background(), bad)
	expectError(t, text, isError, err, CodeInvalidFormat, "ups")
	if called || stub.trackCalls.Load() != 2 {
		t.Errorf("the key was read or the carrier called for a bad number")
	}
}

func Test_UPS_handlerTokenErrors(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		expectCode     ErrCode
		cooldown       time.Duration
		expectRetryHit bool
	}{
		{"401 is a rejected key for ten minutes", 401, CodeCredentialInvalid, 10 * time.Minute, false},
		{"403 is a block for a minute", 403, CodeCarrierUnavailable, time.Minute, false},
		{"429 is busy without a cooldown", 429, CodeServiceBusy, 0, true},
		{"400 has no cooldown", 400, CodeCarrierUnavailable, 0, true},
		{"500 has no cooldown", 500, CodeCarrierUnavailable, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUPSStub(t)
			stub.set(func(s *upsStub) {
				s.tokenStatus, s.tokenBody = tt.status, `{"response":{"errors":[{"code":"10401","message":"SECRET-TEXT"}]}}`
			})
			h, clk := stub.handler(t)
			call := makeCall("ups", upsNumber, testSecret)

			text, isError, err := h.Call(context.Background(), call)
			expectError(t, text, isError, err, tt.expectCode, "ups")
			if stub.tokenCalls.Load() != 1 || stub.trackCalls.Load() != 0 {
				t.Fatalf("token exchanges %d, track calls %d", stub.tokenCalls.Load(), stub.trackCalls.Load())
			}

			// a second call a little later
			clk.add(30 * time.Second)
			text, isError, err = h.Call(context.Background(), call)
			expectError(t, text, isError, err, tt.expectCode, "ups")
			wantCalls := int32(1)
			if tt.expectRetryHit {
				wantCalls = 2
			} else if tt.cooldown == time.Minute {
				wantCalls = 1
			}
			if stub.tokenCalls.Load() != wantCalls {
				t.Errorf("after 30 seconds: %d exchanges, expected %d", stub.tokenCalls.Load(), wantCalls)
			}

			// and after the cooldown
			if tt.cooldown > 0 {
				clk.add(tt.cooldown)
				_, _, _ = h.Call(context.Background(), call)
				if stub.tokenCalls.Load() != wantCalls+1 {
					t.Errorf("after the cooldown: %d exchanges, expected %d", stub.tokenCalls.Load(), wantCalls+1)
				}
			}
		})
	}
}

func Test_UPS_handlerTrackErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		expectCode ErrCode
		cooldown   time.Duration
		cached     bool
	}{
		{"400 is cached", 400, `{}`, CodeInvalidFormat, 0, true},
		{"404 is cached", 404, `{}`, CodeNotFound, 0, true},
		{"warnings only is cached", 200, `{"trackResponse":{"shipment":[{"inquiryNumber":"x","warnings":[{"code":"W","message":"m"}]}]}}`, CodeNotFound, 0, true},
		{"401 is not cached and has no cooldown", 401, `{}`, CodeCarrierUnavailable, 0, false},
		{"403 blocks the key for a minute", 403, `{}`, CodeCarrierUnavailable, time.Minute, false},
		{"429 limits the key for a minute", 429, `{}`, CodeRateLimited, time.Minute, false},
		{"500 is not cached", 500, `{}`, CodeCarrierUnavailable, 0, false},
		{"a malformed answer is not cached", 200, `<html>`, CodeCarrierUnavailable, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newUPSStub(t)
			stub.set(func(s *upsStub) { s.trackStatus, s.trackBody = tt.status, tt.body })
			h, clk := stub.handler(t)
			call := makeCall("ups", upsNumber, testSecret)

			text, isError, err := h.Call(context.Background(), call)
			expectError(t, text, isError, err, tt.expectCode, "ups")
			if stub.trackCalls.Load() != 1 {
				t.Fatalf("track calls %d", stub.trackCalls.Load())
			}

			// 61 seconds later the one minute cooldowns are over and the 120 second cache is not.
			clk.add(61 * time.Second)
			text, isError, err = h.Call(context.Background(), call)
			expectError(t, text, isError, err, tt.expectCode, "ups")
			wantCalls := int32(2)
			if tt.cached {
				wantCalls = 1
			}
			if stub.trackCalls.Load() != wantCalls {
				t.Errorf("track calls = %d, expected %d", stub.trackCalls.Load(), wantCalls)
			}

			// inside a cooldown nothing is sent
			if tt.cooldown > 0 {
				// the call above ended with a new cooldown; one more inside it must not reach the carrier
				before := stub.trackCalls.Load()
				_, _, _ = h.Call(context.Background(), call)
				if stub.trackCalls.Load() != before {
					t.Errorf("a call inside the cooldown reached the carrier: %d, %d", before, stub.trackCalls.Load())
				}
			}
		})
	}
}

func Test_UPS_handlerTrackCooldownIsTheDocumentedLength(t *testing.T) {
	for _, status := range []int{403, 429} {
		stub := newUPSStub(t)
		stub.set(func(s *upsStub) { s.trackStatus = status })
		h, clk := stub.handler(t)
		call := makeCall("ups", upsNumber, testSecret)

		_, _, _ = h.Call(context.Background(), call)
		clk.add(59 * time.Second)
		_, _, _ = h.Call(context.Background(), call)
		if stub.trackCalls.Load() != 1 {
			t.Errorf("%d: the key must be blocked at 59 seconds", status)
		}
		clk.add(2 * time.Second)
		_, _, _ = h.Call(context.Background(), call)
		if stub.trackCalls.Load() != 2 {
			t.Errorf("%d: the key must be free at 61 seconds", status)
		}
	}
}

func Test_UPS_handlerNegativeAnswersExpireWithTheCache(t *testing.T) {
	stub := newUPSStub(t)
	stub.set(func(s *upsStub) { s.trackStatus = 404 })
	h, clk := stub.handler(t)
	call := makeCall("ups", upsNumber, testSecret)

	_, _, _ = h.Call(context.Background(), call)
	clk.add(119 * time.Second)
	_, _, _ = h.Call(context.Background(), call)
	if stub.trackCalls.Load() != 1 {
		t.Errorf("a not-found answer is kept for 120 seconds: %d lookups", stub.trackCalls.Load())
	}
	clk.add(2 * time.Second)
	_, _, _ = h.Call(context.Background(), call)
	if stub.trackCalls.Load() != 2 {
		t.Errorf("a not-found answer is looked up again after 120 seconds: %d lookups", stub.trackCalls.Load())
	}
}

func Test_UPS_handlerANewKeyGetsANewToken(t *testing.T) {
	stub := newUPSStub(t)
	h, _ := stub.handler(t)

	if _, isError, err := h.Call(context.Background(), makeCall("ups", upsNumber, testSecret)); err != nil || isError {
		t.Fatal(isError, err)
	}
	if _, isError, err := h.Call(context.Background(), makeCall("ups", upsNumber, "OTHER-ID:OTHER-SECRET")); err != nil || isError {
		t.Fatal(isError, err)
	}
	if stub.tokenCalls.Load() != 2 {
		t.Errorf("a re-entered key must not reuse the token of the old one: %d exchanges", stub.tokenCalls.Load())
	}
}

func Test_UPS_handlerTrack401DropsTheTokenWithoutExchangingAgain(t *testing.T) {
	stub := newUPSStub(t)
	stub.set(func(s *upsStub) { s.trackStatus = 401 })
	h, _ := stub.handler(t)
	call := makeCall("ups", upsNumber, testSecret)

	text, isError, err := h.Call(context.Background(), call)
	expectError(t, text, isError, err, CodeCarrierUnavailable, "ups")
	if stub.tokenCalls.Load() != 1 || stub.trackCalls.Load() != 1 {
		t.Fatalf("one call must exchange once and look up once: %d %d", stub.tokenCalls.Load(), stub.trackCalls.Load())
	}

	_, _, _ = h.Call(context.Background(), call)
	if stub.tokenCalls.Load() != 2 || stub.trackCalls.Load() != 2 {
		t.Errorf("the next call exchanges again: %d %d", stub.tokenCalls.Load(), stub.trackCalls.Load())
	}
}

func Test_UPS_handlerSecretsNeverAppear(t *testing.T) {
	stub := newUPSStub(t)
	h, _ := stub.handler(t)
	logs := logtest.New(t)

	var outputs []string
	for _, status := range []int{200, 401, 403, 404, 429, 500} {
		stub.set(func(s *upsStub) { s.trackStatus = status })
		for _, number := range []string{upsNumber, "1Z999AA10123456785"} {
			text, _, err := h.Call(context.Background(), makeCall("ups", number, testSecret))
			if err != nil {
				outputs = append(outputs, err.Error())
			}
			outputs = append(outputs, text)
		}
	}
	// a token exchange that fails
	stub.set(func(s *upsStub) { s.tokenStatus = 401; s.trackStatus = 200 })
	h.tokens.Delete(h.key("token", customerA.String(), "ups", testSecret))
	text, _, _ := h.Call(context.Background(), makeCall("ups", "1Z999AA10123456787", testSecret))
	outputs = append(outputs, text)

	for _, o := range outputs {
		for _, secret := range []string{testSecretID, testSecretValue, upsToken} {
			if strings.Contains(o, secret) {
				t.Errorf("%s leaked into %q", secret, o)
			}
		}
	}
	logs.AssertAbsent(t, testSecret, testSecretValue, testSecretID, upsToken, customerA.String(), base64.StdEncoding.EncodeToString([]byte(testSecret)))
	logs.AssertField(t, "carrier")

	for _, key := range h.debugKeys("ups") {
		for _, secret := range []string{testSecretValue, testSecretID, upsToken, customerA.String()} {
			if strings.Contains(key, secret) {
				t.Errorf("a cache key holds %s: %q", secret, key)
			}
		}
	}

	// where the key and the token went on the wire
	stub.set(func(s *upsStub) { s.tokenStatus = 200; s.trackStatus = 200 })
	h2, _ := stub.handler(t)
	if _, isError, err := h2.Call(context.Background(), makeCall("ups", "1Z999AA10123456788", testSecret)); err != nil || isError {
		t.Fatal(isError, err)
	}
	wantBasic := "Basic " + base64.StdEncoding.EncodeToString([]byte(testSecret))
	tr, q := stub.tokenReq, stub.trackReq
	if tr.auth != wantBasic || q.auth != "Bearer "+upsToken {
		t.Errorf("the key is in the Basic header only and the token in the bearer header only: %q %q", tr.auth, q.auth)
	}
	for _, part := range []string{tr.path, tr.query, tr.body, q.path, q.query, q.body} {
		for _, secret := range []string{testSecretID, testSecretValue, upsToken} {
			if strings.Contains(part, secret) {
				t.Errorf("%s is in a url or body: %q", secret, part)
			}
		}
	}
}

// ---- production hosts

func Test_UPS_productionRequestsGoToTheConstantHosts(t *testing.T) {
	var mu sync.Mutex
	var seen []*http.Request
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		seen = append(seen, r)
		mu.Unlock()
		body := `{"access_token":"tok","token_type":"Bearer","expires_in":"3600"}`
		if strings.HasPrefix(r.URL.Path, upsTrackPath) {
			body = answerJSON(pkgJSON(upsNumber, "I", "", actJSON(upsActivityDay, "101500", "", "", "", "Scan", "I")))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
	})

	a := NewUPS()
	clk := newFakeClock()
	d := Deps{Carrier: "ups", Client: &http.Client{Transport: transport}, Now: clk.now, Tokens: newTokenCache(8, clk.now), TokenKey: "k", MaxBody: 1 << 20}
	if res, cerr := a.Track(context.Background(), d, testCredential(), upsNumber); res == nil || cerr != nil {
		t.Fatalf("got %+v %+v", res, cerr)
	}

	if len(seen) != 2 {
		t.Fatalf("expected a token request and a lookup, got %d requests", len(seen))
	}
	for _, r := range seen {
		if r.URL.Scheme != "https" || r.URL.Host != upsHost || r.URL.Port() != "" || r.URL.User != nil {
			t.Errorf("request goes to %s://%s", r.URL.Scheme, r.URL.Host)
		}
	}
	if seen[0].URL.Path != upsTokenPath || seen[1].URL.Path != upsTrackPath+upsNumber {
		t.Errorf("paths = %q %q", seen[0].URL.Path, seen[1].URL.Path)
	}

	// The host is pinned by its hash, so that no host name is written in a test, and the paths by their text.
	sum := sha256.Sum256([]byte(upsHost))
	if hex.EncodeToString(sum[:]) != "29b1701b303f4bcf32595663c2870a6c4ef592774ac0073620d15b991911f78d" {
		t.Errorf("upsHost is not the production host of the carrier")
	}
	if upsTokenPath != "/security/v1/oauth/token" || upsTrackPath != "/api/track/v1/details/" {
		t.Errorf("paths = %q %q", upsTokenPath, upsTrackPath)
	}
	adapter, ok := a.(*upsAdapter)
	if !ok || adapter.trackBase != "https://"+upsHost+upsTrackPath || adapter.tokenURL != "https://"+upsHost+upsTokenPath {
		t.Errorf("production urls = %+v", a)
	}
}

// ---- panic isolation

func Test_UPS_panicInTheTransportIsContained(t *testing.T) {
	stub := newUPSStub(t)
	logs := logtest.New(t)
	h, _ := newTestHandler(t, options{newClient: func(time.Duration) *http.Client {
		return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { panic("PANIC-WITH " + testSecretValue) })}
	}}, stub.adapter())

	text, isError, err := h.Call(context.Background(), makeCall("ups", upsNumber, testSecret))
	if !errors.Is(err, errPanic) || isError || text != "" {
		t.Fatalf("got %q %v %v", text, isError, err)
	}
	logs.AssertAbsent(t, testSecretValue, testSecret, "PANIC-WITH")
	if n := len(h.state("ups").slots); n != 0 {
		t.Errorf("%d slots leaked", n)
	}
}

func Test_UPS_oddAnswersNeverPanic(t *testing.T) {
	bodies := []string{
		`{"trackResponse":{"shipment":[{"package":[{"activity":[{"location":null,"status":null}],"deliveryDate":[{"date":null,"type":null}],"currentStatus":null,"service":null}]}]}}`,
		`{"trackResponse":{"shipment":[{"package":[null,null]}]}}`,
		`{"trackResponse":{"shipment":[null,{"package":[{"activity":[null]}]}]}}`,
		`{"trackResponse":{"shipment":[{"package":[{"trackingNumber":"` + upsNumber + `","activity":[{"date":"20260102","time":"101500","gmtDate":"99999999","gmtTime":"999999","gmtOffset":"-99:99"}]}]}]}}`,
		`[]`, `"x"`, `1`, `true`, `{"trackResponse":[]}`,
	}
	for i, b := range bodies {
		t.Run(fmt.Sprintf("answer %d", i), func(t *testing.T) {
			res, cerr := mapUPS([]byte(b), upsNumber)
			if (res == nil) == (cerr == nil) {
				t.Errorf("exactly one of a result and an error is expected: %+v %+v", res, cerr)
			}
			if res != nil {
				res.finalize("ups", upsNumber, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
				if _, err := res.marshal(); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// ---- opening and closing

// Test_UPS_openingAndClosing shows that a carrier is open only when its table line and
// its adapter are both there: remove either and the server answers with the
// fixed text and nothing is sent.
func Test_UPS_openingAndClosing(t *testing.T) {
	tests := []struct {
		name     string
		table    mcpserver.CarrierTable
		adapters []Adapter
		open     bool
	}{
		{"table line and adapter", mcpserver.DefaultCarrierTable(), []Adapter{NewUPS()}, true},
		{"table line without an adapter", mcpserver.DefaultCarrierTable(), nil, false},
		{"adapter without a table line", mcpserver.CarrierTable{}, []Adapter{NewUPS()}, false},
		{"neither", mcpserver.CarrierTable{}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			var requests atomic.Int32
			h, _ := newTestHandler(t, options{newClient: func(time.Duration) *http.Client {
				return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					requests.Add(1)
					return nil, errors.New("no network in this test")
				})}
			}}, tt.adapters...)

			id := uuid.Must(uuid.NewV4())
			server := &mcpserver.McpServer{
				Identity: commonidentity.Identity{ID: id, CustomerID: customerA},
				URL:      "https://carrier.invalid/ups",
				Status:   mcpserver.StatusActive,
				AuthType: mcpserver.AuthTypeBearer,
			}
			db := dbhandler.NewMockDBHandler(mc)
			db.EXPECT().McpServerGet(gomock.Any(), id).Return(server, nil).AnyTimes()

			tool, err := mcptoolhandler.NewMcpToolHandler(db, "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", 5, nil, tt.table, h)
			if err != nil {
				t.Fatal(err)
			}

			tools, err := tool.ListTools(context.Background(), id)
			if tt.open {
				if err != nil || len(tools) != 1 || tools[0].Name != "track_ups" {
					t.Errorf("an open carrier lists its tool: %+v %v", tools, err)
				}
			} else if err == nil || len(tools) != 0 {
				t.Errorf("a closed carrier lists nothing: %+v %v", tools, err)
			}

			if !tt.open {
				text, isError, err := tool.CallTool(context.Background(), id, "track_ups", fmt.Sprintf(`{"tracking_number":%q}`, upsNumber))
				if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
					t.Errorf("got %q %v %v", text, isError, err)
				}
			}
			if requests.Load() != 0 {
				t.Errorf("nothing may be sent: %d requests", requests.Load())
			}
		})
	}
}

func Test_UPS_productionHandlerHasTheTool(t *testing.T) {
	h, err := New(10)
	if err != nil {
		t.Fatal(err)
	}
	defs, ok := h.Tools("ups")
	if !ok || len(defs) != 1 || defs[0].Name != "track_ups" {
		t.Errorf("the production handler must list track_ups: %+v %v", defs, ok)
	}
}
