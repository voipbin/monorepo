package pipecatcall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestPipecatcall(t *testing.T) {
	tests := []struct {
		name string

		activeflowID  uuid.UUID
		referenceType ReferenceType
		referenceID   uuid.UUID
		hostID        string
		llmType       LLMType
		sttType       STTType
		sttLanguage   string
		ttsType       TTSType
		ttsLanguage   string
		ttsVoiceID    string
	}{
		{
			name: "creates_pipecatcall_with_all_fields",

			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			referenceType: ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			hostID:        "host-123",
			llmType:       LLMType("openai.gpt-4"),
			sttType:       STTTypeDeepgram,
			sttLanguage:   "en-US",
			ttsType:       TTSTypeCartesia,
			ttsLanguage:   "en-US",
			ttsVoiceID:    "voice-123",
		},
		{
			name: "creates_pipecatcall_with_ai_call_reference",

			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			referenceType: ReferenceTypeAICall,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			hostID:        "host-456",
			llmType:       LLMType("anthropic.claude-2"),
			sttType:       STTTypeDeepgram,
			sttLanguage:   "ko-KR",
			ttsType:       TTSTypeElevenLabs,
			ttsLanguage:   "ko-KR",
			ttsVoiceID:    "voice-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &Pipecatcall{
				ActiveflowID:  tt.activeflowID,
				ReferenceType: tt.referenceType,
				ReferenceID:   tt.referenceID,
				HostID:        tt.hostID,
				LLMType:       tt.llmType,
				STTType:       tt.sttType,
				STTLanguage:   tt.sttLanguage,
				TTSType:       tt.ttsType,
				TTSLanguage:   tt.ttsLanguage,
				TTSVoiceID:    tt.ttsVoiceID,
			}

			if pc.ActiveflowID != tt.activeflowID {
				t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", tt.activeflowID, pc.ActiveflowID)
			}
			if pc.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, pc.ReferenceType)
			}
			if pc.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, pc.ReferenceID)
			}
			if pc.HostID != tt.hostID {
				t.Errorf("Wrong HostID. expect: %s, got: %s", tt.hostID, pc.HostID)
			}
			if pc.LLMType != tt.llmType {
				t.Errorf("Wrong LLMType. expect: %s, got: %s", tt.llmType, pc.LLMType)
			}
			if pc.STTType != tt.sttType {
				t.Errorf("Wrong STTType. expect: %s, got: %s", tt.sttType, pc.STTType)
			}
			if pc.STTLanguage != tt.sttLanguage {
				t.Errorf("Wrong STTLanguage. expect: %s, got: %s", tt.sttLanguage, pc.STTLanguage)
			}
			if pc.TTSType != tt.ttsType {
				t.Errorf("Wrong TTSType. expect: %s, got: %s", tt.ttsType, pc.TTSType)
			}
			if pc.TTSLanguage != tt.ttsLanguage {
				t.Errorf("Wrong TTSLanguage. expect: %s, got: %s", tt.ttsLanguage, pc.TTSLanguage)
			}
			if pc.TTSVoiceID != tt.ttsVoiceID {
				t.Errorf("Wrong TTSVoiceID. expect: %s, got: %s", tt.ttsVoiceID, pc.TTSVoiceID)
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
			name:     "reference_type_call",
			constant: ReferenceTypeCall,
			expected: "call",
		},
		{
			name:     "reference_type_ai_call",
			constant: ReferenceTypeAICall,
			expected: "ai_call",
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

func TestSTTTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant STTType
		expected string
	}{
		{
			name:     "stt_type_none",
			constant: STTTypeNone,
			expected: "",
		},
		{
			name:     "stt_type_deepgram",
			constant: STTTypeDeepgram,
			expected: "deepgram",
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

func TestTTSTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TTSType
		expected string
	}{
		{
			name:     "tts_type_none",
			constant: TTSTypeNone,
			expected: "",
		},
		{
			name:     "tts_type_cartesia",
			constant: TTSTypeCartesia,
			expected: "cartesia",
		},
		{
			name:     "tts_type_elevenlabs",
			constant: TTSTypeElevenLabs,
			expected: "elevenlabs",
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
