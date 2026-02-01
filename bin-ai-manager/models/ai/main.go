package ai

import (
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/models/identity"
	"strings"
)

// AI define
type AI struct {
	identity.Identity

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	EngineType  EngineType     `json:"engine_type,omitempty" db:"engine_type"`   // currently not in used.
	EngineModel EngineModel    `json:"engine_model,omitempty" db:"engine_model"` // ai(llm) model. combine with <engine model target>.<model>
	EngineData  map[string]any `json:"engine_data,omitempty" db:"engine_data,json"`
	EngineKey   string         `json:"engine_key,omitempty" db:"engine_key"` // ai(llm) service api key

	InitPrompt string `json:"init_prompt,omitempty" db:"init_prompt"`

	TTSType    TTSType `json:"tts_type,omitempty" db:"tts_type"`
	TTSVoiceID string  `json:"tts_voice_id,omitempty" db:"tts_voice_id"`

	STTType STTType `json:"stt_type,omitempty" db:"stt_type"`

	// ToolNames defines which tools are enabled for this AI
	// ["all"] = all tools, ["connect_call", "send_email"] = specific tools, [] or nil = no tools
	ToolNames []tool.ToolName `json:"tool_names,omitempty" db:"tool_names,json"`

	// timestamp
	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

// EngineType define
type EngineType string

// list of engine types
const (
	EngineTypeNone EngineType = ""
)

type EngineModelTarget string

const (
	EngineModelTargetNone       EngineModelTarget = ""
	EngineModelTargetDialogflow EngineModelTarget = "dialogflow" // dialogflow use

	EngineModelTargetAnthropic  EngineModelTarget = "anthropic"  // Anthropic Claude
	EngineModelTargetAWS        EngineModelTarget = "aws"        // AWS Bedrock
	EngineModelTargetAzure      EngineModelTarget = "azure"      // Azure OpenAI
	EngineModelTargetCerebras   EngineModelTarget = "cerebras"   // Cerebras
	EngineModelTargetDeepSeek   EngineModelTarget = "deepseek"   // DeepSeek
	EngineModelTargetFireworks  EngineModelTarget = "fireworks"  // Fireworks AI
	EngineModelTargetGemini     EngineModelTarget = "gemini"     // Google Gemini
	EngineModelTargetGrok       EngineModelTarget = "grok"       // Grok
	EngineModelTargetGroq       EngineModelTarget = "groq"       // Groq
	EngineModelTargetMistral    EngineModelTarget = "mistral"    // Mistral AI
	EngineModelTargetNvidiaNIM  EngineModelTarget = "nvidia"     // NVIDIA NIM
	EngineModelTargetOllama     EngineModelTarget = "ollama"     // Ollama
	EngineModelTargetOpenAI     EngineModelTarget = "openai"     // OpenAI
	EngineModelTargetOpenRouter EngineModelTarget = "openrouter" // OpenRouter
	EngineModelTargetPerplexity EngineModelTarget = "perplexity" // Perplexity
	EngineModelTargetQwen       EngineModelTarget = "qwen"       // Qwen
	EngineModelTargetSambaNova  EngineModelTarget = "sambanova"  // SambaNova
	EngineModelTargetTogetherAI EngineModelTarget = "together"   // Together AI
)

var EngineModelTargets = []EngineModelTarget{
	EngineModelTargetDialogflow,

	EngineModelTargetAnthropic,
	EngineModelTargetAWS,
	EngineModelTargetAzure,
	EngineModelTargetCerebras,
	EngineModelTargetDeepSeek,
	EngineModelTargetFireworks,
	EngineModelTargetGemini,
	EngineModelTargetGrok,
	EngineModelTargetGroq,
	EngineModelTargetMistral,
	EngineModelTargetNvidiaNIM,
	EngineModelTargetOllama,
	EngineModelTargetOpenAI,
	EngineModelTargetOpenRouter,
	EngineModelTargetPerplexity,
	EngineModelTargetQwen,
	EngineModelTargetSambaNova,
	EngineModelTargetTogetherAI,
}

type EngineModel string

// list of engine models
const (
	EngineModelOpenaiO1Mini            EngineModel = "openai.o1-mini"
	EngineModelOpenaiO1Preview         EngineModel = "openai.o1-preview"
	EngineModelOpenaiO1                EngineModel = "openai.o1"
	EngineModelOpenaiO3Mini            EngineModel = "openai.o3-mini"
	EngineModelOpenaiGPT4O             EngineModel = "openai.gpt-4o"
	EngineModelOpenaiGPT4OMini         EngineModel = "openai.gpt-4o-mini"
	EngineModelOpenaiGPT4Turbo         EngineModel = "openai.gpt-4-turbo"
	EngineModelOpenaiGPT4VisionPreview EngineModel = "openai.gpt-4-vision-preview"
	EngineModelOpenaiGPT4              EngineModel = "openai.gpt-4"
	EngineModelOpenaiGPT3Dot5Turbo     EngineModel = "openai.gpt-3.5-turbo"

	EngineModelDialogflowCX EngineModel = "dialogflow.cx"
	EngineModelDialogflowES EngineModel = "dialogflow.es"
)

func GetEngineModelTarget(engineModel EngineModel) EngineModelTarget {
	mapModelTarget := map[EngineModel]EngineModelTarget{
		EngineModelOpenaiO1Mini:            EngineModelTargetOpenAI,
		EngineModelOpenaiO1Preview:         EngineModelTargetOpenAI,
		EngineModelOpenaiO1:                EngineModelTargetOpenAI,
		EngineModelOpenaiO3Mini:            EngineModelTargetOpenAI,
		EngineModelOpenaiGPT4O:             EngineModelTargetOpenAI,
		EngineModelOpenaiGPT4OMini:         EngineModelTargetOpenAI,
		EngineModelOpenaiGPT4Turbo:         EngineModelTargetOpenAI,
		EngineModelOpenaiGPT4VisionPreview: EngineModelTargetOpenAI,
		EngineModelOpenaiGPT4:              EngineModelTargetOpenAI,
		EngineModelOpenaiGPT3Dot5Turbo:     EngineModelTargetOpenAI,

		EngineModelDialogflowCX: EngineModelTargetDialogflow,
		EngineModelDialogflowES: EngineModelTargetDialogflow,
	}

	res, ok := mapModelTarget[engineModel]
	if !ok {
		return EngineModelTargetNone
	}

	return res
}

func GetEngineModelName(engineModel EngineModel) string {
	tmp := strings.Split(string(engineModel), ".")
	if len(tmp) < 2 {
		return ""
	}
	return tmp[1]
}

func IsValidEngineModel(engineModel EngineModel) bool {
	tmp := strings.Split(string(engineModel), ".")
	if len(tmp) < 2 {
		return false
	}

	for _, target := range EngineModelTargets {
		if EngineModelTarget(tmp[0]) == target {
			return true
		}
	}

	return false
}

// TTSType define
type TTSType string

const (
	TTSTypeNone       TTSType = ""
	TTSTypeAsync      TTSType = "async"       // Generic async TTS adapter
	TTSTypeAWS        TTSType = "aws"         // AWS Polly or Bedrock TTS
	TTSTypeAzure      TTSType = "azure"       // Azure Cognitive TTS
	TTSTypeCartesia   TTSType = "cartesia"    // Cartesia TTS
	TTSTypeDeepgram   TTSType = "deepgram"    // Deepgram TTS
	TTSTypeElevenLabs TTSType = "elevenlabs"  // ElevenLabs TTS
	TTSTypeFish       TTSType = "fish"        // Fish TTS (experimental)
	TTSTypeGoogle     TTSType = "google"      // Google Cloud TTS
	TTSTypeGroq       TTSType = "groq"        // Groq TTS (fast inference)
	TTSTypeHume       TTSType = "hume"        // Hume AI TTS (emotion-driven)
	TTSTypeInworld    TTSType = "inworld"     // Inworld TTS (character voices)
	TTSTypeLMNT       TTSType = "lmnt"        // LMNT TTS
	TTSTypeMiniMax    TTSType = "minimax"     // MiniMax TTS
	TTSTypeNeuphonic  TTSType = "neuphonic"   // Neuphonic TTS
	TTSTypeNvidiaRiva TTSType = "nvidia-riva" // NVIDIA Riva TTS
	TTSTypeOpenAI     TTSType = "openai"      // OpenAI TTS (e.g., tts-1)
	TTSTypePiper      TTSType = "piper"       // Piper open-source TTS
	TTSTypePlayHT     TTSType = "playht"      // PlayHT TTS
	TTSTypeRime       TTSType = "rime"        // Rime TTS
	TTSTypeSarvam     TTSType = "sarvam"      // Sarvam AI TTS
	TTSTypeXTTS       TTSType = "xtts"        // XTTS (cross-lingual TTS)
)

// STTType define
type STTType string

const (
	STTTypeNone       STTType = ""
	STTTypeCartesia   STTType = "cartesia"
	STTTypeDeepgram   STTType = "deepgram"
	STTTypeElevenLabs STTType = "elevenlabs"
)
