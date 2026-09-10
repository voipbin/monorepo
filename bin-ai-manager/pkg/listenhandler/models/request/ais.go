package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
)

// V1DataAIsPost is
// v1 data type request struct for
// /v1/ais POST
type V1DataAIsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Detail     string    `json:"detail,omitempty"`
	Type       ai.Type   `json:"type,omitempty"`

	EngineModel ai.EngineModel `json:"engine_model,omitempty"`
	Parameter   map[string]any `json:"parameter,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`
	RagID       uuid.UUID      `json:"rag_id,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	TTSType    ai.TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string     `json:"tts_voice_id,omitempty"`

	STTType     ai.STTType `json:"stt_type,omitempty"`
	STTLanguage string     `json:"stt_language,omitempty"`

	ToolNames []tool.ToolName `json:"tool_names,omitempty"`

	// McpServerIDs is a pointer to a slice (not a plain slice) so that
	// mcp_server_ids:[] (explicit clear) is distinguishable from the field
	// being absent (leave untouched) on decode -- json.Unmarshal leaves a
	// nil *[]uuid.UUID nil when the key is missing, but allocates a
	// pointer to an empty slice when the key is present as "[]". A plain
	// []uuid.UUID cannot make this distinction on the ENCODE side (both
	// nil and empty marshal identically under omitempty), which is the
	// bug this type fixes end-to-end from bin-api-manager down to here.
	McpServerIDs *[]uuid.UUID `json:"mcp_server_ids,omitempty"`

	VADConfig        *ai.VADConfig `json:"vad_config,omitempty"`
	SmartTurnEnabled bool          `json:"smart_turn_enabled,omitempty"`

	AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty"`
}

// V1DataAIsIDPut is
// v1 data type request struct for
// /v1/ais/<ai-id> PUT
type V1DataAIsIDPut struct {
	Name   string  `json:"name,omitempty"`
	Detail string  `json:"detail,omitempty"`
	Type   ai.Type `json:"type,omitempty"`

	EngineModel ai.EngineModel `json:"engine_model,omitempty"`
	Parameter   map[string]any `json:"parameter,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`
	RagID       uuid.UUID      `json:"rag_id,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	TTSType    ai.TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string     `json:"tts_voice_id,omitempty"`

	STTType     ai.STTType `json:"stt_type,omitempty"`
	STTLanguage string     `json:"stt_language,omitempty"`

	ToolNames []tool.ToolName `json:"tool_names,omitempty"`

	// McpServerIDs is a pointer to a slice (not a plain slice) so that
	// mcp_server_ids:[] (explicit clear) is distinguishable from the field
	// being absent (leave untouched) on decode -- json.Unmarshal leaves a
	// nil *[]uuid.UUID nil when the key is missing, but allocates a
	// pointer to an empty slice when the key is present as "[]". A plain
	// []uuid.UUID cannot make this distinction on the ENCODE side (both
	// nil and empty marshal identically under omitempty), which is the
	// bug this type fixes end-to-end from bin-api-manager down to here.
	McpServerIDs *[]uuid.UUID `json:"mcp_server_ids,omitempty"`

	VADConfig        *ai.VADConfig `json:"vad_config,omitempty"`
	SmartTurnEnabled bool          `json:"smart_turn_enabled,omitempty"`

	AutoAICallAuditEnabled bool `json:"auto_aicall_audit_enabled,omitempty"`
}
