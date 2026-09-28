package aicallhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
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

// resolveMcpToolMap discovers the tools of the AI a's whitelisted MCP servers
// and returns the mapping from each namespaced tool name back to the
// (server_id, original_tool_name) pair it was resolved from, for storage
// under aicall.MetaKeyMcpToolMap. It is what the session-start paths need:
// they store the map and advertise nothing, so no input schema is decoded.
func (h *aicallHandler) resolveMcpToolMap(ctx context.Context, a *ai.AI) map[string]aicall.McpToolRef {
	toolMap := map[string]aicall.McpToolRef{}
	for _, d := range h.discoverMcpTools(ctx, a, false) {
		toolMap[d.name] = d.ref
	}
	return toolMap
}

// resolveTools builds the merged LLM tool list (VoIPBin built-ins + the
// customer's whitelisted McpServer tools) for the AI a, and the same tool map
// resolveMcpToolMap returns. Unlike resolveMcpToolMap it decodes each input
// schema, within the limits in decodeToolSchema.
//
// No session-start path calls this yet: they need only the map, and decoding
// schemas nobody reads would cost memory on every session start. It is the
// entry point for advertising MCP tools to a model (PR B2), which must also
// enforce description limits before it does.
//
// Built-ins are resolved via the existing toolhandler.ToolHandler.GetByNames
// (unchanged, not duplicated here).
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
	schemaBudget := mcpToolSchemaBudgetBytes
	skipped := map[uuid.UUID]*schemaSkips{}

	for _, d := range h.discoverMcpTools(ctx, a, true) {
		params, why := decodeToolSchema(d.inputSchema, &schemaBudget)
		if why != schemaOK {
			s := skipped[d.ref.ServerID]
			if s == nil {
				s = &schemaSkips{}
				skipped[d.ref.ServerID] = s
			}
			if why == schemaOverBudget {
				s.overBudget++
			} else {
				s.invalid++
			}
			continue
		}

		merged = append(merged, tool.Tool{
			Name:        tool.ToolName(d.name),
			Description: d.description,
			Parameters:  params,
			RunLLM:      true,
		})
		toolMap[d.name] = d.ref
	}

	for serverID, s := range skipped {
		log.Warnf("Skipped mcp tools whose input schema could not be used. mcp_server_id: %s, too_large_or_malformed: %d, over_shared_budget: %d", serverID, s.invalid, s.overBudget)
	}

	return merged, toolMap, nil
}

// discoveredMcpTool is one tool taken from a whitelisted server, already
// namespaced and validated. description and inputSchema are set only when
// the caller asked for content, and the schema is still raw.
type discoveredMcpTool struct {
	name        string
	ref         aicall.McpToolRef
	description string
	inputSchema json.RawMessage
}

const (
	// mcpMaxToolNameLen is the longest remote tool name accepted. With the
	// 13-byte "mcp_<8 hex>_" prefix the namespaced name stays within the
	// 64-character function-name limit LLM providers enforce.
	mcpMaxToolNameLen = 64 - len(mcpToolNamePrefix) - 8 - 1

	// mcpMaxToolsPerResolution caps the tools taken across all of an AI's
	// servers. Together with the name limit it bounds the tool map stored
	// on the aicall and published in its webhooks to a few tens of KiB,
	// whatever the number of whitelisted servers.
	mcpMaxToolsPerResolution = 256
)

// mcpDiscoverySlots bounds how many tools/list requests this process runs at
// once, and mcpDiscoverySlotWait how long one resolution may wait for them in
// total; see discoverMcpTools. Measured with ten concurrent session starts of
// eight servers each returning a worst-case 1 MiB list: about 8 to 20 MiB of
// peak heap above baseline with two slots (JSON and SSE framing), 17 to 75 MiB
// with no bound, against a 40M container limit.
//
// The price is throughput: the process lists at most two servers at a time,
// so a burst of session starts whose servers are slow can exhaust the wait
// and start some sessions without their MCP tools (logged). Raising the slot
// count needs the per-listing memory lowered first.
var mcpDiscoverySlots = make(chan struct{}, 2)

// mcpDiscoverySlotWait is a variable only so tests can shorten it.
var mcpDiscoverySlotWait = 2 * time.Second

// listToolsWithSlot lists serverID's tools while holding one of
// mcpDiscoverySlots, charging only the time spent blocked on a slot to
// *waitLeft. listed is false when no slot came free within what was left;
// the slot is released on every path out of ListTools, a panic included.
func (h *aicallHandler) listToolsWithSlot(ctx context.Context, serverID uuid.UUID, waitLeft *time.Duration) (tools []mcptoolhandler.McpTool, listed bool, err error) {
	select {
	case mcpDiscoverySlots <- struct{}{}:
	default:
		if *waitLeft <= 0 {
			return nil, false, nil
		}
		timer := time.NewTimer(*waitLeft)
		start := time.Now()
		select {
		case mcpDiscoverySlots <- struct{}{}:
			timer.Stop()
			*waitLeft -= time.Since(start)
		case <-timer.C:
			*waitLeft = 0
			return nil, false, nil
		case <-ctx.Done():
			timer.Stop()
			*waitLeft -= time.Since(start)
			return nil, false, nil
		}
	}
	defer func() { <-mcpDiscoverySlots }()

	tools, err = h.mcptoolHandler.ListTools(ctx, serverID)
	return tools, true, err
}

// sampleToolNames renders up to three names for a log line, each cut short,
// so an operator can see which tools were dropped without the server
// controlling how much is logged.
func sampleToolNames(names []string) string {
	const maxNames, maxLen = 3, 80
	out := make([]string, 0, maxNames)
	for i, n := range names {
		if i == maxNames {
			break
		}
		out = append(out, strconv.Quote(capErrText(n, maxLen)))
	}
	return strings.Join(out, ", ")
}

// validMcpToolName reports whether a remote tool name can be namespaced and
// stored as is. Names outside the provider function-name character set or
// over the length limit are dropped, never rewritten (design D5): a
// rewritten name would not be the name the server knows.
func validMcpToolName(name string) bool {
	if name == "" || len(name) > mcpMaxToolNameLen {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !isToolNameByte(name[i]) {
			return false
		}
	}
	return true
}

// isToolNameByte reports whether c is in [A-Za-z0-9_-].
func isToolNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '_', c == '-':
		return true
	}
	return false
}

// discoverMcpTools lists the tools of each of a's usable whitelisted servers,
// in whitelist order. With keepContent false only names are kept, so a
// server's descriptions and schemas (up to its whole 1 MiB response) become
// garbage as soon as the next server is listed rather than living until the
// resolution ends.
//
// Listing holds one of mcpDiscoverySlots for the duration of the request.
// Each listing briefly holds several copies of a response of up to 1 MiB,
// and the RPC consumer runs ten workers, so without a process-wide bound a
// burst of session starts for AIs with MCP servers would multiply that by
// ten against a 40M container limit.
//
// The slots are shared by every customer, so waiting for one is bounded:
// a resolution spends at most mcpDiscoverySlotWait in total waiting, and a
// server it cannot get a slot for in that time is skipped and logged. Only
// time spent blocked on a slot counts; time spent listing the customer's own
// servers does not. The session-start context has no deadline of its own, so
// without this bound one customer's slow servers holding both slots would
// delay every other customer's session starts by up to the per-server
// timeout each.
//
// Each McpServer in a.McpServerIDs is best-effort: a non-active server is
// silently skipped (no error, no tools), and a ListTools failure for one
// server is logged and skipped rather than failing the whole resolution --
// a single misbehaving customer MCP server must never break the built-in
// tool list or any other server's tools. A tool with an invalid name is
// dropped and the server's other tools are kept, logged once per server.
// Once mcpMaxToolsPerResolution tools are taken, later tools and servers
// are not listed.
func (h *aicallHandler) discoverMcpTools(ctx context.Context, a *ai.AI, keepContent bool) []discoveredMcpTool {
	log := logrus.WithFields(logrus.Fields{
		"func":  "discoverMcpTools",
		"ai_id": a.ID,
	})

	res := []discoveredMcpTool{}
	if h.mcpServerHandler == nil || h.mcptoolHandler == nil {
		return res
	}

	waitLeft := mcpDiscoverySlotWait

	for i, serverID := range a.McpServerIDs {
		if len(res) >= mcpMaxToolsPerResolution {
			log.Warnf("Reached the limit of %d mcp tools per session; %d whitelisted servers were not listed.", mcpMaxToolsPerResolution, len(a.McpServerIDs)-i)
			break
		}

		server, err := h.mcpServerHandler.Get(ctx, serverID)
		if err != nil {
			log.Warnf("Could not get mcp server, skipping its tools. mcp_server_id: %s, err: %v", serverID, err)
			continue
		}

		if ok, why := mcpServerIsUsable(server, a); !ok {
			log.Debugf("Mcp server is not usable, skipping its tools. mcp_server_id: %s, reason: %s", serverID, why)
			continue
		}

		mcpTools, listed, err := h.listToolsWithSlot(ctx, serverID, &waitLeft)
		if !listed {
			if ctx.Err() != nil {
				log.Warnf("Skipped an mcp server: the session start was cancelled while waiting to list it. mcp_server_id: %s", serverID)
			} else {
				log.Warnf("Skipped an mcp server: no discovery slot was free in time. mcp_server_id: %s", serverID)
			}
			continue
		}
		if err != nil {
			log.Warnf("Could not list tools from mcp server, skipping. mcp_server_id: %s, err: %s", serverID, capErrText(err.Error(), 1024))
			continue
		}

		prefix := mcpToolNamePrefix + mcpServerIDShort(serverID) + "_"
		invalidNames := []string{}
		taken := 0
		for _, mt := range mcpTools {
			if !validMcpToolName(mt.Name) {
				invalidNames = append(invalidNames, mt.Name)
				continue
			}
			if len(res) >= mcpMaxToolsPerResolution {
				log.Warnf("Reached the limit of %d mcp tools per session partway through a server's list. mcp_server_id: %s, taken: %d, listed: %d", mcpMaxToolsPerResolution, serverID, taken, len(mcpTools))
				break
			}
			taken++
			d := discoveredMcpTool{
				name: prefix + mt.Name,
				ref:  aicall.McpToolRef{ServerID: serverID, ToolName: mt.Name},
			}
			if keepContent {
				d.description = mt.Description
				d.inputSchema = mt.InputSchema
			}
			res = append(res, d)
		}
		if len(invalidNames) > 0 {
			log.Warnf("Skipped mcp tools with names that are empty, longer than %d bytes, or outside [A-Za-z0-9_-]. mcp_server_id: %s, skipped: %d, first: %s", mcpMaxToolNameLen, serverID, len(invalidNames), sampleToolNames(invalidNames))
		}
	}

	return res
}

// schemaSkips counts one server's tools dropped by decodeToolSchema.
type schemaSkips struct {
	invalid    int
	overBudget int
}

// schemaVerdict is why decodeToolSchema did or did not decode a schema.
type schemaVerdict int

const (
	schemaOK schemaVerdict = iota
	schemaInvalid
	schemaOverBudget
)

const (
	// mcpMaxToolSchemaBytes is the largest input schema, in raw JSON bytes,
	// that is decoded. Decoding into a generic map costs up to about thirty
	// times the raw size, so the limits here are what keep one tools/list
	// from taking a pod near its memory limit.
	mcpMaxToolSchemaBytes = 64 << 10
	// mcpToolSchemaBudgetBytes is the total raw schema size decoded for one
	// resolution, across every server.
	mcpToolSchemaBudgetBytes = 256 << 10
)

// decodeToolSchema decodes one tool's input schema if it fits both the
// per-tool limit and what is left of budget, charging it to budget. A schema
// that is too large or is not a JSON object is schemaInvalid; one that would
// fit on its own but not in what is left of the budget is schemaOverBudget.
// Either way the caller drops that tool and keeps the server's others. An
// absent or null schema decodes to nil and costs nothing.
//
// The budget is shared across the AI's servers in whitelist order, so one
// server's large schemas can leave nothing for a later server. That fails
// closed: the later tools are dropped and logged, never decoded.
func decodeToolSchema(raw json.RawMessage, budget *int) (map[string]any, schemaVerdict) {
	// An explicit null means the same as an absent schema.
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, schemaOK
	}
	if len(raw) > mcpMaxToolSchemaBytes {
		return nil, schemaInvalid
	}
	if len(raw) > *budget {
		return nil, schemaOverBudget
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, schemaInvalid
	}
	*budget -= len(raw)
	return params, schemaOK
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
// that is deleted, inactive, or no longer owned by the AI's customer all
// produce a generic failure rather than routing the call through.
func (h *aicallHandler) toolHandleMcpCall(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleMcpCall",
		"aicall_id": c.ID,
		"tool_name": tc.Function.Name,
	})
	log.Debugf("handling mcp tool call.")

	res := newToolResult(tc.ID)

	ref, ok := lookupMcpToolRef(c, string(tc.Function.Name))
	if !ok {
		log.Debugf("could not resolve mcp tool name from aicall metadata.")
		fillFailed(res, errMcpToolCallFailed("unknown mcp tool call"))
		return res
	}

	// The whitelist that governs this call is the CURRENT team member's.
	// resolveActiveAIForMcp subsumes resolveAI on every path: for AssistanceTypeAI
	// both issue the same single fetch, for a team it tries the current member and
	// then -- where a retry could differ -- the start member that resolveAI alone
	// would have tried, and for any other type both refuse. So a nil here means
	// resolveAI would have failed too; falling back to it would double an RPC on
	// the LLM's critical path and change nothing. Fail closed instead.
	tmpAI := h.resolveActiveAIForMcp(ctx, c)
	if tmpAI == nil {
		// Warn, not Error: the cause is already logged by the resolver, this is a
		// per-call recoverable refusal, and the neighbouring gates below warn too.
		log.Warnf("Could not resolve the ai for the mcp tool call. refusing.")
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
	if ok, why := mcpServerIsUsable(server, tmpAI); !ok {
		log.Warnf("Mcp server is not usable, refusing the call. mcp_server_id: %s, reason: %s", ref.ServerID, why)
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
// resolveMcpToolMap's callers at write time, but AIcall.Metadata may also come
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

// mcpServerIsUsable reports whether a resolved MCP server may serve tools for
// the given AI: it must not be soft-deleted, it must be active, and it must
// belong to the AI's customer.
//
// All three are checked on BOTH the resolution path (discoverMcpTools, which
// decides what the LLM is even told about) and the dispatch path
// (toolHandleMcpCall, which decides what actually runs). The two run at
// different times -- an AIcall can be reused for hours after its tool map was
// built -- so a gate on only one of them still leaves a window where a
// deleted, paused, or reassigned server is reachable.
//
// tm_delete has to be checked here rather than left to the Get: McpServerGet
// returns soft-deleted rows on purpose, because the REST read of a deleted
// server answers 200.
//
// The customer check is not redundant with the whitelist check in
// toolHandleMcpCall. The whitelist proves the AI still lists this server; it
// does not prove the server still belongs to the AI's customer, and stored ids
// outlive the validation that admitted them.
func mcpServerIsUsable(server *mcpserver.McpServer, a *ai.AI) (bool, string) {
	switch {
	case server == nil:
		return false, "not found"
	case server.TMDelete != nil:
		return false, "deleted"
	case server.Status != mcpserver.StatusActive:
		return false, "not active: " + string(server.Status)
	case server.CustomerID != a.CustomerID:
		return false, "owned by another customer"
	}

	return true, ""
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
// unbounded content there either. It cuts on a rune boundary so a multi-byte
// character is never split into invalid UTF-8.
func capErrText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
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

	mcpToolMap := h.resolveMcpToolMap(ctx, a)

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
