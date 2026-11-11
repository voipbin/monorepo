package casehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	cerrors "monorepo/bin-common-handler/models/errors"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// maxCaseChainHops bounds the PreviousCaseID walk used to find the
// conversation of a continued Case.
const maxCaseChainHops = 10

// caseConversationID returns the id of the Conversation a Case belongs to.
// The Case itself is used when ReferenceType is conversation_message and
// ReferenceID is a non-nil UUID. A continued Case carries an empty
// ReferenceID (see Continue), so the PreviousCaseID chain is followed
// (customer-checked, at most maxCaseChainHops). It returns uuid.Nil when
// there is no conversation (call-origin Case, no reference, a chain that
// ends at a missing or other-customer Case or exceeds the hop limit). A
// database error while walking the chain is returned: skipping silently
// would leave the AI answering a conversation the agent took over.
func (h *caseHandler) caseConversationID(ctx context.Context, c *kase.Case) (uuid.UUID, error) {
	cur := c
	for hop := 0; cur != nil && hop <= maxCaseChainHops; hop++ {
		if cur.ReferenceType == kase.ReferenceTypeConversationMessage {
			if id := uuid.FromStringOrNil(cur.ReferenceID); id != uuid.Nil {
				return id, nil
			}
		}
		if cur.PreviousCaseID == nil {
			return uuid.Nil, nil
		}

		prev, err := h.db.CaseGetByID(ctx, *cur.PreviousCaseID)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return uuid.Nil, nil
			}
			return uuid.Nil, fmt.Errorf("could not get the previous case. err: %w", err)
		}
		if prev.CustomerID != c.CustomerID {
			return uuid.Nil, nil
		}
		cur = prev
	}
	return uuid.Nil, nil
}

// syncConversationOwner sets the owner of the Conversation linked to the
// Case to targetID (uuid.Nil releases it). conversation-manager derives
// owner_type from owner_id and validates the agent and the customer.
//
// onlyIfOwnerID, when not Nil, restricts the change to a Conversation
// whose current owner equals it (Unassign must not release a Conversation
// that was separately given to another agent). A Case without a
// Conversation, a vanished Conversation and an unchanged owner are no-ops.
func (h *caseHandler) syncConversationOwner(ctx context.Context, c *kase.Case, targetID, onlyIfOwnerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "syncConversationOwner",
		"case_id": c.ID,
	})

	cvID, err := h.caseConversationID(ctx, c)
	if err != nil {
		return fmt.Errorf("could not resolve the linked conversation. err: %w", err)
	}
	if cvID == uuid.Nil {
		return nil
	}
	log = log.WithField("conversation_id", cvID)

	cv, err := h.reqHandler.ConversationV1ConversationGet(ctx, cvID)
	if err != nil {
		var ve *cerrors.VoipbinError
		if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
			log.Info("The linked conversation does not exist. Skipping the owner sync.")
			return nil
		}
		return fmt.Errorf("could not get the linked conversation. err: %w", err)
	}

	if cv.CustomerID != c.CustomerID {
		log.Warn("The linked conversation belongs to another customer. Skipping the owner sync.")
		return nil
	}
	if cv.OwnerID == targetID {
		return nil
	}
	if onlyIfOwnerID != uuid.Nil && cv.OwnerID != onlyIfOwnerID {
		log.Info("The linked conversation is owned by someone else. Skipping the owner release.")
		return nil
	}

	if _, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, cvID, map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: targetID,
	}); err != nil {
		return fmt.Errorf("could not update the linked conversation owner. err: %w", err)
	}
	log.WithField("owner_id", targetID).Info("Updated the linked conversation owner.")

	return nil
}
