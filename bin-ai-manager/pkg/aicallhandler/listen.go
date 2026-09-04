package aicallhandler

import (
	"context"
	"strings"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// isListenableCallStatus reports whether a call is in a state transcribe-manager
// will accept. It mirrors transcribehandler.isValidReference's own set exactly;
// diverging would mean starting a transcribe that is then refused.
func isListenableCallStatus(status cmcall.Status) bool {
	return status == cmcall.StatusDialing || status == cmcall.StatusRinging || status == cmcall.StatusProgressing
}

// listenTranscribeIDFromMetadata reads the listen transcribe id off the AIcall's
// metadata, returning uuid.Nil when absent or unparseable.
func listenTranscribeIDFromMetadata(c *aicall.AIcall) uuid.UUID {
	if c.Metadata == nil {
		return uuid.Nil
	}

	tmp, ok := c.Metadata[aicall.MetaKeyListenTranscribeID].(string)
	if !ok {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(tmp)
}

// listenOwnsTranscribeFromMetadata reports whether this AIcall started the
// transcribe session it is listening to, and may therefore stop it.
func listenOwnsTranscribeFromMetadata(c *aicall.AIcall) bool {
	if c.Metadata == nil {
		return false
	}

	owns, ok := c.Metadata[aicall.MetaKeyListenOwnsTranscribe].(bool)
	if !ok {
		return false
	}

	return owns
}

// listenTranscriptNewMarker separates the lines a previous turn already
// evaluated from the ones this turn is seeing for the first time.
//
// Without it the model re-reads the whole window every turn with no way to tell
// what is new, and re-notifies about the same thing repeatedly -- the single
// most likely way this feature becomes annoying rather than useful.
const listenTranscriptNewMarker = "--- NEW SINCE YOUR LAST CHECK ---"

// buildListenTurnMessages assembles a listen evaluation turn's LLM context.
//
// It is built EXPLICITLY, from known-bounded inputs, and getPipecatcallMessages
// is deliberately never called. Two reasons, both structural:
//
//   - The transcript is not, and must never become, message rows. Rows would be
//     webhook-published per spoken sentence, rendered in the agent's panel, and
//     would consume the replay window.
//   - The context size here is a constant, independent of call length. A replay
//     window is not.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.2.
func (h *aicallHandler) buildListenTurnMessages(ctx context.Context, c *aicall.AIcall, window []string, newLines []string) ([]map[string]any, error) {
	res := []map[string]any{}

	// (1) The platform's own Insight guardrails.
	//
	// startInitMessages writes this first for every Insight AIcall, ahead of the
	// customer's prompt -- but the frozen prompt snapshot in Metadata holds ONLY
	// the substituted init_prompt and never captured this. Omitting it would run
	// unsolicited output with none of the "base answers strictly on retrieved
	// data / never expose raw JSON or tool responses / never mention tool names"
	// rules, which is exactly where they matter most. It is a fixed platform
	// constant, so this costs no DB read.
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": InsightSystemPrompt,
	})

	// (2) The customer's own prompt, frozen and already substituted at AIcall
	// start -- so no DB read and no re-substitution here.
	if snapshot := listenPromptSnapshot(c); snapshot != "" {
		res = append(res, map[string]any{
			"role":    string(message.RoleSystem),
			"content": snapshot,
		})
	}

	// (3) The mechanics of a listen turn.
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": ListenTurnSystemPrompt,
	})

	// (4) Recent Q&A, so the AI has continuity with what the agent asked and
	// with its own earlier notifications.
	//
	// Over-fetch and filter in process: ApplyFields builds equality clauses per
	// field and has no IN support, so "role in (user, assistant)" cannot be
	// expressed in the query. FieldDeleted:false IS expressible and is applied,
	// unlike getPipecatcallMessages which does not filter deleted rows today --
	// this is a new code path, so it gets the correct filter rather than
	// inheriting that gap.
	qaRowsDesc, err := h.messageHandler.List(ctx, 30, "", map[message.Field]any{
		message.FieldAIcallID: c.ID,
		message.FieldDeleted:  false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not get the qa messages")
	}

	budget := config.Get().AIcallListenQAContextSize
	qa := []map[string]any{}
	// qaRowsDesc is newest-first; walk it that way, take the newest `budget`
	// conversational rows, then reverse into chronological order for the LLM.
	for _, m := range qaRowsDesc {
		if len(qa) >= budget {
			break
		}
		if m.Role != message.RoleUser && m.Role != message.RoleAssistant {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			// Empty-content rows are the tool-call carriers; they have no
			// conversational value here and would waste the budget.
			continue
		}
		qa = append(qa, map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		})
	}
	for i, j := 0, len(qa)-1; i < j; i, j = i+1, j-1 {
		qa[i], qa[j] = qa[j], qa[i]
	}
	res = append(res, qa...)

	// (5) The transcript block.
	res = append(res, map[string]any{
		"role":    string(message.RoleUser),
		"content": buildListenTranscriptBlock(window, newLines),
	})

	return res, nil
}

// buildListenTranscriptBlock renders the rolling window with the new lines
// marked off.
//
// The window already contains the new lines (both lists are appended to on
// intake), so the seen portion is the window minus its own tail.
func buildListenTranscriptBlock(window []string, newLines []string) string {
	seen := window
	if len(newLines) > 0 && len(window) >= len(newLines) {
		seen = window[:len(window)-len(newLines)]
	}

	var sb strings.Builder
	sb.WriteString("Live call transcript so far:\n")
	for _, line := range seen {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(listenTranscriptNewMarker)
	sb.WriteString("\n")
	for _, line := range newLines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// listenPromptSnapshot returns the frozen, already-substituted customer prompt
// for this AIcall.
//
// For AssistanceTypeAI there is exactly one snapshot. For AssistanceTypeTeam
// there is one per member, and the right one is whichever matches
// CurrentMemberID -- falling back to the first, because a listen turn with the
// wrong team member's prompt is still far better than one with no customer
// instructions at all.
func listenPromptSnapshot(c *aicall.AIcall) string {
	if c.Metadata == nil {
		return ""
	}

	raw, ok := c.Metadata[aicall.MetaKeyPromptSnapshots].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}

	first := ""
	for _, item := range raw {
		snapshot, okItem := item.(map[string]any)
		if !okItem {
			continue
		}

		prompt, _ := snapshot["prompt"].(string)
		if prompt == "" {
			continue
		}
		if first == "" {
			first = prompt
		}

		memberID, _ := snapshot["member_id"].(string)
		if c.CurrentMemberID != uuid.Nil && memberID == c.CurrentMemberID.String() {
			return prompt
		}
	}

	return first
}
