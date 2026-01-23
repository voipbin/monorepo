package ai

import (
	"testing"
)

func TestAI(t *testing.T) {
	tests := []struct {
		name string

		aiName      string
		detail      string
		engineType  EngineType
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
			engineType:  EngineTypeNone,
			engineModel: EngineModelOpenaiGPT4O,
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
			engineType:  "",
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
			engineType:  EngineTypeNone,
			engineModel: EngineModelDialogflowCX,
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
				EngineType:  tt.engineType,
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
			if a.EngineType != tt.engineType {
				t.Errorf("Wrong EngineType. expect: %s, got: %s", tt.engineType, a.EngineType)
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

func TestEngineTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EngineType
		expected string
	}{
		{
			name:     "engine_type_none",
			constant: EngineTypeNone,
			expected: "",
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
			name:     "engine_model_target_dialogflow",
			constant: EngineModelTargetDialogflow,
			expected: "dialogflow",
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
			name:     "engine_model_target_gemini",
			constant: EngineModelTargetGemini,
			expected: "gemini",
		},
		{
			name:     "engine_model_target_groq",
			constant: EngineModelTargetGroq,
			expected: "groq",
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
			name:     "engine_model_openai_gpt4o",
			constant: EngineModelOpenaiGPT4O,
			expected: "openai.gpt-4o",
		},
		{
			name:     "engine_model_openai_gpt4o_mini",
			constant: EngineModelOpenaiGPT4OMini,
			expected: "openai.gpt-4o-mini",
		},
		{
			name:     "engine_model_openai_gpt4_turbo",
			constant: EngineModelOpenaiGPT4Turbo,
			expected: "openai.gpt-4-turbo",
		},
		{
			name:     "engine_model_openai_gpt4",
			constant: EngineModelOpenaiGPT4,
			expected: "openai.gpt-4",
		},
		{
			name:     "engine_model_openai_gpt3_5_turbo",
			constant: EngineModelOpenaiGPT3Dot5Turbo,
			expected: "openai.gpt-3.5-turbo",
		},
		{
			name:     "engine_model_dialogflow_cx",
			constant: EngineModelDialogflowCX,
			expected: "dialogflow.cx",
		},
		{
			name:     "engine_model_dialogflow_es",
			constant: EngineModelDialogflowES,
			expected: "dialogflow.es",
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
			name:        "openai_gpt4o_returns_openai",
			engineModel: EngineModelOpenaiGPT4O,
			expected:    EngineModelTargetOpenAI,
		},
		{
			name:        "openai_gpt4o_mini_returns_openai",
			engineModel: EngineModelOpenaiGPT4OMini,
			expected:    EngineModelTargetOpenAI,
		},
		{
			name:        "dialogflow_cx_returns_dialogflow",
			engineModel: EngineModelDialogflowCX,
			expected:    EngineModelTargetDialogflow,
		},
		{
			name:        "dialogflow_es_returns_dialogflow",
			engineModel: EngineModelDialogflowES,
			expected:    EngineModelTargetDialogflow,
		},
		{
			name:        "unknown_model_returns_none",
			engineModel: EngineModel("unknown.model"),
			expected:    EngineModelTargetNone,
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
			name:        "openai_gpt4o_returns_gpt4o",
			engineModel: EngineModelOpenaiGPT4O,
			expected:    "gpt-4o",
		},
		{
			name:        "dialogflow_cx_returns_cx",
			engineModel: EngineModelDialogflowCX,
			expected:    "cx",
		},
		{
			name:        "invalid_model_returns_empty",
			engineModel: EngineModel("invalid"),
			expected:    "",
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
			name:        "openai_gpt4o_is_valid",
			engineModel: EngineModelOpenaiGPT4O,
			expected:    true,
		},
		{
			name:        "dialogflow_cx_is_valid",
			engineModel: EngineModelDialogflowCX,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
