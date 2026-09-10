package aicallhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/tool"
)

// mcpToolNamePrefix is the reserved namespace prefix marking a tool name as
// resolved from a customer-registered McpServer rather than a VoIPBin
// built-in tool. See docs/plans/2026-09-11-mcp-tool-integration-design.md
// §9.1/§9.2.
const mcpToolNamePrefix = "mcp_"

// toolNameResolver is the exact shape of pkg/toolhandler.ToolHandler's
// GetByNames method, declared locally rather than imported. pkg/toolhandler
// has a test file that imports pkg/aicallhandler (definitions_resource_test.go),
// so aicallhandler importing toolhandler back would be an import cycle;
// Go's structural typing lets toolhandler.ToolHandler (constructed once in
// cmd/ai-manager) satisfy this narrower interface with no import needed here.
type toolNameResolver interface {
	GetByNames(names []tool.ToolName) []tool.Tool
}

// resolveTools builds the merged LLM tool list (VoIPBin built-ins + the
// customer's whitelisted McpServer tools) for the AI a, and the mapping from
// each namespaced MCP tool name back to the (server_id, original_tool_name)
// pair it was resolved from, for storage under aicall.MetaKeyMcpToolMap.
//
// Built-ins are resolved via the existing toolhandler.ToolHandler.GetByNames
// (unchanged, not duplicated here). toolhandler.toolHandler carries no
// state, so a fresh instance is equivalent to any other; this avoids adding
// a third constructor dependency to aicallHandler for a call that is
// stateless.
//
// Each McpServer in a.McpServerIDs is best-effort: a non-active server is
// silently skipped (no error, no tools), and a ListTools failure for one
// server is logged and skipped rather than failing the whole resolution --
// a single misbehaving customer MCP server must never break the built-in
// tool list or any other server's tools. The only errors this can return
// come from the caller having already resolved a in hand; today there is no
// such failure mode inside this function, but the error return is kept so
// resolveTools composes cleanly with callers that may gain one.
func (h *aicallHandler) resolveTools(ctx context.Context, a *ai.AI) ([]tool.Tool, map[string]aicall.McpToolRef, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "resolveTools",
		"ai_id": a.ID,
	})

	builtins := []tool.Tool{}
	if h.toolNameResolver != nil {
		builtins = h.toolNameResolver.GetByNames(a.ToolNames)
	}

	merged := make([]tool.Tool, 0, len(builtins))
	merged = append(merged, builtins...)

	toolMap := map[string]aicall.McpToolRef{}

	if h.mcpServerHandler == nil || h.mcptoolHandler == nil {
		return merged, toolMap, nil
	}

	for _, serverID := range a.McpServerIDs {
		server, err := h.mcpServerHandler.Get(ctx, serverID)
		if err != nil {
			log.Warnf("Could not get mcp server, skipping its tools. mcp_server_id: %s, err: %v", serverID, err)
			continue
		}

		if server.Status != mcpserver.StatusActive {
			log.Debugf("Mcp server is not active, skipping its tools. mcp_server_id: %s, status: %s", serverID, server.Status)
			continue
		}

		mcpTools, err := h.mcptoolHandler.ListTools(ctx, serverID)
		if err != nil {
			log.Warnf("Could not list tools from mcp server, skipping. mcp_server_id: %s, err: %v", serverID, err)
			continue
		}

		prefix := mcpToolNamePrefix + mcpServerIDShort(serverID) + "_"
		for _, mt := range mcpTools {
			namespacedName := prefix + mt.Name

			merged = append(merged, tool.Tool{
				Name:        tool.ToolName(namespacedName),
				Description: mt.Description,
				Parameters:  mt.InputSchema,
				RunLLM:      true,
			})

			toolMap[namespacedName] = aicall.McpToolRef{
				ServerID: serverID,
				ToolName: mt.Name,
			}
		}
	}

	return merged, toolMap, nil
}

// mcpServerIDShort returns the first 8 lowercase-hex characters of id, with
// dashes stripped, used as the collision-resistant-enough namespace segment
// in a remote MCP tool's LLM-facing name (design §9.1 item 3).
func mcpServerIDShort(id uuid.UUID) string {
	stripped := strings.ReplaceAll(id.String(), "-", "")
	if len(stripped) < 8 {
		return stripped
	}
	return stripped[:8]
}

// toolHandleMcpCall dispatches an LLM tool_call whose function name carries
// the mcp_ namespace prefix (design §9.2). It fails closed at every step: an
// unresolvable name, a server no longer in the AI's whitelist, or a server
// that is no longer active all produce a generic failure rather than routing
// the call through.
func (h *aicallHandler) toolHandleMcpCall(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":       "toolHandleMcpCall",
		"aicall_id":  c.ID,
		"tool_name":  tc.Function.Name,
	})
	log.Debugf("handling mcp tool call.")

	res := newToolResult(tc.ID)

	ref, ok := lookupMcpToolRef(c, string(tc.Function.Name))
	if !ok {
		log.Debugf("could not resolve mcp tool name from aicall metadata.")
		fillFailed(res, errMcpToolCallFailed("unknown mcp tool call"))
		return res
	}

	tmpAI, _, _, err := h.resolveAI(ctx, c.AssistanceType, c.AssistanceID)
	if err != nil {
		log.Errorf("Could not resolve AI. err: %v", err)
		fillFailed(res, errMcpToolCallFailed("could not retrieve AI configuration"))
		return res
	}

	if !mcpServerIDIsWhitelisted(tmpAI.McpServerIDs, ref.ServerID) {
		log.Warnf("Mcp server is no longer whitelisted for this ai, refusing the call. mcp_server_id: %s", ref.ServerID)
		fillFailed(res, errMcpToolCallFailed("mcp tool is no longer available"))
		return res
	}

	server, err := h.mcpServerHandler.Get(ctx, ref.ServerID)
	if err != nil {
		log.Errorf("Could not get mcp server. mcp_server_id: %s, err: %v", ref.ServerID, err)
		fillFailed(res, errMcpToolCallFailed("mcp tool is no longer available"))
		return res
	}
	if server.Status != mcpserver.StatusActive {
		log.Warnf("Mcp server is not active, refusing the call. mcp_server_id: %s, status: %s", ref.ServerID, server.Status)
		fillFailed(res, errMcpToolCallFailed("mcp tool is no longer available"))
		return res
	}

	msg, err := h.mcptoolHandler.CallTool(ctx, ref.ServerID, ref.ToolName, tc.Function.Arguments)
	if err != nil {
		log.Errorf("Mcp tool call failed. mcp_server_id: %s, tool_name: %s, err: %v", ref.ServerID, ref.ToolName, capErrText(err.Error(), 200))
		fillFailed(res, errMcpToolCallFailed("MCP tool call failed"))
		return res
	}

	fillSuccess(res, "mcp_tool", ref.ServerID.String(), msg)
	return res
}

// lookupMcpToolRef resolves name against c.Metadata[aicall.MetaKeyMcpToolMap].
// The map is stored as a Go value (map[string]aicall.McpToolRef) by
// resolveTools' callers at write time, but AIcall.Metadata may also come
// back from a JSON round trip (map[string]any with a nested
// map[string]any), so both shapes are handled.
func lookupMcpToolRef(c *aicall.AIcall, name string) (aicall.McpToolRef, bool) {
	raw, ok := c.Metadata[aicall.MetaKeyMcpToolMap]
	if !ok {
		return aicall.McpToolRef{}, false
	}

	switch tm := raw.(type) {
	case map[string]aicall.McpToolRef:
		ref, found := tm[name]
		return ref, found

	case map[string]any:
		entry, found := tm[name]
		if !found {
			return aicall.McpToolRef{}, false
		}
		return decodeMcpToolRef(entry)

	default:
		return aicall.McpToolRef{}, false
	}
}

// decodeMcpToolRef decodes one map entry of MetaKeyMcpToolMap after a JSON
// round trip, where it arrives as map[string]any with server_id/tool_name
// keys (McpToolRef's json tags).
func decodeMcpToolRef(v any) (aicall.McpToolRef, bool) {
	if ref, ok := v.(aicall.McpToolRef); ok {
		return ref, true
	}

	m, ok := v.(map[string]any)
	if !ok {
		return aicall.McpToolRef{}, false
	}

	serverIDRaw, ok := m["server_id"].(string)
	if !ok {
		return aicall.McpToolRef{}, false
	}
	serverID, err := uuid.FromString(serverIDRaw)
	if err != nil {
		return aicall.McpToolRef{}, false
	}

	toolName, ok := m["tool_name"].(string)
	if !ok {
		return aicall.McpToolRef{}, false
	}

	return aicall.McpToolRef{ServerID: serverID, ToolName: toolName}, true
}

// mcpServerIDIsWhitelisted reports whether serverID is present in ids.
func mcpServerIDIsWhitelisted(ids []uuid.UUID, serverID uuid.UUID) bool {
	for _, id := range ids {
		if id == serverID {
			return true
		}
	}
	return false
}

// capErrText bounds a remote MCP server's error text before it appears in a
// log line -- a misbehaving customer server should not be able to inject
// unbounded content there either.
func capErrText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// errMcpToolCallFailed builds the generic, size-bounded failure message
// surfaced to the LLM. The customer's remote MCP server's raw error text is
// deliberately never forwarded verbatim (design §9.2 item 6).
func errMcpToolCallFailed(msg string) error {
	return mcpToolCallError(msg)
}

// mcpToolCallError is a plain string error type so errMcpToolCallFailed
// avoids importing "errors" solely for New.
type mcpToolCallError string

func (e mcpToolCallError) Error() string { return string(e) }

// refreshMcpToolMap performs a read-modify-write of existing's Metadata,
// merging a freshly resolved MetaKeyMcpToolMap into it without disturbing any
// other key another writer has already set (design §9.1's "AIcall reuse and
// Metadata staleness" resolution). It re-reads the row first so a concurrent
// writer (e.g. a listen-start pointer) is merged rather than clobbered, then
// writes back via the same NoTouchTMUpdate DB method
// writeInsightSessionMetadata uses, so this bookkeeping write is not
// misread as agent/session activity by any TMUpdate-based idle rule.
func (h *aicallHandler) refreshMcpToolMap(ctx context.Context, existing *aicall.AIcall, a *ai.AI) error {
	if h.mcpServerHandler == nil || h.mcptoolHandler == nil {
		return nil
	}

	_, mcpToolMap, err := h.resolveTools(ctx, a)
	if err != nil {
		return err
	}

	cur, err := h.db.AIcallGet(ctx, existing.ID)
	if err != nil {
		return err
	}

	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyMcpToolMap] = mcpToolMap

	return h.db.AIcallUpdateNoTouchTMUpdate(ctx, existing.ID, map[aicall.Field]any{
		aicall.FieldMetadata: metadata,
	})
}
