package aicall

import (
	"encoding/json"
	"time"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
	identity.Identity

	AssistanceType AssistanceType `json:"assistance_type,omitempty"`
	AssistanceID   uuid.UUID      `json:"assistance_id,omitempty"`

	AIEngineModel      ai.EngineModel `json:"ai_engine_model,omitempty"`
	AITTSType          ai.TTSType     `json:"ai_tts_type,omitempty"`
	AITTSVoiceID       string         `json:"ai_tts_voice_id,omitempty"`
	AISTTType          ai.STTType     `json:"ai_stt_type,omitempty"`
	AIVADConfig        *ai.VADConfig  `json:"ai_vad_config,omitempty"`
	AISmartTurnEnabled bool           `json:"ai_smart_turn_enabled,omitempty"`

	Parameter map[string]any `json:"parameter,omitempty"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`

	CurrentMemberID uuid.UUID `json:"current_member_id,omitempty"`

	Status Status `json:"status,omitempty"`

	STTLanguage string `json:"stt_language,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`

	TMEnd    *time.Time `json:"tm_end"`
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *AIcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AssistanceType: h.AssistanceType,
		AssistanceID:   h.AssistanceID,

		AIEngineModel:      h.AIEngineModel,
		AITTSType:          h.AITTSType,
		AITTSVoiceID:       h.AITTSVoiceID,
		AISTTType:          h.AISTTType,
		AIVADConfig:        h.AIVADConfig,
		AISmartTurnEnabled: h.AISmartTurnEnabled,

		Parameter: h.Parameter,

		ActiveflowID:  h.ActiveflowID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		ConfbridgeID: h.ConfbridgeID,

		CurrentMemberID: h.CurrentMemberID,

		Status: h.Status,

		STTLanguage: h.STTLanguage,

		Metadata: mcpProjectedMetadata(h.Metadata),

		TMEnd:    h.TMEnd,
		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// McpToolStatus is the documented, non-identifying summary
// ConvertWebhookMessage substitutes for a non-empty MetaKeyMcpToolMap (B22,
// design docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md §7). No
// server UUIDs, no tool names, no schemas -- counts only.
type McpToolStatus struct {
	Servers int `json:"servers"`
	Tools   int `json:"tools"`
}

// MetaKeyMcpToolStatus is the webhook-projection-only Metadata key
// ConvertWebhookMessage writes in place of MetaKeyMcpToolMap (B22). It is
// never written to the stored AIcall row itself -- only to the WebhookMessage
// this function returns.
const MetaKeyMcpToolStatus = "mcp_tool_status"

// mcpProjectedMetadata copies raw (the stored AIcall's Metadata) with
// MetaKeyMcpToolMap removed and, when it was non-empty, replaced with a
// derived MetaKeyMcpToolStatus summary (B22). This is a breaking payload
// change relative to the field the pre-B12 caveat commit
// (NOJIRA-Caveat-mcp-tool-use-not-yet-available) already stated as
// present-but-always-empty: PR B2 is what first makes a non-empty map
// possible in production traffic, so this ships in the same PR that turns
// advertisement on -- no window opens where the raw map could leak.
func mcpProjectedMetadata(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}

	out := make(map[string]any, len(raw))
	for k, v := range raw {
		if k == MetaKeyMcpToolMap {
			continue
		}
		out[k] = v
	}

	if status, ok := mcpToolStatusFromRaw(raw[MetaKeyMcpToolMap]); ok {
		out[MetaKeyMcpToolStatus] = status
	}

	return out
}

// mcpToolStatusFromRaw summarizes v (the raw MetaKeyMcpToolMap value, in
// either the in-process map[string]McpToolRef shape or the map[string]any
// shape Metadata carries after a JSON round trip) into an McpToolStatus.
// ok is false, and no summary is produced, for an absent, malformed, or
// empty map -- matching the "blast radius nil" framing: no entry means no
// mcp_tool_status key at all, not an empty one.
func mcpToolStatusFromRaw(v any) (McpToolStatus, bool) {
	servers := map[uuid.UUID]struct{}{}
	tools := 0

	switch tm := v.(type) {
	case map[string]McpToolRef:
		for _, ref := range tm {
			servers[ref.ServerID] = struct{}{}
			tools++
		}

	case map[string]any:
		for _, entry := range tm {
			ref, ok := decodeMcpToolRefForStatus(entry)
			if !ok {
				continue
			}
			servers[ref.ServerID] = struct{}{}
			tools++
		}

	default:
		return McpToolStatus{}, false
	}

	if tools == 0 {
		return McpToolStatus{}, false
	}

	return McpToolStatus{Servers: len(servers), Tools: tools}, true
}

// decodeMcpToolRefForStatus decodes one MetaKeyMcpToolMap entry after a JSON
// round trip (server_id/tool_name keys, McpToolRef's json tags), for
// counting purposes only. Malformed entries are skipped, not counted --
// matching this whole path's fail-closed posture elsewhere.
func decodeMcpToolRefForStatus(v any) (McpToolRef, bool) {
	if ref, ok := v.(McpToolRef); ok {
		return ref, true
	}

	m, ok := v.(map[string]any)
	if !ok {
		return McpToolRef{}, false
	}

	serverIDRaw, ok := m["server_id"].(string)
	if !ok {
		return McpToolRef{}, false
	}
	serverID, err := uuid.FromString(serverIDRaw)
	if err != nil {
		return McpToolRef{}, false
	}

	toolName, ok := m["tool_name"].(string)
	if !ok {
		return McpToolRef{}, false
	}

	return McpToolRef{ServerID: serverID, ToolName: toolName}, true
}

// CreateWebhookEvent generate WebhookEvent
func (h *AIcall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
