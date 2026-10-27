package aipromptproposal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_CreateWebhookEvent_fields(t *testing.T) {
	updated := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	p := &AIPromptProposal{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		AIID:                   uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		AuditIDs:               []uuid.UUID{uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")},
		BasisPromptHistoryID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		OriginalPrompt:         "old prompt",
		ProposedPrompt:         "new prompt",
		Rationale:              "because",
		Error:                  "some error",
		AppliedPromptHistoryID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
		TMUpdate:               &updated,
	}

	data, err := p.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	expect := map[string]string{
		"id":                        "11111111-1111-1111-1111-111111111111",
		"customer_id":               "22222222-2222-2222-2222-222222222222",
		"ai_id":                     "33333333-3333-3333-3333-333333333333",
		"basis_prompt_history_id":   "55555555-5555-5555-5555-555555555555",
		"original_prompt":           "old prompt",
		"proposed_prompt":           "new prompt",
		"rationale":                 "because",
		"error":                     "some error",
		"applied_prompt_history_id": "66666666-6666-6666-6666-666666666666",
		"tm_update":                 "2026-03-04T05:06:07Z",
	}
	for k, v := range expect {
		if got[k] != v {
			t.Errorf("field %s = %v, expect: %v", k, got[k], v)
		}
	}
	ids, ok := got["audit_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "44444444-4444-4444-4444-444444444444" {
		t.Errorf("audit_ids = %v", got["audit_ids"])
	}
	if got["tm_create"] != nil || got["tm_delete"] != nil {
		t.Errorf("expected null tm_create/tm_delete")
	}
}
