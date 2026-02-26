package team

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/models/identity"
)

// ResolvedAI is a stripped-down version of ai.AI for passing to pipecat-manager.
// It omits EngineKey (sensitive) but includes Identity for logging/tracing.
type ResolvedAI struct {
	identity.Identity

	EngineModel ai.EngineModel  `json:"engine_model,omitempty"`
	EngineData  map[string]any  `json:"engine_data,omitempty"`
	InitPrompt  string          `json:"init_prompt,omitempty"`
	TTSType     ai.TTSType      `json:"tts_type,omitempty"`
	TTSVoiceID  string          `json:"tts_voice_id,omitempty"`
	STTType     ai.STTType      `json:"stt_type,omitempty"`
	ToolNames   []tool.ToolName `json:"tool_names,omitempty"`
}

// ResolvedMember is a member with its AI config and tools fully resolved.
type ResolvedMember struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	AI          ResolvedAI   `json:"ai"`
	Tools       []tool.Tool  `json:"tools"`
	Transitions []Transition `json:"transitions"`
}

// ResolvedTeam carries the fully resolved team config across the RPC boundary.
type ResolvedTeam struct {
	ID            uuid.UUID        `json:"id"`
	StartMemberID uuid.UUID        `json:"start_member_id"`
	Members       []ResolvedMember `json:"members"`
}

// ConvertResolvedAI builds a ResolvedAI from an ai.AI, stripping EngineKey.
func ConvertResolvedAI(a *ai.AI) ResolvedAI {
	return ResolvedAI{
		Identity:    a.Identity,
		EngineModel: a.EngineModel,
		EngineData:  a.EngineData,
		InitPrompt:  a.InitPrompt,
		TTSType:     a.TTSType,
		TTSVoiceID:  a.TTSVoiceID,
		STTType:     a.STTType,
		ToolNames:   a.ToolNames,
	}
}
