package aicall

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
)

func TestAIcall(t *testing.T) {
	tests := []struct {
		name string

		aiID          uuid.UUID
		aiEngineType  ai.EngineType
		aiEngineModel ai.EngineModel
		aiTTSType     ai.TTSType
		aiTTSVoiceID  string
		aiSTTType     ai.STTType
		activeflowID  uuid.UUID
		referenceType ReferenceType
		referenceID   uuid.UUID
		confbridgeID  uuid.UUID
		pipecatcallID uuid.UUID
		status        Status
		gender        Gender
		language      string
	}{
		{
			name: "creates_aicall_with_all_fields",

			aiID:          uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			aiEngineType:  ai.EngineTypeNone,
			aiEngineModel: ai.EngineModelOpenaiGPT4O,
			aiTTSType:     ai.TTSTypeElevenLabs,
			aiTTSVoiceID:  "voice-123",
			aiSTTType:     ai.STTTypeDeepgram,
			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			referenceType: ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			confbridgeID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			pipecatcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
			status:        StatusProgressing,
			gender:        GenderFemale,
			language:      "en-US",
		},
		{
			name: "creates_aicall_with_empty_fields",

			aiID:          uuid.Nil,
			aiEngineType:  "",
			aiEngineModel: "",
			aiTTSType:     "",
			aiTTSVoiceID:  "",
			aiSTTType:     "",
			activeflowID:  uuid.Nil,
			referenceType: ReferenceTypeNone,
			referenceID:   uuid.Nil,
			confbridgeID:  uuid.Nil,
			pipecatcallID: uuid.Nil,
			status:        "",
			gender:        GenderNone,
			language:      "",
		},
		{
			name: "creates_aicall_for_conversation",

			aiID:          uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
			aiEngineType:  ai.EngineTypeNone,
			aiEngineModel: ai.EngineModelDialogflowCX,
			aiTTSType:     ai.TTSTypeGoogle,
			aiTTSVoiceID:  "",
			aiSTTType:     ai.STTTypeCartesia,
			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
			referenceType: ReferenceTypeConversation,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440008"),
			confbridgeID:  uuid.Nil,
			pipecatcallID: uuid.Nil,
			status:        StatusInitiating,
			gender:        GenderMale,
			language:      "ko-KR",
		},
		{
			name: "creates_aicall_for_task",

			aiID:          uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440009"),
			aiEngineType:  ai.EngineTypeNone,
			aiEngineModel: ai.EngineModelOpenaiGPT4OMini,
			aiTTSType:     ai.TTSTypeNone,
			aiTTSVoiceID:  "",
			aiSTTType:     ai.STTTypeNone,
			activeflowID:  uuid.Nil,
			referenceType: ReferenceTypeTask,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440010"),
			confbridgeID:  uuid.Nil,
			pipecatcallID: uuid.Nil,
			status:        StatusTerminated,
			gender:        GenderNeutral,
			language:      "ja-JP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AIcall{
				AIID:          tt.aiID,
				AIEngineType:  tt.aiEngineType,
				AIEngineModel: tt.aiEngineModel,
				AITTSType:     tt.aiTTSType,
				AITTSVoiceID:  tt.aiTTSVoiceID,
				AISTTType:     tt.aiSTTType,
				ActiveflowID:  tt.activeflowID,
				ReferenceType: tt.referenceType,
				ReferenceID:   tt.referenceID,
				ConfbridgeID:  tt.confbridgeID,
				PipecatcallID: tt.pipecatcallID,
				Status:        tt.status,
				Gender:        tt.gender,
				Language:      tt.language,
			}

			if ac.AIID != tt.aiID {
				t.Errorf("Wrong AIID. expect: %s, got: %s", tt.aiID, ac.AIID)
			}
			if ac.AIEngineType != tt.aiEngineType {
				t.Errorf("Wrong AIEngineType. expect: %s, got: %s", tt.aiEngineType, ac.AIEngineType)
			}
			if ac.AIEngineModel != tt.aiEngineModel {
				t.Errorf("Wrong AIEngineModel. expect: %s, got: %s", tt.aiEngineModel, ac.AIEngineModel)
			}
			if ac.AITTSType != tt.aiTTSType {
				t.Errorf("Wrong AITTSType. expect: %s, got: %s", tt.aiTTSType, ac.AITTSType)
			}
			if ac.AITTSVoiceID != tt.aiTTSVoiceID {
				t.Errorf("Wrong AITTSVoiceID. expect: %s, got: %s", tt.aiTTSVoiceID, ac.AITTSVoiceID)
			}
			if ac.AISTTType != tt.aiSTTType {
				t.Errorf("Wrong AISTTType. expect: %s, got: %s", tt.aiSTTType, ac.AISTTType)
			}
			if ac.ActiveflowID != tt.activeflowID {
				t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", tt.activeflowID, ac.ActiveflowID)
			}
			if ac.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, ac.ReferenceType)
			}
			if ac.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, ac.ReferenceID)
			}
			if ac.ConfbridgeID != tt.confbridgeID {
				t.Errorf("Wrong ConfbridgeID. expect: %s, got: %s", tt.confbridgeID, ac.ConfbridgeID)
			}
			if ac.PipecatcallID != tt.pipecatcallID {
				t.Errorf("Wrong PipecatcallID. expect: %s, got: %s", tt.pipecatcallID, ac.PipecatcallID)
			}
			if ac.Status != tt.status {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.status, ac.Status)
			}
			if ac.Gender != tt.gender {
				t.Errorf("Wrong Gender. expect: %s, got: %s", tt.gender, ac.Gender)
			}
			if ac.Language != tt.language {
				t.Errorf("Wrong Language. expect: %s, got: %s", tt.language, ac.Language)
			}
		})
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{
			name:     "reference_type_none",
			constant: ReferenceTypeNone,
			expected: "",
		},
		{
			name:     "reference_type_call",
			constant: ReferenceTypeCall,
			expected: "call",
		},
		{
			name:     "reference_type_conversation",
			constant: ReferenceTypeConversation,
			expected: "conversation",
		},
		{
			name:     "reference_type_task",
			constant: ReferenceTypeTask,
			expected: "task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{
			name:     "status_initiating",
			constant: StatusInitiating,
			expected: "initiating",
		},
		{
			name:     "status_progressing",
			constant: StatusProgressing,
			expected: "progressing",
		},
		{
			name:     "status_pausing",
			constant: StatusPausing,
			expected: "pausing",
		},
		{
			name:     "status_resuming",
			constant: StatusResuming,
			expected: "resuming",
		},
		{
			name:     "status_terminating",
			constant: StatusTerminating,
			expected: "terminating",
		},
		{
			name:     "status_terminated",
			constant: StatusTerminated,
			expected: "terminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestGenderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Gender
		expected string
	}{
		{
			name:     "gender_none",
			constant: GenderNone,
			expected: "",
		},
		{
			name:     "gender_male",
			constant: GenderMale,
			expected: "male",
		},
		{
			name:     "gender_female",
			constant: GenderFemale,
			expected: "female",
		},
		{
			name:     "gender_neutral",
			constant: GenderNeutral,
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name string

		role    MessageRole
		content string
	}{
		{
			name: "creates_message_with_user_role",

			role:    MessageRoleUser,
			content: "Hello, how can you help me?",
		},
		{
			name: "creates_message_with_assistant_role",

			role:    MessageRoleAssistant,
			content: "I can help you with many tasks.",
		},
		{
			name: "creates_message_with_system_role",

			role:    MessageRoleSystem,
			content: "You are a helpful assistant.",
		},
		{
			name: "creates_message_with_tool_role",

			role:    MessageRoleTool,
			content: `{"result": "success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				Role:    tt.role,
				Content: tt.content,
			}

			if m.Role != tt.role {
				t.Errorf("Wrong Role. expect: %s, got: %s", tt.role, m.Role)
			}
			if m.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, m.Content)
			}
		})
	}
}

func TestMessageRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant MessageRole
		expected string
	}{
		{
			name:     "message_role_system",
			constant: MessageRoleSystem,
			expected: "system",
		},
		{
			name:     "message_role_user",
			constant: MessageRoleUser,
			expected: "user",
		},
		{
			name:     "message_role_assistant",
			constant: MessageRoleAssistant,
			expected: "assistant",
		},
		{
			name:     "message_role_function",
			constant: MessageRoleFunction,
			expected: "function",
		},
		{
			name:     "message_role_tool",
			constant: MessageRoleTool,
			expected: "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
