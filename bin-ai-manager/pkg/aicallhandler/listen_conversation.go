package aicallhandler

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// checkListenEligibleConversation is checkListenEligible's step-5 branch for a
// Case created from a messaging conversation (design 2026-09-05 §5.1, §5.1.1).
//
// It runs the whole start INLINE. There is no external session to create, so
// none of the call branch's goroutine, confbridge poll or start lock applies.
// The caller returns proceed=false regardless of the outcome; every outcome is
// metered here under kind="conversation".
func (h *aicallHandler) checkListenEligibleConversation(ctx context.Context, c *aicall.AIcall, kase *kmkase.Case) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "checkListenEligibleConversation",
		"aicall_id": c.ID,
		"case_id":   kase.ID,
	})

	if !config.Get().AIcallListenConversationEnabled {
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_disabled").Inc()
		return
	}

	conversationID := uuid.FromStringOrNil(kase.ReferenceID)
	if conversationID == uuid.Nil {
		// flow-manager's case_create may legitimately store an empty ReferenceID.
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_not_listenable").Inc()
		return
	}

	// Defence in depth. The Case is already tenant-checked upstream, but the
	// conversation id is a free-form ReferenceID string on that row, so assert
	// the conversation itself belongs to the same customer before registering
	// this AIcall as one of its listeners.
	cv, errGet := h.reqHandler.ConversationV1ConversationGet(ctx, conversationID)
	if errGet != nil {
		log.Errorf("Could not get the conversation. err: %v", errGet)
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "failed").Inc()
		return
	}
	if cv.CustomerID != c.CustomerID || cv.CustomerID == uuid.Nil {
		log.Warnf("Cross-customer conversation access blocked. conversation_customer_id: %s", cv.CustomerID)
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_not_listenable").Inc()
		return
	}

	// A Case that is already closed never starts listening. RunListenTurn's
	// own Case check stays the runtime stop for a Case closed mid-session.
	if kase.Status == kmkase.StatusClosed {
		promListenStartTotal.WithLabelValues(string(listenKindConversation), "skipped_not_listenable").Inc()
		return
	}

	h.startListenConversation(ctx, c, conversationID)
}

// startListenConversation registers c as a listener of conversationID: an
// idempotency check, then two idempotent writes (resolver SADD, metadata
// pointer). Returns the metered result for tests.
func (h *aicallHandler) startListenConversation(ctx context.Context, c *aicall.AIcall, conversationID uuid.UUID) string {
	log := logrus.WithFields(logrus.Fields{
		"func":            "startListenConversation",
		"aicall_id":       c.ID,
		"conversation_id": conversationID,
	})
	kindLabel := string(listenKindConversation)

	// Step 0: pointer already set AND resolver membership present -> reused.
	// Pointer set but membership missing (Redis flush) falls through: the SADD
	// below re-registers and the metadata rewrite is a no-op.
	if listenConversationIDFromMetadata(c) == conversationID {
		member, errMember := h.cache.ListenConversationAIcallIDIsMember(ctx, conversationID, c.ID)
		if errMember != nil {
			log.Warnf("Could not check the conversation listen membership; starting fresh. err: %v", errMember)
		} else if member {
			promListenStartTotal.WithLabelValues(kindLabel, "reused").Inc()
			return "reused"
		}
	}

	// Step 1: resolver membership, before the DB pointer, so nothing can route a
	// message at an AIcall the resolver does not yet know.
	if errAdd := h.cache.ListenConversationAIcallIDAdd(ctx, conversationID, c.ID, listenResolverTTL); errAdd != nil {
		log.Errorf("Could not add the conversation listen resolver membership. err: %v", errAdd)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}

	// Step 2: persist the pointer. Re-read the row first so a concurrent
	// metadata write is merged rather than clobbered (same shape as
	// UpdateListenState for the call kind).
	cur, errGet := h.db.AIcallGet(ctx, c.ID)
	if errGet != nil {
		log.Errorf("Could not re-read the aicall before writing the listen pointer. err: %v", errGet)
		h.rollbackListenConversation(ctx, conversationID, c.ID)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}
	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyListenConversationID] = conversationID.String()

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, c.ID, map[aicall.Field]any{
		aicall.FieldMetadata: metadata,
	}); errUpdate != nil {
		log.Errorf("Could not write the conversation listen pointer. err: %v", errUpdate)
		h.rollbackListenConversation(ctx, conversationID, c.ID)
		promListenStartTotal.WithLabelValues(kindLabel, "failed").Inc()
		return "failed"
	}

	promListenStartTotal.WithLabelValues(kindLabel, "started").Inc()
	return "started"
}

// rollbackListenConversation is the best-effort SREM after a failed start. If it
// fails too, the entry expires with listenResolverTTL and the first line that
// reaches RunListenTurn's predicate clears it.
func (h *aicallHandler) rollbackListenConversation(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) {
	if errRem := h.cache.ListenConversationAIcallIDRemove(ctx, conversationID, aicallID); errRem != nil {
		logrus.WithFields(logrus.Fields{"func": "rollbackListenConversation", "aicall_id": aicallID, "conversation_id": conversationID}).
			Warnf("Could not roll back the conversation listen resolver membership. err: %v", errRem)
	}
}

// speakerTagForDirection maps a conversation message's direction to the
// structural speaker tag the listen turn prompt expects.
func speakerTagForDirection(direction cvmessage.Direction) string {
	switch direction {
	case cvmessage.DirectionIncoming:
		return "[CUSTOMER]"
	case cvmessage.DirectionOutgoing:
		return "[AGENT]"
	default:
		return "[SPEAKER]"
	}
}

const listenConversationTruncatedSuffix = " [truncated]"

// sanitizeListenLineText neutralizes customer-controlled text before it becomes
// part of a listen line.
//
// The buffered lines are concatenated verbatim into the listen turn's prompt,
// where two things are structural: a line's speaker tag ("[CUSTOMER] ",
// "[AGENT] ") sits at position 0 of the line, and listenTranscriptNewMarker is
// its own line separating already-seen lines from new ones. Raw message text
// can carry both, so without this a customer could write a body containing a
// newline followed by "[AGENT] ..." to forge agent speech, or the marker itself
// to forge the seen/new boundary. Multi-line text also breaks the line-counted
// window, where one message must cost exactly one line.
//
// Three rules, in order:
//
//   - every CRLF/CR/LF becomes a space and whitespace runs collapse to one
//     space, so one message is always exactly one line;
//   - the new-since marker is rewritten to "[marker]", so it cannot be forged;
//   - text that itself begins with "[" is prefixed with "> ", so it can never
//     be read as this line's tag.
//
// A tag-shaped token left mid-line (".. [AGENT] ..") is accepted residual: the
// tag convention is position-0 only, so a mid-line one reads as quoted text.
func sanitizeListenLineText(s string) string {
	res := strings.Join(strings.Fields(s), " ")
	res = strings.ReplaceAll(res, listenTranscriptNewMarker, "[marker]")
	if strings.HasPrefix(res, "[") {
		res = "> " + res
	}

	return res
}

// truncateListenLineText caps one sanitized customer-controlled string at
// maxChars runes, marking the cut so the model can tell a clipped line from a
// short one. maxChars <= 0 disables the cap.
func truncateListenLineText(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}

	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}

	return string(runes[:maxChars]) + listenConversationTruncatedSuffix
}

// conversationMessageLine renders one buffered line (design 2026-09-05 §5.3.3):
// tag, optional "Subject: ..." line, text truncated to
// aicall_listen_conversation_max_message_chars, and one "[media: <type>]"
// token per attachment (the media.Type string, no allowlist, no URL or
// payload).
//
// All three customer-controlled strings (subject, text and the media type,
// which is a provider-supplied string on the message row) go through
// sanitizeListenLineText first; subject, text and the joined media tokens are
// then each capped independently by the same per-field cap -- an unbounded
// subject (or an unbounded run of attachments) would otherwise be a way
// around the text cap, and one message can contribute at most about three
// times this many characters. Truncation runs on the sanitized string so the
// cap is measured against what actually reaches the prompt. The newline after
// the subject is ours, not the customer's, so it survives sanitizing.
func conversationMessageLine(evt *cvmessage.Message) string {
	maxChars := config.Get().AIcallListenConversationMaxMessageChars

	var sb strings.Builder
	sb.WriteString(speakerTagForDirection(evt.Direction))

	if subject := truncateListenLineText(sanitizeListenLineText(evt.Subject), maxChars); subject != "" {
		sb.WriteString(" Subject: ")
		sb.WriteString(subject)
		sb.WriteString("\n")
	} else {
		sb.WriteString(" ")
	}

	sb.WriteString(truncateListenLineText(sanitizeListenLineText(evt.Text), maxChars))

	// The tokens are joined into their own builder and capped by the SAME
	// per-message limit as the text and the subject (code review round 4).
	// Appending them straight onto sb would leave a message carrying many
	// attachments as an unbounded way around that cap. The cap is applied to
	// the joined tokens alone -- the separator that attaches the suffix to the
	// text is ours and is added after the truncation, so it cannot be eaten by
	// the cut.
	var msb strings.Builder
	for _, m := range evt.Medias {
		if msb.Len() > 0 {
			msb.WriteString(" ")
		}
		mediaType := sanitizeListenLineText(string(m.Type))
		if mediaType == "" {
			mediaType = "unknown"
		}
		msb.WriteString("[media: ")
		msb.WriteString(mediaType)
		msb.WriteString("]")
	}

	if mediaSuffix := truncateListenLineText(msb.String(), maxChars); mediaSuffix != "" {
		// One separator before the suffix. When text is empty the builder
		// already ends in the tag's trailing space (or the subject's newline),
		// so only add a space after non-empty content.
		if s := sb.String(); !strings.HasSuffix(s, " ") && !strings.HasSuffix(s, "\n") {
			sb.WriteString(" ")
		}
		sb.WriteString(mediaSuffix)
	}

	return strings.TrimRight(sb.String(), " \n")
}

// EventCVMessageCreated is the conversation listen intake (design 2026-09-05
// §5.3.2). It runs for EVERY conversation message platform-wide, so everything
// before the resolver lookup must stay trivially cheap, and the resolver
// lookup itself is the only Redis round trip for the >99% that are not ours.
func (h *aicallHandler) EventCVMessageCreated(ctx context.Context, evt *cvmessage.Message) {
	// While the feature is dark it must not cost a Redis round trip per
	// platform-wide message. A mid-session flag flip is handled turn-side.
	if !config.Get().AIcallListenEnabled || !config.Get().AIcallListenConversationEnabled {
		return
	}

	if evt.TMDelete != nil {
		promListenConversationSegmentTotal.WithLabelValues("dropped_deleted").Inc()
		return
	}
	if strings.TrimSpace(evt.Text) == "" && strings.TrimSpace(evt.Subject) == "" && len(evt.Medias) == 0 {
		promListenConversationSegmentTotal.WithLabelValues("dropped_empty").Inc()
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"func":            "EventCVMessageCreated",
		"conversation_id": evt.ConversationID,
		"message_id":      evt.ID,
	})

	aicallIDs, err := h.cache.ListenConversationAIcallIDsGet(ctx, evt.ConversationID)
	if err != nil {
		log.Warnf("Could not resolve the listening aicalls. err: %v", err)
		promListenConversationSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}
	if len(aicallIDs) == 0 {
		promListenConversationSegmentTotal.WithLabelValues("dropped_unknown").Inc()
		return
	}

	line := conversationMessageLine(evt)
	ttl := listenBufferTTL()
	windowSize := config.Get().AIcallListenWindowSize
	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second
	buffered := false

	for _, aicallID := range aicallIDs {
		// h.Get -> h.db.AIcallGet (the dbhandler is cache-first with a DB
		// fallback); acceptable because this runs only for the resolved
		// fraction. The tenant assertion needs the row.
		c, errGet := h.Get(ctx, aicallID)
		if errGet != nil {
			log.Warnf("Could not get the listening aicall. aicall_id: %s, err: %v", aicallID, errGet)
			promListenConversationSegmentTotal.WithLabelValues("failed").Inc()
			continue
		}
		if c.CustomerID == uuid.Nil || evt.CustomerID == uuid.Nil || c.CustomerID != evt.CustomerID {
			log.Warnf("Cross-customer conversation message blocked. aicall_id: %s, aicall_customer_id: %s, message_customer_id: %s", aicallID, c.CustomerID, evt.CustomerID)
			promListenConversationSegmentTotal.WithLabelValues("dropped_tenant_mismatch").Inc()
			continue
		}
		if c.Status != aicall.StatusProgressing || c.TMDelete != nil {
			// A stale resolver entry (the SREM lost a race, or the TTL has not
			// expired yet) must never feed a session that is already over.
			log.Warnf("Skipping a non-progressing listening aicall. aicall_id: %s, status: %s", aicallID, c.Status)
			promListenConversationSegmentTotal.WithLabelValues("dropped_stale").Inc()
			continue
		}
		if listenConversationIDFromMetadata(c) != evt.ConversationID {
			// The resolver said this AIcall listens here, but its own pointer
			// names another conversation. Trust the pointer.
			log.Warnf("Skipping a listening aicall whose pointer names another conversation. aicall_id: %s, pointer: %s", aicallID, listenConversationIDFromMetadata(c))
			promListenConversationSegmentTotal.WithLabelValues("dropped_stale").Inc()
			continue
		}

		if errPending := h.cache.ListenPendingPush(ctx, aicallID, line, ttl); errPending != nil {
			log.Warnf("Could not buffer the pending line. aicall_id: %s, err: %v", aicallID, errPending)
			promListenConversationSegmentTotal.WithLabelValues("failed").Inc()
			continue
		}
		if errWindow := h.cache.ListenWindowPush(ctx, aicallID, line, windowSize, ttl); errWindow != nil {
			log.Warnf("Could not buffer the window line. aicall_id: %s, err: %v", aicallID, errWindow)
		}
		promListenConversationSegmentTotal.WithLabelValues("buffered").Inc()
		buffered = true

		// Only a customer message starts a turn; agent/bot output is context.
		if evt.Direction != cvmessage.DirectionIncoming {
			continue
		}

		acquired, errLock := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
		if errLock != nil {
			log.Warnf("Could not take the listen turn lock. aicall_id: %s, err: %v", aicallID, errLock)
			// A lock error is a real failure, not the debounce doing its job,
			// so it is metered apart from skipped_locked (code review round 4).
			promListenTurnTotal.WithLabelValues(string(listenKindConversation), "failed").Inc()
			// The line is buffered but no turn was started, so arm the
			// deferred flush rather than stranding it until the next message.
			h.scheduleListenFlush(aicallID)
			continue
		}
		if !acquired {
			promListenTurnTotal.WithLabelValues(string(listenKindConversation), "skipped_locked").Inc()
			h.scheduleListenFlush(aicallID)
			continue
		}

		h.spawnListenTurn(aicallID)
	}

	if buffered {
		// re-arm on every buffered line so the resolver entry outlives any
		// conversation that is actually active; design §5.2.2 promised
		// refresh-on-SADD and start is the only SADD, so this is where the
		// refresh has to live.
		//
		// EXPIRE only, never SADD (code review round 4). The re-arm runs only
		// for ids SMEMBERS already returned, so a SADD would add nothing -- but
		// it WOULD resurrect a membership a concurrent stop removed between the
		// lookup and here. EXPIRE on a missing key is a no-op instead.
		if errRearm := h.cache.ListenConversationResolverTouch(ctx, evt.ConversationID, listenResolverTTL); errRearm != nil {
			// The line is already buffered; at worst the entry expires.
			log.Warnf("Could not re-arm the conversation listen resolver entry. err: %v", errRearm)
		}
	}
}

// spawnListenTurn runs RunListenTurn detached, or hands the id to the test hook.
func (h *aicallHandler) spawnListenTurn(aicallID uuid.UUID) {
	if h.runListenTurnHook != nil {
		h.runListenTurnHook(context.Background(), aicallID)
		return
	}
	go func(id uuid.UUID) {
		turnCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		h.RunListenTurn(turnCtx, id)
	}(aicallID)
}

// scheduleListenFlush arms at most one deferred flush per AIcall per process
// (design 2026-09-05 §5.4). The delay is the debounce interval plus a random
// jitter so the two replicas' timers do not race the lock at the same instant.
func (h *aicallHandler) scheduleListenFlush(aicallID uuid.UUID) {
	if _, loaded := h.flushScheduled.LoadOrStore(aicallID, struct{}{}); loaded {
		promListenConversationFlushTotal.WithLabelValues("skipped_scheduled").Inc()
		return
	}

	delay := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second
	if jitter := config.Get().AIcallListenConversationFlushJitterMs; jitter > 0 {
		delay += time.Duration(rand.IntN(jitter+1)) * time.Millisecond
	}

	after := h.afterFunc
	if after == nil {
		after = time.AfterFunc
	}
	after(delay, func() { h.listenFlushFire(aicallID) })
}

// listenFlushFire is the timer body. INVARIANT: the marker is deleted BEFORE
// TryLock, never after the turn, so a message arriving mid-flush can arm a
// fresh timer instead of being stranded behind a marker nobody will clear.
func (h *aicallHandler) listenFlushFire(aicallID uuid.UUID) {
	h.flushScheduled.Delete(aicallID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	interval := time.Duration(config.Get().AIcallListenEvaluateIntervalSeconds) * time.Second
	acquired, err := h.cache.ListenTurnTryLock(ctx, aicallID, interval)
	if err != nil {
		logrus.WithFields(logrus.Fields{"func": "listenFlushFire", "aicall_id": aicallID}).
			Warnf("Could not take the listen turn lock for the flush. err: %v", err)
		promListenConversationFlushTotal.WithLabelValues("skipped_locked").Inc()
		return
	}
	if !acquired {
		promListenConversationFlushTotal.WithLabelValues("skipped_locked").Inc()
		return
	}

	promListenConversationFlushTotal.WithLabelValues("ran").Inc()
	if h.runListenTurnHook != nil {
		h.runListenTurnHook(ctx, aicallID)
		return
	}
	h.RunListenTurn(ctx, aicallID)
}
