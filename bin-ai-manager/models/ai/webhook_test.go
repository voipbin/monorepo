package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage_CurrentPromptHistoryID(t *testing.T) {
	historyID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name       string
		ai         *AI
		expectID   uuid.UUID
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

func Test_ConvertWebhookMessage_EngineKeyNotExposed(t *testing.T) {
	const dummyKey = "dummy-engine-key-value"

	tests := []struct {
		name          string
		engineKey     string
		wantHasKey    bool
		wantHasKeyStr string
	}{
		{"key_set", dummyKey, true, `"has_engine_key":true`},
		{"key_empty", "", false, `"has_engine_key":false`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AI{
				Identity:  commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				EngineKey: tt.engineKey,
			}

			msg := a.ConvertWebhookMessage()
			if msg.HasEngineKey != tt.wantHasKey {
				t.Errorf("HasEngineKey = %v, want %v", msg.HasEngineKey, tt.wantHasKey)
			}

			// ConvertWebhookMessage output and CreateWebhookEvent output must both be key free.
			fromConvert, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			fromEvent, err := a.CreateWebhookEvent()
			if err != nil {
				t.Fatalf("CreateWebhookEvent: %v", err)
			}

			for name, b := range map[string][]byte{"convert": fromConvert, "event": fromEvent} {
				s := string(b)
				if strings.Contains(s, `"engine_key"`) || strings.Contains(s, dummyKey) {
					t.Errorf("%s output must not carry the engine key: %s", name, s)
				}
				if !strings.Contains(s, tt.wantHasKeyStr) {
					t.Errorf("%s output must contain %s: %s", name, tt.wantHasKeyStr, s)
				}
			}
		})
	}
}

// A stored event written before the event masking carries a plaintext engine_key. Unmarshalling it
// into AI and converting again (the timeline replay path) must not bring the key back.
func Test_ConvertWebhookMessage_LegacyEventWithPlaintextKey(t *testing.T) {
	legacy := `{"id":"a1a1a1a1-0000-0000-0000-000000000001","engine_model":"openai.gpt-5","engine_key":"dummy-legacy-key"}`

	var a AI
	if err := json.Unmarshal([]byte(legacy), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	b, err := json.Marshal(a.ConvertWebhookMessage())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "dummy-legacy-key") || strings.Contains(string(b), `"engine_key"`) {
		t.Errorf("converted legacy event must not carry the key: %s", b)
	}
}

// publishAIEvent clears the key and sets the HasEngineKey hint first. The converted message of that
// event copy must report has_engine_key true and no key, so events, websocket and aggregated events
// agree with the REST response.
func Test_ConvertWebhookMessage_MaskedEventHint(t *testing.T) {
	tests := []struct {
		name       string
		hint       bool
		wantHasKey bool
	}{
		{"hint_set", true, true},
		{"hint_unset", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AI{
				Identity:     commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				HasEngineKey: tt.hint,
			}

			msg := a.ConvertWebhookMessage()
			if msg.HasEngineKey != tt.wantHasKey {
				t.Errorf("HasEngineKey = %v, want %v", msg.HasEngineKey, tt.wantHasKey)
			}

			b, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(b), `"engine_key"`) {
				t.Errorf("output must not carry engine_key: %s", b)
			}
		})
	}
}
