package builder

import (
	"fmt"
	"unicode/utf8"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// invalid builds the single error type this package returns. The message names
// the violated rule but never echoes any caller-supplied text, so it is safe to
// log and to return to the client.
func invalid(format string, args ...any) error {
	return cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, ReasonInvalidArgument, fmt.Sprintf(format, args...))
}

// totalRunes counts the runes of every caller-controlled string that reaches
// the model: message contents and all current_draft fields (tool names too).
func totalRunes(req *ChatRequest) int {
	n := 0
	for _, m := range req.Messages {
		n += utf8.RuneCountInString(m.Content)
	}
	if d := req.CurrentDraft; d != nil {
		n += utf8.RuneCountInString(d.Name) + utf8.RuneCountInString(d.Detail) + utf8.RuneCountInString(d.InitPrompt)
		for _, t := range d.ToolNames {
			n += utf8.RuneCountInString(t)
		}
	}
	return n
}

// ValidateRequest checks the request against the shared limits. Failures are
// InvalidArgument with reason INVALID_ARGUMENT (design 4.7).
//
// Roles must be "user" or "assistant" but strict alternation is not enforced:
// the only effect of a skewed history is the caller's own user_turns value.
// Empty content is rejected because it would only produce a provider error.
func ValidateRequest(req *ChatRequest) error {
	if req == nil {
		return invalid("request is required")
	}
	if len(req.Messages) == 0 {
		return invalid("messages must not be empty")
	}
	if len(req.Messages) > MaxMessages {
		return invalid("messages exceeds the limit of %d", MaxMessages)
	}
	for i, m := range req.Messages {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			return invalid("messages[%d].role must be %q or %q", i, RoleUser, RoleAssistant)
		}
		if m.Content == "" {
			return invalid("messages[%d].content must not be empty", i)
		}
		if utf8.RuneCountInString(m.Content) > MaxMessageRunes {
			return invalid("messages[%d].content exceeds %d characters", i, MaxMessageRunes)
		}
	}
	if req.Messages[len(req.Messages)-1].Role != RoleUser {
		return invalid("the last message must be from the user")
	}
	if d := req.CurrentDraft; d != nil {
		if utf8.RuneCountInString(d.Name) > MaxNameRunes {
			return invalid("current_draft.name exceeds %d characters", MaxNameRunes)
		}
		if utf8.RuneCountInString(d.Detail) > MaxDetailRunes {
			return invalid("current_draft.detail exceeds %d characters", MaxDetailRunes)
		}
		if utf8.RuneCountInString(d.InitPrompt) > MaxInitPromptRunes {
			return invalid("current_draft.init_prompt exceeds %d characters", MaxInitPromptRunes)
		}
		// Empty names cost zero runes, so the aggregate cap alone does not bound
		// the list. current_draft is always the previous response's draft, which
		// the parser has already cut down to allow-listed tools, so a real one
		// never has more tools than the allow-list (edits the user makes in the
		// form are not sent back, design 4.2).
		if len(d.ToolNames) > len(AllowedTools) {
			return invalid("current_draft.tool_names exceeds %d entries", len(AllowedTools))
		}
	}
	if totalRunes(req) > MaxTotalRunes {
		return invalid("total input exceeds %d characters", MaxTotalRunes)
	}
	return nil
}
