package toolhandler

import "monorepo/bin-ai-manager/models/tool"

// ConversationSafeTools is the set of tool names enabled for AIcalls with
// ReferenceTypeConversation. Voice-specific tools (connect_call, stop_media,
// stop_flow) are excluded — they assume a live phone call.
//
// See docs/plans/2026-04-27-conversation-ai-talk-design.md §9 for rationale
// and docs/plans/2026-04-27-conversation-ai-talk-plan.md Slice 0 for the v1
// decision to ship this whitelist as a documented utility (not yet wired to
// the pipecat session payload).
var ConversationSafeTools = map[tool.ToolName]bool{
	tool.ToolNameSendEmail:         true,
	tool.ToolNameSendMessage:       true,
	tool.ToolNameSetVariables:      true,
	tool.ToolNameGetVariables:      true,
	tool.ToolNameStopService:       true,
	tool.ToolNameGetAIcallMessages: true,
	tool.ToolNameSearchKnowledge:   true,
	tool.ToolNameDescribeAction:    true,
}

// FilterToolsForConversation returns the subset of names that are safe for a
// conversation-typed AIcall. ToolNameAll expands to the whitelist (not the
// full registry); when ToolNameAll is present anywhere in the input, the
// output is exactly the whitelist. Otherwise, only names present in
// ConversationSafeTools are kept.
//
// Per Slice 0 (Path A) this is a documented utility and is intentionally not
// yet wired to the pipecat session payload. A future Path B follow-up will
// hook it into the call site.
func FilterToolsForConversation(names []tool.ToolName) []tool.ToolName {
	for _, n := range names {
		if n == tool.ToolNameAll {
			out := make([]tool.ToolName, 0, len(ConversationSafeTools))
			for k := range ConversationSafeTools {
				out = append(out, k)
			}
			return out
		}
	}

	out := make([]tool.ToolName, 0, len(names))
	for _, n := range names {
		if ConversationSafeTools[n] {
			out = append(out, n)
		}
	}
	return out
}
