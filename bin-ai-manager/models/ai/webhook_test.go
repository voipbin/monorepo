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
