package aicall

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name      string
		aicall    *AIcall
		checkFunc func(t *testing.T, wh *WebhookMessage, ac *AIcall)
	}{
		{
			name: "converts_aicall_with_all_fields",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				Parameter:      map[string]any{"key": "value"},
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "voice-123",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.Must(uuid.NewV4()),
				ReferenceType:  ReferenceTypeCall,
				ReferenceID:    uuid.Must(uuid.NewV4()),
				ConfbridgeID:   uuid.Must(uuid.NewV4()),
				PipecatcallID:  uuid.Must(uuid.NewV4()),
				Status:         StatusProgressing,
				STTLanguage:    "en-US",
				TMEnd:          ptrTime(time.Now()),
				TMCreate:       ptrTime(time.Now()),
				TMUpdate:       ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.AssistanceType != ac.AssistanceType {
					t.Errorf("Wrong AssistanceType. expect: %s, got: %s", ac.AssistanceType, wh.AssistanceType)
				}
				if wh.AssistanceID != ac.AssistanceID {
					t.Errorf("Wrong AssistanceID. expect: %s, got: %s", ac.AssistanceID, wh.AssistanceID)
				}
				if wh.AIEngineModel != ac.AIEngineModel {
					t.Errorf("Wrong AIEngineModel. expect: %s, got: %s", ac.AIEngineModel, wh.AIEngineModel)
				}
				if wh.Parameter["key"] != ac.Parameter["key"] {
					t.Errorf("Wrong Parameter. expect: %v, got: %v", ac.Parameter, wh.Parameter)
				}
				if wh.AITTSType != ac.AITTSType {
					t.Errorf("Wrong AITTSType. expect: %s, got: %s", ac.AITTSType, wh.AITTSType)
				}
				if wh.AITTSVoiceID != ac.AITTSVoiceID {
					t.Errorf("Wrong AITTSVoiceID. expect: %s, got: %s", ac.AITTSVoiceID, wh.AITTSVoiceID)
				}
				if wh.AISTTType != ac.AISTTType {
					t.Errorf("Wrong AISTTType. expect: %s, got: %s", ac.AISTTType, wh.AISTTType)
				}
				if wh.Status != ac.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", ac.Status, wh.Status)
				}
				if wh.STTLanguage != ac.STTLanguage {
					t.Errorf("Wrong STTLanguage. expect: %s, got: %s", ac.STTLanguage, wh.STTLanguage)
				}
			},
		},
		{
			name: "converts_aicall_with_current_member_id",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				CurrentMemberID: uuid.Must(uuid.NewV4()),
				Status:          StatusProgressing,
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.CurrentMemberID != ac.CurrentMemberID {
					t.Errorf("Wrong CurrentMemberID. expect: %s, got: %s", ac.CurrentMemberID, wh.CurrentMemberID)
				}
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.Status != ac.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", ac.Status, wh.Status)
				}
			},
		},
		{
			name: "converts_aicall_with_empty_fields",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, ac *AIcall) {
				if wh.ID != ac.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", ac.ID, wh.ID)
				}
				if wh.CurrentMemberID != uuid.Nil {
					t.Errorf("Wrong CurrentMemberID. expect: %s, got: %s", uuid.Nil, wh.CurrentMemberID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := tt.aicall.ConvertWebhookMessage()
			if wh == nil {
				t.Error("Expected non-nil webhook, got nil")
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, wh, tt.aicall)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		aicall    *AIcall
		wantError bool
	}{
		{
			name: "creates_webhook_event_successfully",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
				Status:         StatusProgressing,
				STTLanguage:    "ko-KR",
				ReferenceType:  ReferenceTypeConversation,
				ReferenceID:    uuid.Must(uuid.NewV4()),
				TMCreate:       ptrTime(time.Now()),
			},
			wantError: false,
		},
		{
			name: "creates_webhook_event_with_empty_aicall",
			aicall: &AIcall{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.aicall.CreateWebhookEvent()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateWebhookEvent() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify it's valid JSON
				var wh WebhookMessage
				if errUnmarshal := json.Unmarshal(data, &wh); errUnmarshal != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", errUnmarshal)
				}
			}
		})
	}
}

func TestConvertWebhookMessage_includesMetadata(t *testing.T) {
	h := &AIcall{
		Metadata: map[string]any{
			MetaKeyPromptSnapshots: []PromptSnapshot{
				{Prompt: "hello world"},
			},
		},
	}
	msg := h.ConvertWebhookMessage()
	if msg.Metadata == nil {
		t.Fatal("expected Metadata to be non-nil in WebhookMessage")
	}
	if _, ok := msg.Metadata[MetaKeyPromptSnapshots]; !ok {
		t.Errorf("expected %q key in Metadata", MetaKeyPromptSnapshots)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// TestConvertWebhookMessage_McpToolMapNeverLeaks pins B22: a non-empty
// mcp_tool_map metadata entry must never appear in ConvertWebhookMessage's
// output -- it carries the customer's own MCP server UUIDs and remote tool
// names, which must not leak onto the messaging webhook payload.
func TestConvertWebhookMessage_McpToolMapNeverLeaks(t *testing.T) {
	serverID := uuid.Must(uuid.NewV4())
	h := &AIcall{
		Metadata: map[string]any{
			MetaKeyMcpToolMap: map[string]McpToolRef{
				"mcp_aaaaaaaa_search_tickets": {ServerID: serverID, ToolName: "search_tickets"},
			},
		},
	}

	msg := h.ConvertWebhookMessage()

	if _, ok := msg.Metadata[MetaKeyMcpToolMap]; ok {
		t.Errorf("expected %q to be removed from the webhook projection, got: %v", MetaKeyMcpToolMap, msg.Metadata)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("unexpected error marshaling: %v", err)
	}
	if strings.Contains(string(data), serverID.String()) {
		t.Errorf("the mcp server UUID leaked into the webhook payload: %s", data)
	}
	if strings.Contains(string(data), "search_tickets") {
		t.Errorf("the mcp remote tool name leaked into the webhook payload: %s", data)
	}
}

// TestConvertWebhookMessage_McpToolStatusSummary pins B22's replacement: a
// non-empty mcp_tool_map projects to mcp_tool_status={servers, tools} --
// counts only, no server UUIDs, no tool names, no schemas.
func TestConvertWebhookMessage_McpToolStatusSummary(t *testing.T) {
	serverA := uuid.Must(uuid.NewV4())
	serverB := uuid.Must(uuid.NewV4())
	h := &AIcall{
		Metadata: map[string]any{
			MetaKeyMcpToolMap: map[string]McpToolRef{
				"mcp_aaaaaaaa_search_tickets": {ServerID: serverA, ToolName: "search_tickets"},
				"mcp_aaaaaaaa_create_ticket":  {ServerID: serverA, ToolName: "create_ticket"},
				"mcp_bbbbbbbb_lookup_order":   {ServerID: serverB, ToolName: "lookup_order"},
			},
		},
	}

	msg := h.ConvertWebhookMessage()

	raw, ok := msg.Metadata[MetaKeyMcpToolStatus]
	if !ok {
		t.Fatalf("expected %q key in Metadata, got: %v", MetaKeyMcpToolStatus, msg.Metadata)
	}
	status, ok := raw.(McpToolStatus)
	if !ok {
		t.Fatalf("expected %q to be an McpToolStatus, got: %T", MetaKeyMcpToolStatus, raw)
	}
	if status.Servers != 2 {
		t.Errorf("expected Servers=2 (deduped by server id), got: %d", status.Servers)
	}
	if status.Tools != 3 {
		t.Errorf("expected Tools=3, got: %d", status.Tools)
	}
}

// TestConvertWebhookMessage_McpToolStatusAbsentWhenEmpty pins that an empty
// or absent mcp_tool_map produces no mcp_tool_status key at all -- matching
// existing PR B1 framing that the blast radius is nil until a non-empty map
// exists in production traffic.
func TestConvertWebhookMessage_McpToolStatusAbsentWhenEmpty(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
	}{
		{name: "no metadata at all", metadata: nil},
		{name: "mcp_tool_map key absent", metadata: map[string]any{MetaKeyPromptSnapshots: []PromptSnapshot{}}},
		{name: "mcp_tool_map present but empty", metadata: map[string]any{MetaKeyMcpToolMap: map[string]McpToolRef{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &AIcall{Metadata: tt.metadata}
			msg := h.ConvertWebhookMessage()
			if _, ok := msg.Metadata[MetaKeyMcpToolStatus]; ok {
				t.Errorf("expected no %q key, got: %v", MetaKeyMcpToolStatus, msg.Metadata)
			}
		})
	}
}
