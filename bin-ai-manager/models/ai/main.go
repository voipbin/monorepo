package ai

import (
	"sort"
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/models/identity"
)

// AI define
type AI struct {
	identity.Identity

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	EngineModel EngineModel    `json:"engine_model,omitempty" db:"engine_model"` // ai(llm) model. combine with <engine model target>.<model>
	Parameter   map[string]any `json:"parameter,omitempty" db:"parameter,json"`
	EngineKey   string         `json:"engine_key,omitempty" db:"engine_key"` // ai(llm) service api key

	InitPrompt string `json:"init_prompt,omitempty" db:"init_prompt"`

	TTSType    TTSType `json:"tts_type,omitempty" db:"tts_type"`
	TTSVoiceID string  `json:"tts_voice_id,omitempty" db:"tts_voice_id"`

	STTType STTType `json:"stt_type,omitempty" db:"stt_type"`

	// ToolNames defines which tools are enabled for this AI
	// ["all"] = all tools, ["connect_call", "send_email"] = specific tools, [] or nil = no tools
	ToolNames []tool.ToolName `json:"tool_names,omitempty" db:"tool_names,json"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

type EngineModelTarget string

const (
	EngineModelTargetNone       EngineModelTarget = ""
	EngineModelTargetAnthropic  EngineModelTarget = "anthropic"  // Anthropic Claude
	EngineModelTargetAWS        EngineModelTarget = "aws"        // AWS Bedrock
	EngineModelTargetAzure      EngineModelTarget = "azure"      // Azure OpenAI
	EngineModelTargetCerebras   EngineModelTarget = "cerebras"   // Cerebras
	EngineModelTargetDeepSeek   EngineModelTarget = "deepseek"   // DeepSeek
	EngineModelTargetFireworks  EngineModelTarget = "fireworks"  // Fireworks AI
	EngineModelTargetGemini     EngineModelTarget = "gemini"     // Google Gemini
	EngineModelTargetGrok       EngineModelTarget = "grok"       // Grok (xAI)
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
	EngineModelGeminiGemini2Dot5Flash EngineModel = "gemini.gemini-2.5-flash"
	EngineModelGeminiGemini2Dot5Pro   EngineModel = "gemini.gemini-2.5-pro"
	EngineModelGeminiGemini2Dot0Flash EngineModel = "gemini.gemini-2.0-flash"
	EngineModelGeminiGeminiProLatest  EngineModel = "gemini.gemini-pro-latest"

	EngineModelOpenaiGPT5Dot2 EngineModel = "openai.gpt-5.2"
	EngineModelOpenaiGPT5Dot1 EngineModel = "openai.gpt-5.1"
	EngineModelOpenaiGPT5     EngineModel = "openai.gpt-5"
	EngineModelOpenaiGPT5Mini EngineModel = "openai.gpt-5-mini"
	EngineModelOpenaiGPT5Nano EngineModel = "openai.gpt-5-nano"

	EngineModelGrok3     EngineModel = "grok.grok-3"
	EngineModelGrok3Mini EngineModel = "grok.grok-3-mini"
)

func GetEngineModelTarget(engineModel EngineModel) EngineModelTarget {
	mapModelTarget := map[EngineModel]EngineModelTarget{
		EngineModelGeminiGemini2Dot5Flash: EngineModelTargetGemini,
		EngineModelGeminiGemini2Dot5Pro:   EngineModelTargetGemini,
		EngineModelGeminiGemini2Dot0Flash: EngineModelTargetGemini,
		EngineModelGeminiGeminiProLatest:  EngineModelTargetGemini,

		EngineModelOpenaiGPT5Dot2: EngineModelTargetOpenAI,
		EngineModelOpenaiGPT5Dot1: EngineModelTargetOpenAI,
		EngineModelOpenaiGPT5:     EngineModelTargetOpenAI,
		EngineModelOpenaiGPT5Mini: EngineModelTargetOpenAI,
		EngineModelOpenaiGPT5Nano: EngineModelTargetOpenAI,

		EngineModelGrok3:     EngineModelTargetGrok,
		EngineModelGrok3Mini: EngineModelTargetGrok,
	}

	res, ok := mapModelTarget[engineModel]
	if !ok {
		return EngineModelTargetNone
	}

	return res
}

func GetEngineModelName(engineModel EngineModel) string {
	s := string(engineModel)
	idx := strings.Index(s, ".")
	if idx < 0 || idx == len(s)-1 {
		return ""
	}
	return s[idx+1:]
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

var validTTSTypes = map[TTSType]bool{
	TTSTypeNone: true, TTSTypeAsync: true, TTSTypeAWS: true,
	TTSTypeAzure: true, TTSTypeCartesia: true, TTSTypeDeepgram: true,
	TTSTypeElevenLabs: true, TTSTypeFish: true, TTSTypeGoogle: true,
	TTSTypeGroq: true, TTSTypeHume: true, TTSTypeInworld: true,
	TTSTypeLMNT: true, TTSTypeMiniMax: true, TTSTypeNeuphonic: true,
	TTSTypeNvidiaRiva: true, TTSTypeOpenAI: true, TTSTypePiper: true,
	TTSTypePlayHT: true, TTSTypeRime: true, TTSTypeSarvam: true,
	TTSTypeXTTS: true,
}

// IsValid returns true if the TTSType is a known valid value.
func (t TTSType) IsValid() bool {
	return validTTSTypes[t]
}

// ValidValues returns a sorted list of valid TTSType values (excluding empty string).
func (t TTSType) ValidValues() []string {
	res := make([]string, 0, len(validTTSTypes))
	for k := range validTTSTypes {
		if k != TTSTypeNone {
			res = append(res, string(k))
		}
	}
	sort.Strings(res)
	return res
}

// STTType define
type STTType string

const (
	STTTypeNone       STTType = ""
	STTTypeCartesia   STTType = "cartesia"
	STTTypeDeepgram   STTType = "deepgram"
	STTTypeElevenLabs STTType = "elevenlabs"
	STTTypeGoogle     STTType = "google"
)

var validSTTTypes = map[STTType]bool{
	STTTypeNone: true, STTTypeCartesia: true,
	STTTypeDeepgram: true, STTTypeElevenLabs: true,
	STTTypeGoogle: true,
}

// IsValid returns true if the STTType is a known valid value.
func (s STTType) IsValid() bool {
	return validSTTTypes[s]
}

// ValidValues returns a sorted list of valid STTType values (excluding empty string).
func (s STTType) ValidValues() []string {
	res := make([]string, 0, len(validSTTTypes))
	for k := range validSTTTypes {
		if k != STTTypeNone {
			res = append(res, string(k))
		}
	}
	sort.Strings(res)
	return res
}
