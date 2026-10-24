package message

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

	m := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		WidgetID:     uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		SessionID:    uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		Direction:    DirectionOutbound,
		Status:       StatusDelivered,
		SenderID:     uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000005"),
		ActiveflowID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000006"),
		Text:         "synthetic text",
		TMCreate:     &tmCreate,
	}

	expect := &WebhookMessage{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"), CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002")},
		WidgetID:  uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		SessionID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		Direction: DirectionOutbound,
		Status:    StatusDelivered,
		SenderID:  uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000005"),
		Text:      "synthetic text",
		TMCreate:  &tmCreate,
	}

	res := m.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	m := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000002"),
		},
		WidgetID:     uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000003"),
		SessionID:    uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000004"),
		Direction:    DirectionInbound,
		Status:       StatusSent,
		ActiveflowID: uuid.FromStringOrNil("11111111-aaaa-bbbb-cccc-000000000006"),
		Text:         "synthetic text",
	}

	data, err := m.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the internal activeflow marker must never be part of the external payload
	if _, ok := got["activeflow_id"]; ok {
		t.Errorf("activeflow_id must not be exposed. got: %v", got)
	}
	expect := map[string]any{
		"id":          "11111111-aaaa-bbbb-cccc-000000000001",
		"customer_id": "11111111-aaaa-bbbb-cccc-000000000002",
		"widget_id":   "11111111-aaaa-bbbb-cccc-000000000003",
		"session_id":  "11111111-aaaa-bbbb-cccc-000000000004",
		"direction":   "inbound",
		"sender_id":   uuid.Nil.String(), // omitempty is inert on a uuid.UUID value
		"status":      "sent",
		"text":        "synthetic text",
		"tm_create":   nil,
		"tm_delete":   nil,
	}
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, got)
	}
}
