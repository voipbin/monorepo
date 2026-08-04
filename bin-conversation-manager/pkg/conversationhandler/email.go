package conversationhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	emmemail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

// EmailEventSent records a sent email as an outgoing conversation message.
//
// The email-manager email_created event fires before the provider send is
// attempted, so the conversation message is recorded at status=progressing.
// That status is later reconciled by EmailEventUpdated, which follows
// email-manager's own email_updated event rather than guessing an outcome
// here -- see the package doc on EmailEventUpdated.
//
// One email may have multiple destinations; each destination yields its own
// (self, peer) conversation and its own outgoing message. Idempotency is keyed
// on a composite transaction_id of email.ID + ":" + normalized(peer), unique
// per (email, recipient), so a re-delivered event is a no-op.
func (h *conversationHandler) EmailEventSent(ctx context.Context, e *emmemail.Email) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EmailEventSent",
		"email_id": e.ID,
	})

	// Source is a pointer and may be nil on a deserialized event. Skip without
	// side effects rather than dereferencing a nil address.
	if e.Source == nil {
		log.Debugf("Email has no source address. Skipping.")
		return nil
	}
	self := *e.Source

	// Process each destination independently. A failure on one recipient must
	// not drop the remaining recipients of the same email: the subscribe
	// consumer is at-most-once (the message is acked before processing), so an
	// early return here would permanently lose the un-attempted destinations.
	// Per-destination errors are logged and accumulated; the first error is
	// returned at the end for observability only.
	var firstErr error
	for _, destination := range e.Destinations {
		peer := destination

		// Composite dedup key: unique per (email, recipient).
		normalizedPeer, _ := commonaddress.NormalizeTarget(peer.Type, peer.Target)
		txID := e.ID.String() + ":" + normalizedPeer

		// Idempotency: skip if a message already exists for this (email, peer).
		existing, errGet := h.db.MessageGetsByTransactionID(ctx, txID, "", 1)
		if errGet != nil {
			log.Errorf("Could not check existing message. transaction_id: %s, err: %v", txID, errGet)
			if firstErr == nil {
				firstErr = errors.Wrapf(errGet, "could not check existing message. transaction_id: %s", txID)
			}
			continue
		}
		if len(existing) > 0 {
			log.Debugf("Message already exists for this email destination. Skipping. transaction_id: %s", txID)
			continue
		}

		// Locate or create the conversation thread for this (self, peer) pair.
		cv, errConv := h.GetOrCreateBySelfAndPeer(
			ctx,
			e.CustomerID,
			conversation.TypeEmail,
			"", // email has no external dialog id; same as SMS
			self,
			peer,
		)
		if errConv != nil {
			log.Errorf("Could not get conversation. email_id: %s, err: %v", e.ID, errConv)
			if firstErr == nil {
				firstErr = errors.Wrapf(errConv, "could not get conversation. email_id: %s", e.ID)
			}
			continue
		}

		// Create the outgoing conversation message. PK is auto-generated
		// (id=uuid.Nil); reference_id points at the source email; status is
		// progressing because the email has not yet been sent at email_created.
		source, dst := messagehandler.DeriveEndpoints(cv, message.DirectionOutgoing)
		convMsg, errMsg := h.messageHandler.Create(ctx, messagehandler.MessageCreateArgs{
			ID:             uuid.Nil,
			CustomerID:     e.CustomerID,
			ConversationID: cv.ID,
			Direction:      message.DirectionOutgoing,
			Status:         message.StatusProgressing,
			ReferenceType:  message.ReferenceTypeEmail,
			ReferenceID:    e.ID,
			TransactionID:  txID,
			Text:           e.Content,
			Subject:        e.Subject,
			Medias:         []media.Media{},
			Source:         source,
			Destination:    dst,
		})
		if errMsg != nil {
			log.Errorf("Could not create a message. email_id: %s, err: %v", e.ID, errMsg)
			if firstErr == nil {
				firstErr = errors.Wrapf(errMsg, "could not create a message. email_id: %s", e.ID)
			}
			continue
		}
		log.WithField("message_id", convMsg.ID).Debugf("Created an email conversation message. message_id: %s", convMsg.ID)
	}

	return firstErr
}

// EmailEventUpdated reconciles an outgoing conversation message's status
// with an email-manager status change.
//
// conversation-manager is a broker for email: it does not infer delivery
// outcomes on its own (e.g. by assuming success once the send RPC returns).
// It only reflects the status email-manager itself reports via its
// provider-webhook-driven lifecycle (initiated -> processed -> delivered/
// open/click, or -> bounce/dropped/deferred/unsubscribe/spamreport; see
// bin-email-manager/docs/domain.md "Email Status Lifecycle"). Non-terminal
// statuses (e.g. processed, deferred) are no-ops here; the message stays at
// status=progressing until a terminal outcome is reported.
func (h *conversationHandler) EmailEventUpdated(ctx context.Context, e *emmemail.Email) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EmailEventUpdated",
		"email_id": e.ID,
	})

	newStatus, ok := emailStatusToMessageStatus(e.Status)
	if !ok {
		// Non-terminal provider status (e.g. initiated, processed, deferred).
		// Nothing to reconcile yet.
		return nil
	}

	var firstErr error
	for _, destination := range e.Destinations {
		normalizedPeer, _ := commonaddress.NormalizeTarget(destination.Type, destination.Target)
		txID := e.ID.String() + ":" + normalizedPeer

		existing, errGet := h.db.MessageGetsByTransactionID(ctx, txID, "", 1)
		if errGet != nil {
			log.Errorf("Could not find the message for the email destination. transaction_id: %s, err: %v", txID, errGet)
			if firstErr == nil {
				firstErr = errors.Wrapf(errGet, "could not find the message. transaction_id: %s", txID)
			}
			continue
		}
		if len(existing) == 0 {
			// The message has not been materialized yet by EmailEventSent
			// (event ordering race), or it never was. Nothing to update.
			log.Debugf("No message found for this email destination yet. Skipping. transaction_id: %s", txID)
			continue
		}

		if existing[0].Status == message.StatusDone || existing[0].Status == message.StatusFailed {
			// Already at a terminal status. Provider webhooks are not
			// guaranteed to arrive in order (e.g. a delayed "delivered" can
			// land after an earlier "bounce" already marked this failed);
			// once terminal, further reconciliation is a no-op rather than
			// letting a stale/out-of-order webhook flip the outcome.
			continue
		}

		if _, errUpd := h.messageHandler.UpdateStatus(ctx, existing[0].ID, newStatus); errUpd != nil {
			log.Errorf("Could not update the message status. message_id: %s, err: %v", existing[0].ID, errUpd)
			if firstErr == nil {
				firstErr = errors.Wrapf(errUpd, "could not update the message status. message_id: %s", existing[0].ID)
			}
			continue
		}
	}

	return firstErr
}

// emailStatusToMessageStatus maps an email-manager provider-webhook status
// onto conversation-manager's 3-state message status. It returns ok=false
// for statuses that are not a terminal outcome, so the caller can leave the
// message untouched rather than guess.
func emailStatusToMessageStatus(status emmemail.Status) (message.Status, bool) {
	switch status {
	case emmemail.StatusDelivered, emmemail.StatusOpen, emmemail.StatusClick, emmemail.StatusUnsubscribe, emmemail.StatusSpamreport:
		return message.StatusDone, true

	case emmemail.StatusBounce, emmemail.StatusDropped:
		return message.StatusFailed, true

	default:
		// StatusNone, StatusInitiated, StatusProcessed, StatusDeffered, or an
		// unrecognized provider status: none are a terminal outcome.
		return "", false
	}
}
