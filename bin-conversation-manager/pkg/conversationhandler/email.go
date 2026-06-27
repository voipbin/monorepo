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
// Outbound-only: the email-manager email_created event fires before the
// provider send is attempted, so the conversation message is recorded at
// status=progressing and is never updated afterward (the conversation message
// is an immutable send-time fact log, mirroring the SMS absorption path; the
// service does not subscribe to email_updated/email_deleted).
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
