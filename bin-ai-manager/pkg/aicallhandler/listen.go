package aicallhandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/pkg/utilhandler"
	kmkase "monorepo/bin-contact-manager/models/kase"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

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
	if c == nil || c.Metadata == nil {
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
	turnPrompt := ListenTurnSystemPrompt
	if listenKindOf(c) == listenKindConversation {
		turnPrompt = ListenTurnConversationSystemPrompt
	}
	res = append(res, map[string]any{
		"role":    string(message.RoleSystem),
		"content": turnPrompt,
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
		"content": buildListenTranscriptBlock(listenTranscriptHeader(listenKindOf(c)), window, newLines),
	})

	return res, nil
}

// buildListenTranscriptBlock renders the rolling window plus the new lines
// under a kind-specific header. The header is a parameter (rather than a
// second wrapper function) because the only caller is buildListenTurnMessages
// and an unused wrapper would fail golangci-lint's unused check.
//
// The window already contains the new lines (both lists are appended to on
// intake), so the seen portion is the window minus its own tail.
func buildListenTranscriptBlock(header string, window []string, newLines []string) string {
	seen := window
	if len(newLines) > 0 && len(window) >= len(newLines) {
		seen = window[:len(window)-len(newLines)]
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
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
// Conversation-kind AIcalls additionally pass an empty-buffer short-circuit and
// a turn-time Case status check before anything is counted or popped.
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
		promListenTurnTotal.WithLabelValues(listenKindLabelUnknown, "skipped_invalid").Inc()
		return
	}
	kind := listenKindOf(c)
	kindLabel := kind.label()

	if c.Status != aicall.StatusProgressing ||
		c.ReferenceType != aicall.ReferenceTypeContactCase ||
		kind == listenKindNone {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_invalid").Inc()
		return
	}

	if kind == listenKindConversation {
		// Empty short-circuit BEFORE the Case RPC and BEFORE the turn counter:
		// a deferred flush that finds nothing must cost neither (§5.4). The
		// check is non-atomic with the pop below; a line landing in between only
		// means one turn is counted normally.
		pending, errLen := h.cache.ListenPendingLen(ctx, aicallID)
		if errLen != nil {
			log.Warnf("Could not read the pending buffer length; proceeding as non-empty. err: %v", errLen)
		} else if pending == 0 {
			promListenTurnTotal.WithLabelValues(kindLabel, "skipped_empty").Inc()
			return
		}

		// Turn-time Case status check: contact-manager publishes nothing on
		// Close, so this is the conversation kind's primary stop signal (§5.5.2).
		kase, errCase := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
		if errCase != nil {
			log.Errorf("Could not get the case for the listen turn. err: %v", errCase)
			promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
			return
		}
		if kase.Status == kmkase.StatusClosed {
			h.stopListening(ctx, c)
			promListenTurnTotal.WithLabelValues(kindLabel, "skipped_case_closed").Inc()
			return
		}
	}

	// Hard backstop against a pathologically long call. Reaching it stops
	// listening cleanly; the Q&A panel keeps working normally.
	turns, errCount := h.cache.ListenTurnCountIncr(ctx, aicallID, listenBufferTTL())
	if errCount != nil {
		log.Warnf("Could not increment the listen turn counter. err: %v", errCount)
	} else if turns > int64(config.Get().AIcallListenMaxTurnsPerAIcall) {
		h.stopListening(ctx, c)
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_cap").Inc()
		return
	}

	lines, err := h.cache.ListenPendingPopAll(ctx, aicallID)
	if err != nil {
		log.Errorf("Could not drain the pending buffer. err: %v", err)
		promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
		return
	}
	if len(lines) == 0 {
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_empty").Inc()
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

	kindLabel := listenKindOf(c).label()

	window, errWindow := h.cache.ListenWindowGet(ctx, c.ID)
	if errWindow != nil {
		log.Warnf("Could not read the transcript window; evaluating the new lines alone. err: %v", errWindow)
		window = lines
	}

	llmMessages, err := h.buildListenTurnMessages(ctx, c, window, lines)
	if err != nil {
		log.Errorf("Could not build the listen turn context. err: %v", err)
		promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
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
		promListenTurnTotal.WithLabelValues(kindLabel, "skipped_register_failed").Inc()
		return
	}

	pc, err := h.startListenPipecatcall(ctx, c, turnPipecatcallID, llmMessages)
	if err != nil {
		log.Errorf("Could not start the listen pipecatcall. err: %v", err)
		promListenTurnTotal.WithLabelValues(kindLabel, "failed").Inc()
		return
	}

	if errTerm := h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultListenTurnTimeout); errTerm != nil {
		// Non-fatal: the turn already ran. A missed terminate leaves one idle
		// session that its own timeout will reap.
		log.Warnf("Could not schedule the listen pipecatcall terminate. err: %v", errTerm)
	}

	promListenTurnTotal.WithLabelValues(kindLabel, "ran").Inc()
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

// stopListening stops this AIcall listening. It is EXACTLY two steps, in order,
// and nothing else:
//
//  1. If this AIcall owns the transcribe session, stop it.
//  2. Clear the listen state.
//
// IT NEVER CALLS ProcessTerminate. That function ends the AIcall ITSELF (status
// to terminated plus an activeflow service stop), which would kill the agent's
// entire Insight Q&A session. Stopping listening must leave the panel working
// normally -- the agent can still ask questions about a call that just ended,
// and that is often exactly when they do.
func (h *aicallHandler) stopListening(ctx context.Context, c *aicall.AIcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "stopListening",
		"aicall_id": c.ID,
	})

	if listenOwnsTranscribeFromMetadata(c) {
		transcribeID := listenTranscribeIDFromMetadata(c)
		if transcribeID != uuid.Nil {
			// HostID is fetched FRESH rather than read from a persisted column,
			// precisely because it is regenerated on every transcribe-manager
			// restart -- a stale one addresses a queue that no longer exists.
			tr, errGet := h.reqHandler.TranscribeV1TranscribeGet(ctx, transcribeID)
			if errGet != nil {
				log.Warnf("Could not get the transcribe to stop it. err: %v", errGet)
				promListenStopFailedTotal.Inc()
			} else if tr.Status == tmtranscribe.StatusProgressing {
				if _, errStop := h.reqHandler.TranscribeV1TranscribeStop(ctx, tr.HostID, tr.ID); errStop != nil {
					// NON-FATAL, and the fallback is stated rather than assumed:
					// if the owning pod restarted, its per-pod request queue no
					// longer exists and this times out. The session's audio
					// transport still ends when the call itself ends (hanging up
					// closes the Asterisk WebSocket feeding the STT stream), so
					// the failure mode is a slightly-longer-than-necessary STT
					// session, never a permanently orphaned one.
					log.Warnf("Could not stop the transcribe. err: %v", errStop)
					promListenStopFailedTotal.Inc()
				}
			}
		}
	}

	h.clearListenState(ctx, c)
}

// clearListenState removes every trace of this AIcall's listening state.
//
// STEP ORDER IS LOAD-BEARING. Step 2 needs the transcribe id and step 3 destroys
// it, so reading and using it must come first. Getting this backwards leaves a
// stale (transcribe_id, aicall_id) pairing that intake can still match, feeding
// segments to an AIcall that stopped listening.
func (h *aicallHandler) clearListenState(ctx context.Context, c *aicall.AIcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "clearListenState",
		"aicall_id": c.ID,
	})

	// 1. Read the transcribe id from the AIcall already in hand -- no extra
	//    fetch; every caller holds `c`.
	transcribeID := listenTranscribeIDFromMetadata(c)

	// 2. Remove ONLY this AIcall's resolver membership. Never DEL the key: a
	//    shared transcribe must stay resolvable for whichever AIcalls are still
	//    listening to it. Redis drops the key once the set empties.
	if transcribeID != uuid.Nil {
		if errRem := h.cache.ListenAIcallIDRemove(ctx, transcribeID, c.ID); errRem != nil {
			log.Warnf("Could not remove the listen resolver membership. err: %v", errRem)
		}
	}

	if conversationID := listenConversationIDFromMetadata(c); conversationID != uuid.Nil {
		if errRem := h.cache.ListenConversationAIcallIDRemove(ctx, conversationID, c.ID); errRem != nil {
			log.Warnf("Could not remove the conversation listen resolver membership. err: %v", errRem)
		}
	}

	// The per-AIcall keys are never shared, so a plain delete is correct for
	// them. The turn-id set is deliberately NOT deleted: it is short-TTL and
	// self-expiring, and a tool call arriving late for an already-stopped turn
	// still correctly resolves as a listen turn -- which is exactly what it was.
	if errClear := h.cache.ListenStateClear(ctx, c.ID); errClear != nil {
		log.Warnf("Could not clear the listen state keys. err: %v", errClear)
	}

	// 3. Clear the DB bookkeeping LAST, since step 2 consumed the value it
	//    holds. AIcallUpdateNoTouchTMUpdate, not AIcallUpdate: listening stops
	//    on hangup, which is exactly when an agent is most likely to type a
	//    follow-up question, and a tm_update bump here would have Send's cooldown
	//    reject it.
	metadata := map[string]any{}
	for k, v := range c.Metadata {
		if k == aicall.MetaKeyListenTranscribeID || k == aicall.MetaKeyListenOwnsTranscribe || k == aicall.MetaKeyListenConversationID {
			continue
		}
		metadata[k] = v
	}

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, c.ID, map[aicall.Field]any{
		aicall.FieldListenCallID: uuid.Nil,
		aicall.FieldMetadata:     metadata,
	}); errUpdate != nil {
		log.Warnf("Could not clear the listen state on the aicall row. err: %v", errUpdate)
	}
}

// listenStopPageSize is one page of stopListenByCallID's sweep.
//
// Small on purpose: the realistic population is 1-2 AIcalls (two Cases open on
// one call), so this is sized for the normal case and the loop below handles
// the tail rather than the page size being inflated to cover it.
const listenStopPageSize = 10

// listenStopMaxPages bounds that sweep.
//
// A hangup handler must terminate. 200 listening AIcalls on ONE call is far
// beyond anything the product can produce -- reaching this bound means
// something is wrong (a corrupted listen_call_id, a runaway Case creator), and
// the right response is to stop and say so loudly, not to page forever inside
// an event handler.
const listenStopMaxPages = 20

// stopListenByCallID stops every AIcall listening to the given call.
//
// PLURAL ON PURPOSE: the active-AIcall unique key is per Case, not per customer,
// so two Cases open on one call each get their own AIcall and both must be
// cleared.
//
// Before clearing, it runs one final flush turn per AIcall with a non-empty
// buffer. The debounce means the last few lines before a hangup are still
// sitting unevaluated, and this is the only chance to read them -- there is no
// next segment coming. It bypasses the debounce lock deliberately, which is why
// the turn body was extracted as runListenTurnWithLines.
//
// IT PAGINATES, and that is not theoretical tidiness (review round 1 finding
// MEDIUM-2). A single fixed page silently dropped every row past the first
// page, and a dropped row is an AIcall left believing it is still listening: its
// resolver membership survives, its listen_call_id survives, and if it owned the
// transcribe session that session is never stopped.
func (h *aicallHandler) stopListenByCallID(ctx context.Context, callID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "stopListenByCallID",
		"call_id": callID,
	})

	// A ZERO CALL ID MUST NEVER REACH THE QUERY (review round 1 security
	// finding MEDIUM-2). callID comes from an unvalidated call_hangup event
	// field, and the migration declares listen_call_id NOT NULL DEFAULT
	// 0x00... -- i.e. uuid.Nil is the value EVERY non-listening contact_case
	// AIcall carries, across every tenant. A malformed or mis-routed event with
	// a zero id would therefore select an arbitrary page of unrelated tenants'
	// non-listening AIcalls and run the whole teardown against them.
	if callID == uuid.Nil {
		log.Warn("Ignoring a listen stop request for a nil call id.")
		return
	}

	filters := map[aicall.Field]any{
		aicall.FieldReferenceType: aicall.ReferenceTypeContactCase,
		aicall.FieldListenCallID:  callID,
		aicall.FieldDeleted:       false,
	}

	// Keyset pagination on tm_create desc, the same shape AIcallList itself
	// implements (WHERE tm_create < token). utilhandler.ISO8601Layout, NOT
	// RFC3339Nano: AIcallList's own default token is TimeGetCurTime(), which
	// uses this fixed-precision layout, and a token built with a different
	// layout would not error -- it would silently mis-paginate.
	token := ""
	for page := 0; page < listenStopMaxPages; page++ {
		listening, err := h.List(ctx, listenStopPageSize, token, filters)
		if err != nil {
			log.Errorf("Could not list the listening aicalls. page: %d, err: %v", page, err)
			return
		}

		for _, c := range listening {
			lines, errPop := h.cache.ListenPendingPopAll(ctx, c.ID)
			if errPop != nil {
				log.Warnf("Could not drain the pending buffer for the final flush. aicall_id: %s, err: %v", c.ID, errPop)
			} else if len(lines) > 0 {
				h.runListenTurnWithLines(ctx, c, lines)
			}

			h.stopListening(ctx, c)
		}

		if len(listening) < listenStopPageSize {
			// A short page proves the source is exhausted -- this is the
			// overwhelmingly common exit, on the very first iteration.
			return
		}

		last := listening[len(listening)-1]
		if last.TMCreate == nil {
			// Cannot construct a continuation token. Defensive only: the query
			// itself is WHERE tm_create < token, and NULL < value is NULL (not
			// TRUE) under SQL three-valued logic, so such a row cannot appear
			// on any page after the first.
			log.Warnf("Could not build the next page token; the listen stop sweep may be incomplete. aicall_id: %s", last.ID)
			return
		}
		token = last.TMCreate.UTC().Format(utilhandler.ISO8601Layout)
	}

	log.Warnf("The listen stop sweep hit its page budget; some listening aicalls may not have been stopped. max_pages: %d", listenStopMaxPages)
}

// speakerTag renders a transcript segment's direction as a structural speaker
// label.
//
// The labels are STRUCTURAL, not localized, so prompt behaviour does not fork by
// call language.
//
// The mapping (design §5.9, wording corrected in review round 11 finding
// LOW-2 to match Asterisk's actual read/write convention unambiguously):
// transcript.Direction is relative to the transcribed CHANNEL's own read/write
// direction -- "in" is audio Asterisk reads FROM that channel (what the
// channel's own party said), "out" is audio Asterisk writes TO it (what was
// played to them). The listened channel's own party is always Case.Peer --
// case_create only ever creates a Case from a CRM-eligible peer (never an
// internal agent/extension/SIP/conference/AI endpoint), so in=CUSTOMER is a
// code-checked invariant, not a bare assumption resting on which leg happens
// to be transcribed. Once that leg is bridged to an agent, out=AGENT follows.
//
// Depends on waitForConfbridgeReady already having confirmed a live,
// exactly-2-party confbridge before listening starts -- with 3+ parties in the
// bridge, "out" stops reliably meaning "the agent" (design §5.1.1 step 7's
// closing note); "in" is unaffected regardless of party count, since it never
// depended on who else is in the bridge.
//
// MAPPING STATUS: the general channel-relative mechanism was independently
// confirmed against real production transcript data during design review
// (design §5.9), and which leg is transcribed is confirmed structurally, not
// assumed. What is NOT yet confirmed end-to-end is one real agent-bridged
// call's transcript segments against known speaker identity -- the
// implementation plan's Task 0 Step 1 records that as PROVISIONAL, proceeding
// on the structural reading, with empirical verification tracked separately.
// Test_speakerTag pins whichever mapping that check confirms; if it comes back
// reversed, this function and that test flip together.
//
// DirectionBoth (and anything unrecognised) is tagged [SPEAKER] rather than
// guessed -- a wrong attribution is worse than an unattributed line.
func speakerTag(direction tmtranscript.Direction) string {
	switch direction {
	case tmtranscript.DirectionIn:
		return "[CUSTOMER]"
	case tmtranscript.DirectionOut:
		return "[AGENT]"
	default:
		return "[SPEAKER]"
	}
}

// EventTMTranscriptCreated is layer 1: transcript intake.
//
// NO LLM, NO DB WRITE, NO WEBHOOK. This runs for every final STT result
// PLATFORM-WIDE -- flow-driven, summary-driven, customer-started -- not just for
// calls being listened to, so the per-event cost has to stay at one Redis round
// trip. An empty resolver set means "not a session we started" and is the
// overwhelmingly common outcome.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.3.
func (h *aicallHandler) EventTMTranscriptCreated(ctx context.Context, evt *tmtranscript.Transcript) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "EventTMTranscriptCreated",
		"transcribe_id": evt.TranscribeID,
	})

	// transcripthandler.dbDelete publishes EventTypeTranscriptCreated on DELETE
	// as well as on create -- a known upstream bug this design defends against
	// rather than fixes, because changing the emitted event type is a
	// routing-key-visible change affecting every current subscriber. Without
	// this guard a deleted line replays into the LLM as freshly-spoken content.
	if evt.TMDelete != nil || strings.TrimSpace(evt.Message) == "" {
		promListenSegmentTotal.WithLabelValues("dropped_deleted").Inc()
		return
	}

	aicallIDs, err := h.cache.ListenAIcallIDsGet(ctx, evt.TranscribeID)
	if err != nil {
		log.Warnf("Could not resolve the listening aicalls. err: %v", err)
		promListenSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}
	if len(aicallIDs) == 0 {
		promListenSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}

	line := fmt.Sprintf("%s %s", speakerTag(evt.Direction), strings.TrimSpace(evt.Message))
	ttl := listenBufferTTL()
	windowSize := config.Get().AIcallListenWindowSize
	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second

	// Fan out per listening AIcall. Two Cases open on one call each get their
	// own AIcall and each buffers and debounces independently.
	for _, aicallID := range aicallIDs {
		if errPending := h.cache.ListenPendingPush(ctx, aicallID, line, ttl); errPending != nil {
			log.Warnf("Could not buffer the pending line. aicall_id: %s, err: %v", aicallID, errPending)
			continue
		}
		if errWindow := h.cache.ListenWindowPush(ctx, aicallID, line, windowSize, ttl); errWindow != nil {
			log.Warnf("Could not buffer the window line. aicall_id: %s, err: %v", aicallID, errWindow)
		}
		promListenSegmentTotal.WithLabelValues("buffered").Inc()

		// Leaky-bucket debounce. Losing the race is the NORMAL case and is not
		// an error -- the line stays buffered for whichever turn did win, which
		// is exactly what decouples LLM invocations from speech volume.
		acquired, errLock := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
		if errLock != nil {
			log.Warnf("Could not take the listen turn lock. aicall_id: %s, err: %v", aicallID, errLock)
			continue
		}
		if !acquired {
			promListenTurnTotal.WithLabelValues(string(listenKindCall), "skipped_locked").Inc()
			continue
		}

		// Detached: this handler must return promptly, and the turn's own
		// lifetime is bounded by its pipecatcall terminate.
		go func(id uuid.UUID) {
			turnCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			h.RunListenTurn(turnCtx, id)
		}(aicallID)
	}
}
