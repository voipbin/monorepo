package builderhandler

import "time"

// JSONMode selects the response_format sent to the provider. It exists so the
// evaluation can compare modes by parse-failure rate (design 2.5).
type JSONMode string

const (
	// JSONModeSchema sends json_schema with Strict:false (the default of
	// DefaultConfig, used by the Assistant Builder). The Gemini OpenAI-compatible
	// endpoint does not enforce strict mode, so Parse still validates everything.
	// The Flow Builder does not use it: see FlowConfig.
	JSONModeSchema JSONMode = "json_schema"
	// JSONModeObject sends json_object (valid JSON, no schema).
	JSONModeObject JSONMode = "json_object"
	// JSONModeNone sends no response_format; Parse extracts JSON from prose.
	JSONModeNone JSONMode = "none"
)

// Config is the per-turn behaviour of the Builder. It is a plain value type
// with no pointer or slice fields so the evaluation can copy it and change one
// field to compare variants.
//
// Model, ReasoningEffort, MaxOutputTokens and LLMTimeout are operational
// values. SystemPrompt, JSONMode and DataBlockInSystem are "evaluation
// overrides": the evaluation changes them, the Assistant Builder keeps the
// defaults, and the Flow Builder fixes the JSON mode itself (FlowConfig). They
// are the price of measuring the production code path rather than a copy of it.
//
// All numeric defaults are initial values that have NOT been measured.
type Config struct {
	Model           string
	ReasoningEffort string // "" omits the field
	MaxOutputTokens int
	LLMTimeout      time.Duration

	// SystemPrompt is the system prompt. Evaluation overrides it; production
	// uses the SystemPrompt constant.
	SystemPrompt string
	// JSONMode is the response_format. Evaluation overrides it; the Assistant
	// Builder uses JSONModeSchema and the Flow Builder JSONModeObject
	// (FlowConfig).
	JSONMode JSONMode
	// DataBlockInSystem merges the session-facts block into the system prompt
	// instead of prefixing the last user message. Evaluation only; production
	// is false.
	DataBlockInSystem bool
}

// Initial, unmeasured defaults (design 4.3, 4.6).
const (
	defaultModel           = "gemini-3.8-flash"
	defaultReasoningEffort = "none"
	defaultMaxOutputTokens = 4096
	defaultLLMTimeout      = 40 * time.Second
)

// DefaultConfig returns the production behaviour.
func DefaultConfig() Config {
	return Config{
		Model:           defaultModel,
		ReasoningEffort: defaultReasoningEffort,
		MaxOutputTokens: defaultMaxOutputTokens,
		LLMTimeout:      defaultLLMTimeout,
		SystemPrompt:    SystemPrompt,
		JSONMode:        JSONModeSchema,
	}
}
