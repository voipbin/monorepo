package ai

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage_CurrentPromptHistoryID(t *testing.T) {
	historyID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		ai        *AI
		expectID  uuid.UUID
		wantInJSON bool
	}{
		{
			name: "non_zero_current_prompt_history_id_is_copied",
			ai: &AI{
				Identity:               commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				CurrentPromptHistoryID: historyID,
			},
			expectID:   historyID,
			wantInJSON: true,
		},
		{
			name: "zero_current_prompt_history_id_is_present_in_json",
			ai: &AI{
				Identity:               commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				CurrentPromptHistoryID: uuid.Nil,
			},
			expectID:   uuid.Nil,
			wantInJSON: true, // uuid.UUID is [16]byte; omitempty has no effect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := tt.ai.ConvertWebhookMessage()
			if wh == nil {
				t.Fatal("ConvertWebhookMessage returned nil")
			}

			if wh.CurrentPromptHistoryID != tt.expectID {
				t.Errorf("CurrentPromptHistoryID mismatch. expect: %s, got: %s", tt.expectID, wh.CurrentPromptHistoryID)
			}

			data, err := json.Marshal(wh)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			_, present := raw["current_prompt_history_id"]
			if tt.wantInJSON && !present {
				t.Errorf("expected current_prompt_history_id to be present in JSON output")
			}
		})
	}
}

// is_insight_active must be carried into WebhookMessage and must appear on the
// wire even when false — with omitempty, "inactive" would be indistinguishable
// from "field absent" for any client parsing the JSON. The RST struct docs for
// /ais track WebhookMessage, so this also guards the documented contract.
func TestConvertWebhookMessage_IsInsightActive(t *testing.T) {
	tests := []struct {
		name   string
		active bool
	}{
		{"active_insight_ai", true},
		{"inactive_insight_ai", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AI{
				Identity:        commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				Type:            TypeInsight,
				IsInsightActive: tt.active,
			}

			wh := a.ConvertWebhookMessage()
			if wh.IsInsightActive != tt.active {
				t.Errorf("IsInsightActive mismatch. expect: %v, got: %v", tt.active, wh.IsInsightActive)
			}

			data, err := json.Marshal(wh)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			got, present := raw["is_insight_active"]
			if !present {
				t.Fatalf("expected is_insight_active to be present in JSON output even when %v", tt.active)
			}
			if got != tt.active {
				t.Errorf("is_insight_active in JSON: expect %v, got %v", tt.active, got)
			}
		})
	}
}
