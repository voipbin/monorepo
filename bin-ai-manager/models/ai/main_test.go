package ai

import (
	"testing"
)

func TestAI(t *testing.T) {
	tests := []struct {
		name string

		aiName      string
		detail      string
		engineModel EngineModel
		engineKey   string
		initPrompt  string
		ttsType     TTSType
		ttsVoiceID  string
		sttType     STTType
	}{
		{
			name: "creates_ai_with_all_fields",

			aiName:      "Test AI Agent",
			detail:      "A test AI agent for unit testing",
			engineModel: EngineModelOpenaiGPT5,
			engineKey:   "sk-test-key",
			initPrompt:  "You are a helpful assistant.",
			ttsType:     TTSTypeElevenLabs,
			ttsVoiceID:  "voice-123",
			sttType:     STTTypeDeepgram,
		},
		{
			name: "creates_ai_with_empty_fields",

			aiName:      "",
			detail:      "",
			engineModel: "",
			engineKey:   "",
			initPrompt:  "",
			ttsType:     "",
			ttsVoiceID:  "",
			sttType:     "",
		},
		{
			name: "creates_ai_with_dialogflow_engine",

			aiName:      "Dialogflow Agent",
			detail:      "A Dialogflow-powered agent",
			engineModel: EngineModelGeminiGemini2Dot5Flash,
			engineKey:   "dialogflow-key",
			initPrompt:  "",
			ttsType:     TTSTypeGoogle,
			ttsVoiceID:  "",
			sttType:     STTTypeCartesia,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AI{
				Name:        tt.aiName,
				Detail:      tt.detail,
				EngineModel: tt.engineModel,
				EngineKey:   tt.engineKey,
				InitPrompt:  tt.initPrompt,
				TTSType:     tt.ttsType,
				TTSVoiceID:  tt.ttsVoiceID,
				STTType:     tt.sttType,
			}

			if a.Name != tt.aiName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.aiName, a.Name)
			}
			if a.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, a.Detail)
			}
			if a.EngineModel != tt.engineModel {
				t.Errorf("Wrong EngineModel. expect: %s, got: %s", tt.engineModel, a.EngineModel)
			}
			if a.EngineKey != tt.engineKey {
				t.Errorf("Wrong EngineKey. expect: %s, got: %s", tt.engineKey, a.EngineKey)
			}
			if a.InitPrompt != tt.initPrompt {
				t.Errorf("Wrong InitPrompt. expect: %s, got: %s", tt.initPrompt, a.InitPrompt)
			}
			if a.TTSType != tt.ttsType {
				t.Errorf("Wrong TTSType. expect: %s, got: %s", tt.ttsType, a.TTSType)
			}
			if a.TTSVoiceID != tt.ttsVoiceID {
				t.Errorf("Wrong TTSVoiceID. expect: %s, got: %s", tt.ttsVoiceID, a.TTSVoiceID)
			}
			if a.STTType != tt.sttType {
				t.Errorf("Wrong STTType. expect: %s, got: %s", tt.sttType, a.STTType)
			}
		})
	}
}

func TestEngineModelTargetConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EngineModelTarget
		expected string
	}{
		{
			name:     "engine_model_target_none",
			constant: EngineModelTargetNone,
			expected: "",
		},
		{
			name:     "engine_model_target_gemini",
			constant: EngineModelTargetGemini,
			expected: "gemini",
		},
		{
			name:     "engine_model_target_deepseek",
			constant: EngineModelTargetDeepSeek,
			expected: "deepseek",
		},
		{
			name:     "engine_model_target_anthropic",
			constant: EngineModelTargetAnthropic,
			expected: "anthropic",
		},
		{
			name:     "engine_model_target_aws",
			constant: EngineModelTargetAWS,
			expected: "aws",
		},
		{
			name:     "engine_model_target_azure",
			constant: EngineModelTargetAzure,
			expected: "azure",
		},
		{
			name:     "engine_model_target_openai",
			constant: EngineModelTargetOpenAI,
			expected: "openai",
		},
		{
			name:     "engine_model_target_groq",
			constant: EngineModelTargetGroq,
			expected: "groq",
		},
		{
			name:     "engine_model_target_grok",
			constant: EngineModelTargetGrok,
			expected: "grok",
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

func TestEngineModelConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EngineModel
		expected string
	}{
		{
			name:     "engine_model_gemini_2_5_flash",
			constant: EngineModelGeminiGemini2Dot5Flash,
			expected: "gemini.gemini-2.5-flash",
		},
		{
			name:     "engine_model_gemini_2_5_pro",
			constant: EngineModelGeminiGemini2Dot5Pro,
			expected: "gemini.gemini-2.5-pro",
		},
		{
			name:     "engine_model_gemini_2_0_flash",
			constant: EngineModelGeminiGemini2Dot0Flash,
			expected: "gemini.gemini-2.0-flash",
		},
		{
			name:     "engine_model_gemini_pro_latest",
			constant: EngineModelGeminiGeminiProLatest,
			expected: "gemini.gemini-pro-latest",
		},
		{
			name:     "engine_model_openai_gpt5_dot2",
			constant: EngineModelOpenaiGPT5Dot2,
			expected: "openai.gpt-5.2",
		},
		{
			name:     "engine_model_openai_gpt5_dot1",
			constant: EngineModelOpenaiGPT5Dot1,
			expected: "openai.gpt-5.1",
		},
		{
			name:     "engine_model_openai_gpt5",
			constant: EngineModelOpenaiGPT5,
			expected: "openai.gpt-5",
		},
		{
			name:     "engine_model_openai_gpt5_mini",
			constant: EngineModelOpenaiGPT5Mini,
			expected: "openai.gpt-5-mini",
		},
		{
			name:     "engine_model_openai_gpt5_nano",
			constant: EngineModelOpenaiGPT5Nano,
			expected: "openai.gpt-5-nano",
		},
		{
			name:     "engine_model_grok3",
			constant: EngineModelGrok3,
			expected: "grok.grok-3",
		},
		{
			name:     "engine_model_grok3_mini",
			constant: EngineModelGrok3Mini,
			expected: "grok.grok-3-mini",
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

func TestGetEngineModelTarget(t *testing.T) {
	tests := []struct {
		name        string
		engineModel EngineModel
		expected    EngineModelTarget
	}{
		{
			name:        "openai_gpt5_returns_openai",
			engineModel: EngineModelOpenaiGPT5,
			expected:    EngineModelTargetOpenAI,
		},
		{
			name:        "openai_gpt5_mini_returns_openai",
			engineModel: EngineModelOpenaiGPT5Mini,
			expected:    EngineModelTargetOpenAI,
		},
		{
			name:        "gemini_2_5_flash_returns_gemini",
			engineModel: EngineModelGeminiGemini2Dot5Flash,
			expected:    EngineModelTargetGemini,
		},
		{
			name:        "gemini_pro_latest_returns_gemini",
			engineModel: EngineModelGeminiGeminiProLatest,
			expected:    EngineModelTargetGemini,
		},
		{
			name:        "unknown_model_returns_none",
			engineModel: EngineModel("unknown.model"),
			expected:    EngineModelTargetNone,
		},
		{
			name:        "grok3_returns_grok",
			engineModel: EngineModelGrok3,
			expected:    EngineModelTargetGrok,
		},
		{
			name:        "grok3_mini_returns_grok",
			engineModel: EngineModelGrok3Mini,
			expected:    EngineModelTargetGrok,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := GetEngineModelTarget(tt.engineModel)
			if res != tt.expected {
				t.Errorf("Wrong target. expect: %s, got: %s", tt.expected, res)
			}
		})
	}
}

func TestGetEngineModelName(t *testing.T) {
	tests := []struct {
		name        string
		engineModel EngineModel
		expected    string
	}{
		{
			name:        "openai_gpt5_returns_gpt5",
			engineModel: EngineModelOpenaiGPT5,
			expected:    "gpt-5",
		},
		{
			name:        "gemini_2_5_flash_returns_name",
			engineModel: EngineModelGeminiGemini2Dot5Flash,
			expected:    "gemini-2.5-flash",
		},
		{
			name:        "invalid_model_returns_empty",
			engineModel: EngineModel("invalid"),
			expected:    "",
		},
		{
			name:        "grok3_returns_grok3",
			engineModel: EngineModelGrok3,
			expected:    "grok-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := GetEngineModelName(tt.engineModel)
			if res != tt.expected {
				t.Errorf("Wrong model name. expect: %s, got: %s", tt.expected, res)
			}
		})
	}
}

func TestIsValidEngineModel(t *testing.T) {
	tests := []struct {
		name        string
		engineModel EngineModel
		expected    bool
	}{
		{
			name:        "openai_gpt5_is_valid",
			engineModel: EngineModelOpenaiGPT5,
			expected:    true,
		},
		{
			name:        "gemini_2_5_flash_is_valid",
			engineModel: EngineModelGeminiGemini2Dot5Flash,
			expected:    true,
		},
		{
			name:        "anthropic_model_is_valid",
			engineModel: EngineModel("anthropic.claude-3"),
			expected:    true,
		},
		{
			name:        "invalid_model_no_dot",
			engineModel: EngineModel("invalid"),
			expected:    false,
		},
		{
			name:        "invalid_target",
			engineModel: EngineModel("unknown.model"),
			expected:    false,
		},
		{
			name:        "grok_model_is_valid",
			engineModel: EngineModel("grok.grok-3"),
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsValidEngineModel(tt.engineModel)
			if res != tt.expected {
				t.Errorf("Wrong result. expect: %v, got: %v", tt.expected, res)
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
			name:     "tts_type_elevenlabs",
			constant: TTSTypeElevenLabs,
			expected: "elevenlabs",
		},
		{
			name:     "tts_type_google",
			constant: TTSTypeGoogle,
			expected: "google",
		},
		{
			name:     "tts_type_openai",
			constant: TTSTypeOpenAI,
			expected: "openai",
		},
		{
			name:     "tts_type_deepgram",
			constant: TTSTypeDeepgram,
			expected: "deepgram",
		},
		{
			name:     "tts_type_cartesia",
			constant: TTSTypeCartesia,
			expected: "cartesia",
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
			name:     "stt_type_cartesia",
			constant: STTTypeCartesia,
			expected: "cartesia",
		},
		{
			name:     "stt_type_deepgram",
			constant: STTTypeDeepgram,
			expected: "deepgram",
		},
		{
			name:     "stt_type_elevenlabs",
			constant: STTTypeElevenLabs,
			expected: "elevenlabs",
		},
		{
			name:     "stt_type_google",
			constant: STTTypeGoogle,
			expected: "google",
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

func TestTTSTypeIsValid(t *testing.T) {
	tests := []struct {
		name     string
		ttsType  TTSType
		expected bool
	}{
		{"empty_string_is_valid", TTSTypeNone, true},
		{"google_is_valid", TTSTypeGoogle, true},
		{"openai_is_valid", TTSTypeOpenAI, true},
		{"elevenlabs_is_valid", TTSTypeElevenLabs, true},
		{"cartesia_is_valid", TTSTypeCartesia, true},
		{"deepgram_is_valid", TTSTypeDeepgram, true},
		{"aws_is_valid", TTSTypeAWS, true},
		{"azure_is_valid", TTSTypeAzure, true},
		{"async_is_valid", TTSTypeAsync, true},
		{"fish_is_valid", TTSTypeFish, true},
		{"groq_is_valid", TTSTypeGroq, true},
		{"hume_is_valid", TTSTypeHume, true},
		{"inworld_is_valid", TTSTypeInworld, true},
		{"lmnt_is_valid", TTSTypeLMNT, true},
		{"minimax_is_valid", TTSTypeMiniMax, true},
		{"neuphonic_is_valid", TTSTypeNeuphonic, true},
		{"nvidia_riva_is_valid", TTSTypeNvidiaRiva, true},
		{"piper_is_valid", TTSTypePiper, true},
		{"playht_is_valid", TTSTypePlayHT, true},
		{"rime_is_valid", TTSTypeRime, true},
		{"sarvam_is_valid", TTSTypeSarvam, true},
		{"xtts_is_valid", TTSTypeXTTS, true},
		{"gcp_is_invalid", TTSType("gcp"), false},
		{"random_string_is_invalid", TTSType("random"), false},
		{"polly_is_invalid", TTSType("polly"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttsType.IsValid() != tt.expected {
				t.Errorf("TTSType(%q).IsValid() = %v, want %v", tt.ttsType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestTTSTypeValidValues(t *testing.T) {
	values := TTSTypeNone.ValidValues()

	if len(values) == 0 {
		t.Fatal("ValidValues() returned empty slice")
	}

	// Should not contain empty string
	for _, v := range values {
		if v == "" {
			t.Error("ValidValues() should not contain empty string")
		}
	}

	// Should be sorted
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			t.Errorf("ValidValues() not sorted: %q comes after %q", values[i], values[i-1])
		}
	}

	// Should contain known values
	knownValues := map[string]bool{
		"google": false, "openai": false, "elevenlabs": false, "cartesia": false,
	}
	for _, v := range values {
		if _, ok := knownValues[v]; ok {
			knownValues[v] = true
		}
	}
	for k, found := range knownValues {
		if !found {
			t.Errorf("ValidValues() missing expected value: %q", k)
		}
	}
}

func TestSTTTypeIsValid(t *testing.T) {
	tests := []struct {
		name     string
		sttType  STTType
		expected bool
	}{
		{"empty_string_is_valid", STTTypeNone, true},
		{"cartesia_is_valid", STTTypeCartesia, true},
		{"deepgram_is_valid", STTTypeDeepgram, true},
		{"elevenlabs_is_valid", STTTypeElevenLabs, true},
		{"google_is_valid", STTTypeGoogle, true},
		{"gcp_is_invalid", STTType("gcp"), false},
		{"random_string_is_invalid", STTType("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sttType.IsValid() != tt.expected {
				t.Errorf("STTType(%q).IsValid() = %v, want %v", tt.sttType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestSTTTypeValidValues(t *testing.T) {
	values := STTTypeNone.ValidValues()

	if len(values) == 0 {
		t.Fatal("ValidValues() returned empty slice")
	}

	// Should not contain empty string
	for _, v := range values {
		if v == "" {
			t.Error("ValidValues() should not contain empty string")
		}
	}

	// Should be sorted
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			t.Errorf("ValidValues() not sorted: %q comes after %q", values[i], values[i-1])
		}
	}

	// Should contain known values
	knownValues := map[string]bool{
		"deepgram": false, "cartesia": false, "elevenlabs": false, "google": false,
	}
	for _, v := range values {
		if _, ok := knownValues[v]; ok {
			knownValues[v] = true
		}
	}
	for k, found := range knownValues {
		if !found {
			t.Errorf("ValidValues() missing expected value: %q", k)
		}
	}
}
