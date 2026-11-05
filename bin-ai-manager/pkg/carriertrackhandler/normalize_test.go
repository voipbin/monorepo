package carriertrackhandler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func Test_sanitizeText(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		max    int
		expect string
	}{
		{"plain", "Departed Facility", 200, "Departed Facility"},
		{"control characters and line breaks", "a\x00b\r\nc\td\u0007e", 200, "a b c d e"},
		{"collapses runs of space", "a    b", 200, "a b"},
		{"trims", "  hello  ", 200, "hello"},
		{"masks an email address", "Left with jane.doe@example.com at the door", 200, "Left with [redacted] at the door"},
		{"masks a phone number", "Call +1 (555) 010-9999 now", 200, "Call [redacted] now"},
		{"masks a grouped phone number", "Call (555) 010-9999 now", 200, "Call [redacted] now"},
		{"masks a dotted phone number", "Call 555.010.9999 now", 200, "Call [redacted] now"},
		{"masks a plain digit phone number", "Call 5550109999 now", 200, "Call 5550109999 now"},
		{"keeps a date", "Delivered 2026-11-05 at 10:00", 200, "Delivered 2026-11-05 at 10:00"},
		{"keeps a timestamp", "At 2026-11-05T10:00:00Z", 200, "At 2026-11-05T10:00:00Z"},
		{"keeps a tracking number", "Package 794644790138 and 1Z999AA10123456784", 200, "Package 794644790138 and 1Z999AA10123456784"},
		{"keeps short numbers", "Scan 12 of 34", 200, "Scan 12 of 34"},
		{"cuts on a rune boundary", strings.Repeat("\u00e9", 10), 4, strings.Repeat("\u00e9", 4)},
		{"cuts to the limit", strings.Repeat("a", 300), 200, strings.Repeat("a", 200)},
		{"empty", "", 200, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeText(tt.in, tt.max)
			if got != tt.expect {
				t.Errorf("sanitizeText(%q, %d) = %q, expected %q", tt.in, tt.max, got, tt.expect)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result is not valid utf-8: %q", got)
			}
		})
	}
}

func str(s string) *string { return &s }

func sampleResult(nEvents int) *Result {
	events := make([]Event, 0, nEvents)
	for i := 0; i < nEvents; i++ {
		events = append(events, Event{
			Time:        str(fmt.Sprintf("2026-01-01T%02d:00:00Z", i%24)),
			Description: fmt.Sprintf("event %d %s", i, strings.Repeat("x", 400)),
			Location:    &Location{Text: strings.Repeat("L", 400)},
		})
	}
	return &Result{
		Status:            StatusInTransit,
		StatusText:        strings.Repeat("S", 400),
		Service:           str(strings.Repeat("V", 400)),
		CarrierStatusCode: str(strings.Repeat("C", 400)),
		Events:            events,
	}
}

func Test_Result_finalize_capsEveryField(t *testing.T) {
	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	r := sampleResult(20)
	r.finalize("fake", "FAKE12345678", now)

	if len(r.Events) != maxEvents || !r.EventsTruncated {
		t.Errorf("events = %d truncated = %v, expected %d and true", len(r.Events), r.EventsTruncated, maxEvents)
	}
	if n := utf8.RuneCountInString(r.StatusText); n != maxTextLen {
		t.Errorf("status_text has %d runes", n)
	}
	for i, e := range r.Events {
		if utf8.RuneCountInString(e.Description) > maxTextLen || utf8.RuneCountInString(e.Location.Text) > maxLocationLen {
			t.Errorf("event %d is not capped", i)
		}
	}
	if utf8.RuneCountInString(*r.Service) > maxServiceLen || utf8.RuneCountInString(*r.CarrierStatusCode) > maxStatusCodeLen {
		t.Errorf("service or code not capped")
	}
	if r.LastEvent == nil || r.LastEvent.Description != r.Events[0].Description {
		t.Errorf("last_event must default to the newest event")
	}
	if r.Carrier != "fake" || r.TrackingNumber != "FAKE12345678" || r.FetchedAt != "2026-02-03T04:05:06Z" || r.Cached || r.Note == "" {
		t.Errorf("fixed fields wrong: %+v", r)
	}

	d := r.Display
	if len(d.Title) > 200 || len(d.Description) > 1000 {
		t.Errorf("display title or description too long")
	}
	if len(d.Fields) == 0 || len(d.Fields) > maxDisplayFields {
		t.Errorf("display has %d fields", len(d.Fields))
	}
	for _, f := range d.Fields {
		if utf8.RuneCountInString(f.Label) > 50 || utf8.RuneCountInString(f.Value) > 500 || f.Value == "" {
			t.Errorf("display field out of bounds: %+v", f)
		}
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 16<<10 {
		t.Errorf("worst-case result is %d bytes, over the 16 KiB cache entry limit", len(b))
	}
}

func Test_Result_finalize_unknownStatusBecomesUnknown(t *testing.T) {
	r := &Result{Status: Status("weird")}
	r.finalize("fake", "N", time.Now())
	if r.Status != StatusUnknown {
		t.Errorf("status = %q", r.Status)
	}
}

func Test_Result_marshal_allowList(t *testing.T) {
	r := sampleResult(3)
	r.finalize("fake", "N1", time.Now())
	text, err := r.marshal()
	if err != nil {
		t.Fatal(err)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &top); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	expect := []string{"cached", "carrier", "carrier_status_code", "delivered_at", "display", "estimated_delivery", "events", "events_truncated", "fetched_at", "last_event", "note", "service", "status", "status_text", "time_zone_known", "tracking_number"}
	if !reflect.DeepEqual(keys, expect) {
		t.Errorf("keys = %v, expected %v", keys, expect)
	}

	for _, forbidden := range []string{"recipient", "receiver", "sender", "shipper", "signature", "signed", "address", "postal", "payment", "latitude"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Errorf("the output must not contain %q", forbidden)
		}
	}
}

func Test_cleanStructured(t *testing.T) {
	tests := []struct {
		in     string
		limit  int
		expect string
	}{
		{"2026-11-05", 64, "2026-11-05"},
		{"2026-11-05T10:00:00-05:00", 64, "2026-11-05T10:00:00-05:00"},
		{"1Z999AA10123456784", 64, "1Z999AA10123456784"},
		{"a\r\nb\x00c", 64, "a b c"},
		{strings.Repeat("x", 500), 64, strings.Repeat("x", 64)},
		{"   ", 64, ""},
	}
	for _, tt := range tests {
		if got := cleanStructured(tt.in, tt.limit); got != tt.expect {
			t.Errorf("cleanStructured(%q) = %q, expected %q", tt.in, got, tt.expect)
		}
	}
}

func Test_Result_finalize_keepsStructuredFieldsAndCapsThem(t *testing.T) {
	long := strings.Repeat("9", 100000)
	r := &Result{
		Status:      StatusDelivered,
		DeliveredAt: str("2026-11-05T10:00:00Z"),
		EstimatedDelivery: EstimatedDelivery{
			Date: str(long), From: str(long), To: str(long),
		},
		Events: []Event{{Time: str(long), TimeLocal: str(long), Description: "x"}, {Time: str("2026-11-04T10:00:00Z"), Description: "y"}},
	}
	r.LastEvent = &Event{Time: str(long), Description: "z"}
	r.finalize("fake", "794644790138", time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))

	if *r.DeliveredAt != "2026-11-05T10:00:00Z" {
		t.Errorf("a clean timestamp must be kept as it is: %q", *r.DeliveredAt)
	}
	if r.TrackingNumber != "794644790138" {
		t.Errorf("tracking number = %q", r.TrackingNumber)
	}
	for _, v := range []*string{r.EstimatedDelivery.Date, r.EstimatedDelivery.From, r.EstimatedDelivery.To, r.Events[0].Time, r.Events[0].TimeLocal, r.LastEvent.Time} {
		if v == nil || len(*v) > maxTimeLen {
			t.Errorf("a time field is not capped")
		}
	}

	fieldValues := map[string]string{}
	for _, f := range r.Display.Fields {
		fieldValues[f.Label] = f.Value
	}
	if fieldValues["Tracking number"] != "794644790138" || fieldValues["Delivered"] != "2026-11-05T10:00:00Z" {
		t.Errorf("the card lost its structured values: %+v", fieldValues)
	}
	if r.Display.Title != "FAKE package 794644790138" {
		t.Errorf("title = %q", r.Display.Title)
	}

	text, err := r.marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > 16<<10 {
		t.Errorf("a result with hostile time fields is %d bytes", len(text))
	}
}
