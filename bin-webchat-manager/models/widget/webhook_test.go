package widget

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	theme := &ThemeConfig{PrimaryColor: "#112233"}

	w := &Widget{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		Name:               "test widget",
		Status:             StatusInactive,
		DirectID:           uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		Hash:               "synthetic-hash",
		SessionFlowID:      uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		MessageFlowID:      uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000005"),
		SessionIdleTimeout: 900,
		ThemeConfig:        theme,
		TMCreate:           &tmCreate,
		TMUpdate:           &tmUpdate,
	}

	expect := &WebhookMessage{
		Identity:           commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"), CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002")},
		Name:               "test widget",
		Status:             StatusInactive,
		SessionFlowID:      "11111111-aaaa-bbbb-cccc-000000000004",
		MessageFlowID:      "11111111-aaaa-bbbb-cccc-000000000005",
		SessionIdleTimeout: 900,
		ThemeConfig:        theme,
		DirectHash:         "synthetic-hash",
		TMCreate:           &tmCreate,
		TMUpdate:           &tmUpdate,
	}

	res := w.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	w := &Widget{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		Name:     "test widget",
		Status:   StatusActive,
		DirectID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		Hash:     "synthetic-hash",
	}

	data, err := w.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the internal direct id must never be part of the external payload, the hash is exposed
	if _, ok := got["direct_id"]; ok {
		t.Errorf("direct_id must not be exposed. got: %v", got)
	}
	expect := map[string]any{
		"id":                   "11111111-aaaa-bbbb-cccc-000000000001",
		"customer_id":          "11111111-aaaa-bbbb-cccc-000000000002",
		"name":                 "test widget",
		"status":               "active",
		"session_flow_id":      uuid.Nil.String(),
		"message_flow_id":      uuid.Nil.String(),
		"session_idle_timeout": float64(0),
		"direct_hash":          "synthetic-hash",
	}
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, got)
	}
}

// A timestamp outside the year range 0 to 9999 cannot be encoded, so the event
// payload is not produced.
func Test_CreateWebhookEvent_encodeFailure(t *testing.T) {
	tmCreate := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	m := &Widget{
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
