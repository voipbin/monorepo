package aicall

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name      string
		aicall    *AIcall
		checkFunc func(t *testing.T, wh *WebhookMessage, ac *AIcall)
	}{
		{
			name: "converts_aicall_with_all_fields",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				Parameter:  map[string]any{"key": "value"},
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "voice-123",
				AISTTType:     ai.STTTypeDeepgram,
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				ConfbridgeID:  uuid.Must(uuid.NewV4()),
				PipecatcallID: uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				STTLanguage:   "en-US",
				TMEnd:         ptrTime(time.Now()),
				TMCreate:      ptrTime(time.Now()),
				TMUpdate:      ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.AssistanceType != ac.AssistanceType {
					t.Errorf("Wrong AssistanceType. expect: %s, got: %s", ac.AssistanceType, wh.AssistanceType)
				}
				if wh.AssistanceID != ac.AssistanceID {
					t.Errorf("Wrong AssistanceID. expect: %s, got: %s", ac.AssistanceID, wh.AssistanceID)
				}
				if wh.AIEngineModel != ac.AIEngineModel {
					t.Errorf("Wrong AIEngineModel. expect: %s, got: %s", ac.AIEngineModel, wh.AIEngineModel)
				}
				if wh.Parameter["key"] != ac.Parameter["key"] {
					t.Errorf("Wrong Parameter. expect: %v, got: %v", ac.Parameter, wh.Parameter)
				}
				if wh.AITTSType != ac.AITTSType {
					t.Errorf("Wrong AITTSType. expect: %s, got: %s", ac.AITTSType, wh.AITTSType)
				}
				if wh.AITTSVoiceID != ac.AITTSVoiceID {
					t.Errorf("Wrong AITTSVoiceID. expect: %s, got: %s", ac.AITTSVoiceID, wh.AITTSVoiceID)
				}
				if wh.AISTTType != ac.AISTTType {
					t.Errorf("Wrong AISTTType. expect: %s, got: %s", ac.AISTTType, wh.AISTTType)
				}
				if wh.Status != ac.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", ac.Status, wh.Status)
				}
				if wh.STTLanguage != ac.STTLanguage {
					t.Errorf("Wrong STTLanguage. expect: %s, got: %s", ac.STTLanguage, wh.STTLanguage)
				}
			},
		},
		{
			name: "converts_aicall_with_current_member_id",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				CurrentMemberID: uuid.Must(uuid.NewV4()),
				Status:          StatusProgressing,
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.CurrentMemberID != ac.CurrentMemberID {
					t.Errorf("Wrong CurrentMemberID. expect: %s, got: %s", ac.CurrentMemberID, wh.CurrentMemberID)
				}
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.Status != ac.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", ac.Status, wh.Status)
				}
			},
		},
		{
			name: "converts_aicall_with_empty_fields",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.CurrentMemberID != uuid.Nil {
					t.Errorf("Wrong CurrentMemberID. expect: %s, got: %s", uuid.Nil, wh.CurrentMemberID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := tt.aicall.ConvertWebhookMessage()
			if wh == nil {
				t.Error("Expected non-nil webhook, got nil")
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, wh, tt.aicall)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		aicall    *AIcall
		wantError bool
	}{
		{
			name: "creates_webhook_event_successfully",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
				Status:         StatusProgressing,
				STTLanguage:   "ko-KR",
				ReferenceType: ReferenceTypeConversation,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				TMCreate:      ptrTime(time.Now()),
			},
			wantError: false,
		},
		{
			name: "creates_webhook_event_with_empty_aicall",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.aicall.CreateWebhookEvent()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateWebhookEvent() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify it's valid JSON
				var wh WebhookMessage
				if errUnmarshal := json.Unmarshal(data, &wh); errUnmarshal != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", errUnmarshal)
				}
			}
		})
	}
}

func TestConvertWebhookMessage_includesMetadata(t *testing.T) {
	h := &AIcall{
		Metadata: map[string]any{
			MetaKeyPromptSnapshots: []PromptSnapshot{
				{Prompt: "hello world"},
			},
		},
	}
	msg := h.ConvertWebhookMessage()
	if msg.Metadata == nil {
		t.Fatal("expected Metadata to be non-nil in WebhookMessage")
	}
	if _, ok := msg.Metadata[MetaKeyPromptSnapshots]; !ok {
		t.Errorf("expected %q key in Metadata", MetaKeyPromptSnapshots)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
