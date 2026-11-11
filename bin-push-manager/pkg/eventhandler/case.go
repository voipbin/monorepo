package eventhandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

// EventCaseNoteCreated handles the contact-manager's case_note_created event. Only a note that an agent wrote is
// notified, to the agent who owns the case, unless he wrote it himself. A note of any other author type, and a
// note of an agent without an author id, is not notified.
func (h *eventHandler) EventCaseNoteCreated(ctx context.Context, n *cmcasenote.CaseNote) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCaseNoteCreated",
		"customer_id": n.CustomerID,
		"case_id":     n.CaseID,
		"note_id":     n.ID,
	})

	if n.AuthorType != cmcasenote.AuthorTypeAgent || n.AuthorID == nil || *n.AuthorID == uuid.Nil {
		return nil
	}
	if n.CustomerID == uuid.Nil || n.CaseID == uuid.Nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, h.eventTimeout)
	defer cancel()

	c, err := h.reqHandler.ContactV1CaseGet(ctx, n.CustomerID, n.CaseID)
	if err != nil {
		log.Errorf("Could not get the case. err: %v", err)
		return fmt.Errorf("could not get the case. err: %w", err)
	}

	if c == nil || c.CustomerID != n.CustomerID {
		log.Warn("The case does not belong to the customer of the note. Skipping.")
		return nil
	}
	if c.OwnerType != commonidentity.OwnerTypeAgent || c.OwnerID == uuid.Nil || c.OwnerID == *n.AuthorID {
		return nil
	}

	targets, err := h.settingsHandler.FilterRecipients(ctx, n.CustomerID, []uuid.UUID{c.OwnerID}, notification.KindCaseNote, uuid.Nil)
	if err != nil {
		log.Errorf("Could not apply the settings. err: %v", err)
		return fmt.Errorf("could not apply the settings. err: %w", err)
	}

	h.sendAll(ctx, n.CustomerID, targets, notification.KindCaseNote, map[string]string{
		"case_id": n.CaseID.String(),
		"note_id": n.ID.String(),
	})

	return nil
}

// EventCaseAssigned handles the contact-manager's case_assigned event. The new owner is told, also when he
// assigned the case to himself: the event does not say who assigned it.
func (h *eventHandler) EventCaseAssigned(ctx context.Context, e *cmkase.CaseAssignedEvent) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCaseAssigned",
		"customer_id": e.CustomerID,
		"case_id":     e.CaseID,
	})

	if e.CustomerID == uuid.Nil || e.CaseID == uuid.Nil || e.OwnerID == uuid.Nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, h.eventTimeout)
	defer cancel()

	targets, err := h.settingsHandler.FilterRecipients(ctx, e.CustomerID, []uuid.UUID{e.OwnerID}, notification.KindCaseAssigned, uuid.Nil)
	if err != nil {
		log.Errorf("Could not apply the settings. err: %v", err)
		return fmt.Errorf("could not apply the settings. err: %w", err)
	}

	h.sendAll(ctx, e.CustomerID, targets, notification.KindCaseAssigned, map[string]string{
		"case_id": e.CaseID.String(),
	})

	return nil
}
