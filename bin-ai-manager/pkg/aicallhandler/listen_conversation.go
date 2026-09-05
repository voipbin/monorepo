package aicallhandler

import (
	"context"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	kmkase "monorepo/bin-contact-manager/models/kase"

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
		logrus.WithFields(logrus.Fields{"func": "rollbackListenConversation", "aicall_id": aicallID}).
			Warnf("Could not roll back the conversation listen resolver membership. err: %v", errRem)
	}
}
