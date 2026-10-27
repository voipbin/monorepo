package team

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_ConvertWebhookMessage_fields(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	h := &Team{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		Name:          "team a",
		Detail:        "detail",
		StartMemberID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		Members:       []Member{{ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"), Name: "m1"}},
		Parameter:     map[string]any{"k": "v"},
		DirectID:      uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		DirectHash:    "hash1",
		TMCreate:      &created,
	}

	got := h.ConvertWebhookMessage()

	if got.ID.String() != "11111111-1111-1111-1111-111111111111" || got.CustomerID.String() != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("identity not mapped: %v", got.Identity)
	}
	if got.Name != "team a" || got.Detail != "detail" || got.DirectHash != "hash1" {
		t.Errorf("scalar fields not mapped: %q %q %q", got.Name, got.Detail, got.DirectHash)
	}
	if got.StartMemberID.String() != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("StartMemberID = %s", got.StartMemberID)
	}
	if len(got.Members) != 1 || got.Members[0].Name != "m1" {
		t.Errorf("Members = %v", got.Members)
	}
	if got.Parameter["k"] != "v" {
		t.Errorf("Parameter = %v", got.Parameter)
	}
	if got.TMCreate == nil || !got.TMCreate.Equal(created) || got.TMUpdate != nil || got.TMDelete != nil {
		t.Errorf("timestamps not mapped: %v %v %v", got.TMCreate, got.TMUpdate, got.TMDelete)
	}
}

func Test_CreateWebhookEvent_excludesDirectID(t *testing.T) {
	h := &Team{
		Name:       "team a",
		DirectID:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
		DirectHash: "hash1",
	}

	data, err := h.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["name"] != "team a" || got["direct_hash"] != "hash1" {
		t.Errorf("unexpected payload: %v", got)
	}
	if _, ok := got["direct_id"]; ok {
		t.Errorf("direct_id must not be exposed: %v", got)
	}
}

func Test_ConvertResolvedAI(t *testing.T) {
	tests := []struct {
		name string
		in   *ai.AI
	}{
		{
			name: "full ai",
			in: &ai.AI{
				Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"), CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")},
				EngineModel: ai.EngineModelOpenaiGPT5,
				EngineKey:   "fake-engine-key-value",
				Parameter:   map[string]any{"temperature": 0.5},
				InitPrompt:  "hello",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "voice-1",
				STTType:     ai.STTTypeDeepgram,
				ToolNames:   []tool.ToolName{tool.ToolNameConnectCall},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertResolvedAI(tt.in)

			if got.ID.String() != "11111111-1111-1111-1111-111111111111" || got.CustomerID.String() != "22222222-2222-2222-2222-222222222222" {
				t.Errorf("identity = %v", got.Identity)
			}
			if got.EngineModel != ai.EngineModelOpenaiGPT5 {
				t.Errorf("EngineModel = %q", got.EngineModel)
			}
			if got.Parameter["temperature"] != 0.5 || got.InitPrompt != "hello" {
				t.Errorf("parameter/prompt = %v %q", got.Parameter, got.InitPrompt)
			}
			if got.TTSType != ai.TTSTypeElevenLabs || got.TTSVoiceID != "voice-1" || got.STTType != ai.STTTypeDeepgram {
				t.Errorf("tts/stt = %q %q %q", got.TTSType, got.TTSVoiceID, got.STTType)
			}
			if len(got.ToolNames) != 1 || got.ToolNames[0] != tool.ToolNameConnectCall {
				t.Errorf("ToolNames = %v", got.ToolNames)
			}

			// the engine key must never be serialized
			data, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if strings.Contains(string(data), "fake-engine-key-value") {
				t.Errorf("engine key leaked: %s", data)
			}
		})
	}
}
