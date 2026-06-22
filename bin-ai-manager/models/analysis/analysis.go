package analysis

import (
	"encoding/json"
)

// Request is the generic LLM gateway input. Internal callers only.
//
// The gateway is stateless: it takes a prompt + arbitrary data payload + a JSON
// schema, runs a single structured-output (json_schema, strict) chat completion,
// and returns the schema-conformant JSON. It persists nothing; the caller owns
// any async lifecycle and storage.
type Request struct {
	Prompt     string          `json:"prompt"`          // system/instruction text
	Data       json.RawMessage `json:"data"`            // arbitrary caller-supplied payload, rendered into the user message
	Schema     json.RawMessage `json:"schema"`          // JSON Schema for response_format=json_schema (required)
	SchemaName string          `json:"schema_name"`     // required by OpenAI json_schema (response_format.json_schema.name)
	Model      string          `json:"model,omitempty"` // optional; must be in the allowed model set, else default
}

// Response carries the structured LLM output and accounting.
type Response struct {
	Result       json.RawMessage `json:"result"`        // the schema-conformant JSON object
	Model        string          `json:"model"`         // model actually used
	FinishReason string          `json:"finish_reason"` // "stop" / "length" / ... so the caller can detect truncation BEFORE its own Validate()
	Truncated    bool            `json:"truncated"`     // true when FinishReason=="length" (output cut, JSON likely invalid)
	PromptTokens int             `json:"prompt_tokens"`
	OutputTokens int             `json:"output_tokens"`
}
