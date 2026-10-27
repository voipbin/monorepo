package session

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmEnd := time.Date(2026, 1, 2, 4, 4, 5, 0, time.UTC)
	tmDelete := time.Date(2026, 1, 3, 4, 4, 5, 0, time.UTC)

	s := &Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		WidgetID:     uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		Status:       StatusEnded,
		PageURL:      "https://example.com/page",
		Referrer:     "https://example.com/ref",
		Peer:         commonaddress.Address{Target: "peer-target"},
		Local:        commonaddress.Address{Target: "local-target"},
		ActiveflowID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		TMCreate:     &tmCreate,
		TMEnd:        &tmEnd,
		TMDelete:     &tmDelete,
	}

	expect := &WebhookMessage{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"), CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002")},
		WidgetID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		Status:   StatusEnded,
		PageURL:  "https://example.com/page",
		Referrer: "https://example.com/ref",
		Peer:     commonaddress.Address{Target: "peer-target"},
		Local:    commonaddress.Address{Target: "local-target"},
		TMCreate: &tmCreate,
		TMEnd:    &tmEnd,
	}

	res := s.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmDelete := time.Date(2026, 1, 3, 4, 4, 5, 0, time.UTC)

	s := &Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		WidgetID:     uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		Status:       StatusActive,
		ActiveflowID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		TMDelete:     &tmDelete,
	}

	data, err := s.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// internal fields must never be part of the external payload
	for _, key := range []string{"activeflow_id", "tm_delete"} {
		if _, ok := got[key]; ok {
			t.Errorf("%s must not be exposed. got: %v", key, got)
		}
	}
	for _, key := range []string{"id", "customer_id", "widget_id", "status", "peer", "local"} {
		if _, ok := got[key]; !ok {
			t.Errorf("%s must be exposed. got: %v", key, got)
		}
	}
	if got["id"] != "11111111-aaaa-bbbb-cccc-000000000001" || got["status"] != "active" || got["widget_id"] != "11111111-aaaa-bbbb-cccc-000000000003" {
		t.Errorf("Wrong values. got: %v", got)
	}
}

// A timestamp outside the year range 0 to 9999 cannot be encoded, so the event
// payload is not produced.
func Test_CreateWebhookEvent_encodeFailure(t *testing.T) {
	tmCreate := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	m := &Session{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001")},
		TMCreate: &tmCreate,
	}

	res, err := m.CreateWebhookEvent()
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
}
