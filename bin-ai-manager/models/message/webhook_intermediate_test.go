package message

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
)

func Test_IntermediateWebhookMessage_CreateWebhookEvent(t *testing.T) {
	m := &IntermediateWebhookMessage{
		Identity:     identity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"), CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")},
		AIcallID:     uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		ActiveflowID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		ActiveAIID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		Role:         RoleAssistant,
		Content:      "delta",
		Direction:    DirectionOutgoing,
		Sequence:     7,
	}

	data, err := m.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	expect := map[string]any{
		"id":            "11111111-1111-1111-1111-111111111111",
		"customer_id":   "22222222-2222-2222-2222-222222222222",
		"aicall_id":     "33333333-3333-3333-3333-333333333333",
		"activeflow_id": "44444444-4444-4444-4444-444444444444",
		"active_ai_id":  "55555555-5555-5555-5555-555555555555",
		"role":          string(RoleAssistant),
		"content":       "delta",
		"direction":     string(DirectionOutgoing),
		"sequence":      float64(7),
	}
	for k, v := range expect {
		if got[k] != v {
			t.Errorf("field %s = %v, expect: %v", k, got[k], v)
		}
	}
}
