package call

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// OutboundWhitelistRejectedEvent overrides the subscription address of the global topic exchange
// (VOIP-1404 / VOIP-1405 §2.2). The assertion pins the POINTER type: the event data reaches
// notifyhandler as a POINTER and the assertion matches the dynamic type; a VALUE of this
// pointer-receiver type would fail the assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*OutboundWhitelistRejectedEvent)(nil)

func TestOutboundWhitelistRejectedEventEventSubscriptionID(t *testing.T) {
	callID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name   string
		event  *OutboundWhitelistRejectedEvent
		expect string
	}{
		{
			"normal",
			&OutboundWhitelistRejectedEvent{
				CallID:             callID,
				CustomerID:         uuid.Must(uuid.NewV4()),
				DestinationCountry: "US",
			},
			callID.String(),
		},
		{
			"empty call id",
			&OutboundWhitelistRejectedEvent{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.event.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestOutboundWhitelistRejectedEventHasNoOwnID pins the reason the override is mandatory rather
// than optional: the payload carries no top-level `id`, so without the override notifyhandler's
// JSON fallback finds nothing and every one of these events would be routed to the `-` placeholder
// address. The address must also never be the customer id.
func TestOutboundWhitelistRejectedEventHasNoOwnID(t *testing.T) {
	e := &OutboundWhitelistRejectedEvent{
		CallID:             uuid.Must(uuid.NewV4()),
		CustomerID:         uuid.Must(uuid.NewV4()),
		DestinationCountry: "KR",
	}

	m, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Could not marshal the event. err: %v", err)
	}

	parsed := map[string]any{}
	if errUnmarshal := json.Unmarshal(m, &parsed); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal the event. err: %v", errUnmarshal)
	}

	if _, ok := parsed["id"]; ok {
		t.Errorf("The event must not carry a top-level id. payload: %s", m)
	}

	if e.EventSubscriptionID() == e.CustomerID.String() {
		t.Errorf("Subscription address must not be the customer id. customer_id: %s", e.CustomerID)
	}
	if e.EventSubscriptionID() != e.CallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", e.CallID, e.EventSubscriptionID())
	}
}

// TestOutboundWhitelistRejectedEventJSONKeySet pins the payload-compatibility contract of the
// map->struct conversion (VOIP-1405 §3.3 / AC3): fanout consumers must observe the exact same JSON
// key SET the inline map used to produce. Only the key ORDER differs (Go marshals maps key-sorted
// and structs in declaration order), which is why this compares key sets and never bytes.
func TestOutboundWhitelistRejectedEventJSONKeySet(t *testing.T) {
	callID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	country := "GB"

	// the exact shape pkg/callhandler/outgoing_call.go published before the conversion
	legacy := map[string]any{
		"customer_id":         customerID,
		"call_id":             callID,
		"destination_country": country,
	}
	current := &OutboundWhitelistRejectedEvent{
		CallID:             callID,
		CustomerID:         customerID,
		DestinationCountry: country,
	}

	legacyParsed := marshalToMap(t, legacy)
	currentParsed := marshalToMap(t, current)

	if !reflect.DeepEqual(keysOf(legacyParsed), keysOf(currentParsed)) {
		t.Fatalf("Wrong match. expect: %v, got: %v", keysOf(legacyParsed), keysOf(currentParsed))
	}

	// the values must round-trip identically as well, not merely the key names
	if !reflect.DeepEqual(legacyParsed, currentParsed) {
		t.Errorf("Wrong match. expect: %v, got: %v", legacyParsed, currentParsed)
	}
}

func marshalToMap(t *testing.T, data any) map[string]any {
	t.Helper()

	m, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Could not marshal the data. err: %v", err)
	}

	res := map[string]any{}
	if errUnmarshal := json.Unmarshal(m, &res); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal the data. err: %v", errUnmarshal)
	}

	return res
}

func keysOf(m map[string]any) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)

	return res
}

func TestCallEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expect   string
	}{
		{"call_created", EventTypeCallCreated, "call_created"},
		{"call_updated", EventTypeCallUpdated, "call_updated"},
		{"call_deleted", EventTypeCallDeleted, "call_deleted"},
		{"call_dialing", EventTypeCallDialing, "call_dialing"},
		{"call_ringing", EventTypeCallRinging, "call_ringing"},
		{"call_progressing", EventTypeCallProgressing, "call_progressing"},
		{"call_terminating", EventTypeCallTerminating, "call_terminating"},
		{"call_canceling", EventTypeCallCanceling, "call_canceling"},
		{"call_hangup", EventTypeCallHangup, "call_hangup"},
		{"call.outbound_whitelist_rejected", EventTypeCallOutboundWhitelistRejected, "call.outbound_whitelist_rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.constant)
			}
		})
	}
}
