package aicallhandler

import (
	"context"
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

// defaultListenTurnTimeout is how long a listen evaluation turn's pipecatcall
// may live before it is terminated, in milliseconds.
const defaultListenTurnTimeout = 60000

// RunListenTurn evaluates whatever has been said since the last turn.
//
// Preconditions first, then an ATOMIC drain, then the turn body. The drain is a
// single LPOP-count command precisely so a line pushed concurrently cannot be
// lost between a read and a trim.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.
func (h *aicallHandler) RunListenTurn(ctx context.Context, aicallID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RunListenTurn",
		"aicall_id": aicallID,
	})

	c, err := h.Get(ctx, aicallID)
	if err != nil {
		// No AIcall means nothing to stop and nothing to evaluate.
		promListenTurnTotal.WithLabelValues("skipped_invalid").Inc()
		return
	}

	// The flag check lives HERE, in the require-list, not in a separate earlier
	// step: everything a failing condition does next needs `c`, which does not
	// exist until the fetch above. It is also what makes a rollback real -- with
	// no flag read on this path, a session that started while the flag was on
	// would run to call-end or the turn cap regardless.
	if !config.Get().AIcallListenEnabled {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_disabled").Inc()
		return
	}

	if c.Status != aicall.StatusProgressing ||
		c.ReferenceType != aicall.ReferenceTypeContactCase ||
		listenTranscribeIDFromMetadata(c) == uuid.Nil {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_invalid").Inc()
		return
	}

	// Hard backstop against a pathologically long call. Reaching it stops
	// listening cleanly; the Q&A panel keeps working normally.
	turns, errCount := h.cache.ListenTurnCountIncr(ctx, aicallID, listenBufferTTL())
	if errCount != nil {
		log.Warnf("Could not increment the listen turn counter. err: %v", errCount)
	} else if turns > int64(config.Get().AIcallListenMaxTurnsPerAIcall) {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues("skipped_cap").Inc()
		return
	}

	lines, err := h.cache.ListenPendingPopAll(ctx, aicallID)
	if err != nil {
		log.Errorf("Could not drain the pending buffer. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}
	if len(lines) == 0 {
		promListenTurnTotal.WithLabelValues("skipped_empty").Inc()
		return
	}

	h.runListenTurnWithLines(ctx, c, lines)
}

// runListenTurnWithLines is the turn body, taking the lines to evaluate rather
// than draining them.
//
// EXTRACTED DELIBERATELY, and this is not a refactoring nicety. The hangup path
// must evaluate the last words of a call immediately, without waiting for the
// debounce lock and without a buffer left to drain -- it has already drained
// it. RunListenTurn owns the preconditions, the counter, the lock and the
// drain; this owns the evaluation. Two callers, one turn body, no duplicated
// LLM-invocation logic.
func (h *aicallHandler) runListenTurnWithLines(ctx context.Context, c *aicall.AIcall, lines []string) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "runListenTurnWithLines",
		"aicall_id": c.ID,
	})

	window, errWindow := h.cache.ListenWindowGet(ctx, c.ID)
	if errWindow != nil {
		log.Warnf("Could not read the transcript window; evaluating the new lines alone. err: %v", errWindow)
		window = lines
	}

	llmMessages, err := h.buildListenTurnMessages(ctx, c, window, lines)
	if err != nil {
		log.Errorf("Could not build the listen turn context. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}

	// A FRESH, THROWAWAY id -- never written to c.PipecatcallID. That single
	// decision is what keeps this whole design safe:
	//   - no AIcallUpdate per turn, so no tm_update bump and no Send cooldown
	//     interference,
	//   - interruptPreviousPipecatcall is never called, so an in-flight answer
	//     to the agent is never killed,
	//   - and the id mismatch itself becomes the drop signal for any text this
	//     turn emits (see messagehandler's foreign-pipecatcall guard).
	//
	// Tool calls still route correctly: pipecat-manager resolves the AIcall from
	// the PIPECATCALL's ReferenceID (= c.ID), not from AIcall.PipecatcallID.
	turnPipecatcallID := h.utilHandler.UUIDCreate()

	// Register it as a genuine listen turn BEFORE starting the session, and
	// ABORT if that fails. Proceeding unregistered would make every tool call
	// this turn issues resolve listenTurn=false: its rows would be permanently
	// tagged OriginNone (never excluded from future Q&A replay) and its
	// notify_agent call would be rejected -- precisely the failure the
	// registration exists to prevent, reintroduced through the one write this
	// function owns.
	ttl := time.Duration(config.Get().AIcallListenTurnPipecatcallIDTTLSeconds) * time.Second
	if errAdd := h.cache.ListenTurnPipecatcallIDAdd(ctx, c.ID, turnPipecatcallID, ttl); errAdd != nil {
		log.Warnf("Could not register the listen turn id; skipping this turn. err: %v", errAdd)
		promListenTurnTotal.WithLabelValues("skipped_register_failed").Inc()
		return
	}

	pc, err := h.startListenPipecatcall(ctx, c, turnPipecatcallID, llmMessages)
	if err != nil {
		log.Errorf("Could not start the listen pipecatcall. err: %v", err)
		promListenTurnTotal.WithLabelValues("failed").Inc()
		return
	}

	if errTerm := h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultListenTurnTimeout); errTerm != nil {
		// Non-fatal: the turn already ran. A missed terminate leaves one idle
		// session that its own timeout will reap.
		log.Warnf("Could not schedule the listen pipecatcall terminate. err: %v", errTerm)
	}

	promListenTurnTotal.WithLabelValues("ran").Inc()
}

// startListenPipecatcall is a sibling of startPipecatcall that takes the
// pipecatcall id and the message list as parameters, instead of reading
// c.PipecatcallID and calling getPipecatcallMessages.
//
// STTTypeNone and TTSTypeNone are not incidental: a listen turn has no audio
// legs at all. It is a text-in, tool-call-out evaluation. That is also why the
// two STT-driven pipecat message handlers need no foreign-pipecatcall guard --
// a listen turn structurally cannot produce their events.
func (h *aicallHandler) startListenPipecatcall(ctx context.Context, c *aicall.AIcall, pipecatcallID uuid.UUID, llmMessages []map[string]any) (*pmpipecatcall.Pipecatcall, error) {
	res, err := h.reqHandler.PipecatV1PipecatcallStart(
		ctx,
		pipecatcallID,
		c.CustomerID,
		c.ActiveflowID,
		pmpipecatcall.ReferenceTypeAICall,
		c.ID,
		pmpipecatcall.LLMType(c.AIEngineModel),
		llmMessages,
		pmpipecatcall.STTTypeNone,
		"",
		pmpipecatcall.TTSTypeNone,
		"",
		"",
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not start the listen pipecatcall")
	}

	return res, nil
}

// listenBufferTTL is the TTL applied to the pending buffer, the rolling window,
// the debounce lock and the turn counter. The listen-turn id set uses its own,
// much shorter TTL -- it only needs to outlive one turn.
func listenBufferTTL() time.Duration {
	return time.Duration(config.Get().AIcallListenBufferTTLHours) * time.Hour
}

// stopListening is implemented in Task 25 of the implementation plan; this
// no-op stub keeps this commit green on its own.
// TODO(Task 25): replace with the real two-step stop (owned-transcribe stop,
// then clearListenState).
func (h *aicallHandler) stopListening(ctx context.Context, c *aicall.AIcall) {
	_, _ = ctx, c
}
