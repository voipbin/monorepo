package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"time"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// insightDefaultListLimit / insightMaxListLimit bound the "limit" argument
// accepted by both Insight tools. Kept smaller than resourceListPageSize
// (get_resource's enrichment page size) because these are Insight-facing
// summary lists, not a full resource dump.
const (
	insightDefaultListLimit = 20
	insightMaxListLimit     = 50
)

// resolveInsightListLimit clamps an LLM-supplied limit into
// [1, insightMaxListLimit], defaulting to insightDefaultListLimit when unset
// or non-positive.
func resolveInsightListLimit(v int) uint64 {
	if v <= 0 {
		return insightDefaultListLimit
	}
	if v > insightMaxListLimit {
		return insightMaxListLimit
	}
	return uint64(v)
}

// toolHandleGetContactInteractions lists past interactions (calls,
// conversation messages) tied to the Case's peer/contact. Scope is always
// the current Insight AIcall's own Case (c.ReferenceID) -- there is no
// case_id/contact_id argument (design VOIP-1234 §3: implicit scoping
// removes the IDOR-shaped bug class entirely rather than defending against
// it).
func (h *aicallHandler) toolHandleGetContactInteractions(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetContactInteractions",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_contact_interactions.")

	res := newToolResult(tc.ID)

	var args struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	limit := resolveInsightListLimit(args.Limit)

	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		fillFailed(res, fmt.Errorf("get_contact_interactions is only supported for contact_case reference type"))
		return res
	}

	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		if stderrors.Is(err, requesthandler.ErrNotFound) {
			fillSuccess(res, "interaction_list", c.ReferenceID.String(), msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the case. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		fillSuccess(res, "interaction_list", c.ReferenceID.String(), msgResourceNotFound)
		return res
	}

	var interactions []*cminteraction.Interaction
	if kase.ContactID != nil {
		interactions, _, err = h.reqHandler.ContactV1InteractionList(
			ctx, c.CustomerID, limit, "", "", "", *kase.ContactID, uuid.Nil, time.Time{})
	} else {
		interactions, _, err = h.reqHandler.ContactV1InteractionList(
			ctx, c.CustomerID, limit, "", string(kase.PeerType), kase.PeerTarget, uuid.Nil, uuid.Nil, time.Time{})
	}
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	if len(interactions) == 0 {
		fillSuccess(res, "interaction_list", c.ReferenceID.String(), "no interactions found")
		return res
	}

	lines := make([]string, 0, len(interactions))
	for _, it := range interactions {
		ts := "unknown"
		if it.TMInteraction != nil {
			ts = it.TMInteraction.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf(
			"[%s] direction=%s peer=%s/%s reference_type=%s reference_id=%s",
			ts, it.Direction, it.PeerType, it.PeerTarget, it.ReferenceType, it.ReferenceID,
		))
	}

	body := renderBodyLines("", lines, uint64(len(interactions)) >= limit, "interactions")
	fillSuccess(res, "interaction_list", c.ReferenceID.String(), body)
	return res
}

// toolHandleGetConversationContent retrieves the message transcript of a
// conversation, given the reference_id of a conversation_message-type
// interaction the LLM discovered via get_contact_interactions (design
// VOIP-1234 §5: explicit target selection, not an implicit
// server-picks-the-most-recent-thread auto-resolution).
//
// Resolution is a FIXED 2 RPC calls regardless of message/thread count:
//  1. ConversationV1MessageGet(reference_id) -- resolves the message AND is
//     the ownership/IDOR check (reference_id is now LLM-suppliable, unlike
//     the implicit Case scoping used elsewhere in this file).
//  2. ConversationV1MessageList(filters={conversation_id}) -- one list call
//     for the whole surrounding thread, capped at limit.
func (h *aicallHandler) toolHandleGetConversationContent(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetConversationContent",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_conversation_content.")

	res := newToolResult(tc.ID)

	var args struct {
		ReferenceID string `json:"reference_id"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}
	if args.ReferenceID == "" {
		fillFailed(res, fmt.Errorf("reference_id is required, call get_contact_interactions first to discover candidate ids"))
		return res
	}
	refID, err := uuid.FromString(args.ReferenceID)
	if err != nil || refID == uuid.Nil {
		fillFailed(res, fmt.Errorf("invalid reference_id"))
		return res
	}
	limit := resolveInsightListLimit(args.Limit)

	// RPC 1/2: resolve the message + ownership check (single masking site,
	// mirrors resolveResource's IDOR-safe contract in tool_resource.go).
	msg, err := h.reqHandler.ConversationV1MessageGet(ctx, refID)
	if err != nil {
		if stderrors.Is(err, requesthandler.ErrNotFound) {
			fillSuccess(res, "conversation_content", args.ReferenceID, msgResourceNotFound)
			return res
		}
		log.Errorf("Could not get the message. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}
	if msg == nil || msg.CustomerID != c.CustomerID || msg.CustomerID == uuid.Nil {
		log.Warnf("Cross-customer message access blocked. reference_id: %s", refID)
		fillSuccess(res, "conversation_content", args.ReferenceID, msgResourceNotFound)
		return res
	}

	// RPC 2/2: one list call filtered by conversation_id -- NOT a per-message
	// fetch loop. This is the fixed-cost path decided in design VOIP-1234 §5
	// after the original N+1 draft was rejected as wasteful.
	filters := map[cvmessage.Field]any{
		cvmessage.FieldConversationID: msg.ConversationID.String(),
	}
	msgs, err := h.reqHandler.ConversationV1MessageList(ctx, "", limit, filters)
	if err != nil {
		log.Errorf("Could not list conversation messages. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	if len(msgs) == 0 {
		fillSuccess(res, "conversation_content", args.ReferenceID, "no messages found")
		return res
	}

	lines := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ts := "unknown"
		if m.TMCreate != nil {
			ts = m.TMCreate.UTC().Format(time.RFC3339)
		}
		lines = append(lines, fmt.Sprintf("[%s %s] %s", ts, m.Direction, m.Text))
	}

	body := renderBodyLines("", lines, uint64(len(msgs)) >= limit, "messages")
	fillSuccess(res, "conversation_content", args.ReferenceID, body)
	return res
}
