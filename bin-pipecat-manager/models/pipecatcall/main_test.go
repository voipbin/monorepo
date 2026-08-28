package pipecatcall

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
)

// Pipecatcall carries the explicit subscription address of the global topic exchange
// (VOIP-1404/1419). The assertion pins the POINTER type: the event data reaches notifyhandler
// as a pointer and the eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Pipecatcall)(nil)

func TestPipecatcallEventSubscriptionID(t *testing.T) {
	ownID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	pc := &Pipecatcall{
		Identity: identity.Identity{
			ID:         ownID,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		ReferenceType: ReferenceTypeAICall,
		ReferenceID:   referenceID,
	}

	res := pc.EventSubscriptionID()
	if res != ownID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", ownID.String(), res)
	}

	// Mutation checks: the subscription address is the pipecatcall's OWN id — none of the other
	// id-shaped fields on the struct may ever leak into the routing key.
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. id: %s", customerID)
	}
	if res == activeflowID.String() {
		t.Errorf("Subscription address must not be the activeflow id. id: %s", activeflowID)
	}
	if res == referenceID.String() {
		t.Errorf("Subscription address must not be the reference id. id: %s", referenceID)
	}
}

func TestPipecatcallEventSubscriptionIDEmpty(t *testing.T) {
	pc := &Pipecatcall{}

	if res := pc.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}

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
			llmType:       LLMType("openai.gpt-5"),
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
