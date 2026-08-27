package casenote

import (
	"github.com/gofrs/uuid"
)

// Event types published by contact-manager for CaseNote state changes.
//
// CaseNote is transport-isolated from customer-facing data (see the CaseNote doc comment), so
// both events are published through the plain notifyHandler.PublishEvent() primitive -- never
// PublishWebhookEvent(). The string values are the wire contract and must not change: they were
// published as bare literals before VOIP-1405 named them.
const (
	// EventTypeCaseNoteCreated is published when an agent-facing note is added to a Case.
	EventTypeCaseNoteCreated string = "case_note_created"

	// EventTypeCaseNoteDeleted is published when a note is soft-deleted from a Case.
	EventTypeCaseNoteDeleted string = "case_note_deleted"
)

// CaseNoteDeletedEvent is the payload of case_note_deleted.
//
// It replaces the `map[string]uuid.UUID{"id":..., "case_id":..., "customer_id":...}` literal
// published by casehandler.CaseNoteDelete, with an identical JSON key SET (field order differs
// only).
//
// SILENT-FAILURE WARNING (design §2.3): unlike the other new event structs of this ticket, this
// payload DOES carry a top-level `id` -- the note id. Without the EventSubscriptionID override
// below, the default resolution would happily address these events by that note id, producing
// well-formed routing keys that no case binding ever matches and that no placeholder metric can
// flag. The override, and the golden-table row asserting the address is the case id, are both
// mandatory.
type CaseNoteDeletedEvent struct {
	ID         uuid.UUID `json:"id"`
	CaseID     uuid.UUID `json:"case_id"`
	CustomerID uuid.UUID `json:"customer_id"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2): the parent CaseID, NOT the note's own ID. The note id
// first becomes knowable in the create event, so nobody can bind to it in advance, while every
// consumer follows the case. It pairs case_note_deleted with case_note_created (whose CaseNote
// payload overrides the same way) in one key space.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *CaseNoteDeletedEvent) EventSubscriptionID() string {
	return h.CaseID.String()
}
