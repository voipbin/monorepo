package kase

import (
	"github.com/gofrs/uuid"
)

// Event types published by contact-manager for Case-scoped state changes.
//
// All of them are internal, agent-facing audit events published through the plain
// notifyHandler.PublishEvent() primitive -- never PublishWebhookEvent() (see
// pkg/casehandler/casenote.go's precedent). The string values are the wire contract and must not
// change: they were published as bare literals before VOIP-1405 named them.
const (
	// EventTypeCaseTagAdded is published when a tag is assigned to a Case (design VOIP-1254).
	EventTypeCaseTagAdded string = "case_tag_added"

	// EventTypeCaseTagRemoved is published when a tag is unassigned from a Case (design VOIP-1254).
	EventTypeCaseTagRemoved string = "case_tag_removed"

	// EventTypeCaseContactAttributed is published when a Case's Contact attribution is set
	// (design VOIP-1253).
	EventTypeCaseContactAttributed string = "case_contact_attributed"

	// EventTypeCaseContactDetached is published when a Case's Contact attribution is cleared
	// (design VOIP-1253).
	EventTypeCaseContactDetached string = "case_contact_detached"
)

// CaseTagEvent is the payload of case_tag_added / case_tag_removed.
//
// It replaces the `map[string]uuid.UUID{"case_id":..., "tag_id":...}` literal these two sites
// published before VOIP-1405. The JSON key SET is identical (field order differs -- Go marshals
// maps key-sorted and structs in declaration order -- which is JSON-semantically irrelevant), so
// fanout consumers observe no change. The reason for the struct is the global topic exchange: a
// map cannot satisfy the pointer-receiver eventtopic.SubscriptionIdentifier assertion, and this
// payload has no top-level `id` for the default resolution to fall back on, so without the struct
// every one of these events would publish under the `-` placeholder address (design §2.2).
type CaseTagEvent struct {
	CaseID uuid.UUID `json:"case_id"`
	TagID  uuid.UUID `json:"tag_id"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2). The tag event has no own id at all; the case is the axis
// consumers follow, so every case-scoped event of contact-manager converges on the case id and a
// single `contact-manager.case.<case-id>.#` binding covers the whole case lifecycle
// (design §4 address-convergence note).
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *CaseTagEvent) EventSubscriptionID() string {
	return h.CaseID.String()
}

// CaseContactEvent is the payload of case_contact_attributed / case_contact_detached.
//
// It replaces the `map[string]uuid.UUID{"case_id":..., "contact_id":...}` literal published by
// casehandler.UpdateContact, which emits BOTH event types from the same site. ContactID is
// uuid.Nil on detach -- consumers read the event type to disambiguate, exactly as they did with
// the map. Same JSON key set, same placeholder-avoidance rationale as CaseTagEvent.
type CaseContactEvent struct {
	CaseID    uuid.UUID `json:"case_id"`
	ContactID uuid.UUID `json:"contact_id"`
}

// EventSubscriptionID returns the case id as the subscription address -- see CaseTagEvent's
// method comment for the rationale and the pointer-receiver requirement.
func (h *CaseContactEvent) EventSubscriptionID() string {
	return h.CaseID.String()
}
