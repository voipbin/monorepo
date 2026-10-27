package aiaudit_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aiaudit"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_CreateWebhookEvent_fields(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	score := 7

	a := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		AIcallID:        uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		AIID:            uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		PromptHistoryID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		Status:          aiaudit.StatusFailed,
		OverallScore:    &score,
		Evaluation:      json.RawMessage(`{"k":1}`),
		MessageIDs:      []uuid.UUID{uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")},
		Language:        "ko-KR",
		Error:           string(aiaudit.ErrorCancelled),
		TMCreate:        &created,
	}

	data, err := a.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	expectStr := map[string]string{
		"id":                "11111111-1111-1111-1111-111111111111",
		"customer_id":       "22222222-2222-2222-2222-222222222222",
		"aicall_id":         "33333333-3333-3333-3333-333333333333",
		"ai_id":             "44444444-4444-4444-4444-444444444444",
		"prompt_history_id": "55555555-5555-5555-5555-555555555555",
		"status":            "failed",
		"language":          "ko-KR",
		"error":             "cancelled",
		"tm_create":         "2026-01-02T03:04:05Z",
	}
	for k, v := range expectStr {
		if got[k] != v {
			t.Errorf("field %s = %v, expect: %v", k, got[k], v)
		}
	}
	if got["overall_score"] != float64(7) {
		t.Errorf("overall_score = %v, expect: 7", got["overall_score"])
	}
	ids, ok := got["message_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "66666666-6666-6666-6666-666666666666" {
		t.Errorf("message_ids = %v", got["message_ids"])
	}
	if got["tm_update"] != nil || got["tm_delete"] != nil {
		t.Errorf("expected nil tm_update/tm_delete, got %v / %v", got["tm_update"], got["tm_delete"])
	}
}

func Test_CreateWebhookEvent_zero(t *testing.T) {
	data, err := (&aiaudit.AIAudit{}).CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	// overall_score and evaluation have no omitempty and are always present
	for _, k := range []string{"overall_score", "evaluation", "tm_create"} {
		v, ok := got[k]
		if !ok || v != nil {
			t.Errorf("expected key %s present with null, got %v (present: %v)", k, v, ok)
		}
	}
	if _, ok := got["status"]; ok {
		t.Errorf("empty status should be omitted")
	}
}
