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
				AIID:          uuid.Must(uuid.NewV4()),
				AIEngineType:  ai.EngineTypeNone,
				AIEngineModel: ai.EngineModelOpenaiGPT4O,
				AIEngineData:  map[string]any{"key": "value"},
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "voice-123",
				AISTTType:     ai.STTTypeDeepgram,
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				ConfbridgeID:  uuid.Must(uuid.NewV4()),
				PipecatcallID: uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				Gender:        GenderFemale,
				Language:      "en-US",
				TMEnd:         ptrTime(time.Now()),
				TMCreate:      ptrTime(time.Now()),
				TMUpdate:      ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.AIID != ac.AIID {
					t.Errorf("Wrong AIID. expect: %s, got: %s", ac.AIID, wh.AIID)
				}
				if wh.Status != ac.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", ac.Status, wh.Status)
				}
				if wh.Gender != ac.Gender {
					t.Errorf("Wrong Gender. expect: %s, got: %s", ac.Gender, wh.Gender)
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
				AIID:          uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				Gender:        GenderMale,
				Language:      "ko-KR",
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

func ptrTime(t time.Time) *time.Time {
	return &t
}
