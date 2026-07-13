package kase

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Case is a thin, per-channel session header that groups related
// Interactions into a start/end unit agents can pick up, work, and close --
// without touching the existing Interaction projection pipeline.
//
// Package named "kase" (not "case", a Go reserved word) following this
// monorepo's convention for keyword collisions.
//
// See docs/plans/2026-07-07-contact-case-management-design.md §3.1.
type Case struct {
	ID         uuid.UUID `json:"id"          db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	// PeerType/PeerTarget identify the remote party this Case is scoped
	// to. PeerTarget is normalized via commonaddress.NormalizeTarget --
	// bit-identical to contact_addresses.target and interaction.peer_target.
	PeerType   commonaddress.Type `json:"peer_type"   db:"peer_type"`
	PeerTarget string             `json:"peer_target" db:"peer_target"`

	// ReferenceType reuses contact_interactions.reference_type's EXISTING
	// stored vocabulary ("call", "conversation_message", ...) -- NOT
	// conversation-manager's message.ReferenceType (a different, unrelated
	// enum). Case.ReferenceType must match Interaction.ReferenceType
	// exactly for the design §4 get-or-create join to work.
	ReferenceType string `json:"reference_type" db:"reference_type"`

	// Name/Detail are optional, freeform case metadata settable only at
	// creation time via Create (design VOIP-1243 §3.4). Empty string is
	// persisted as the column's default/empty value, not NULL.
	Name   string `json:"name,omitempty"   db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	// ContactID is a nullable denormalized cache; single source of truth
	// is this column itself, every write goes through
	// casehandler.UpdateContact (design VOIP-1253).
	ContactID *uuid.UUID `json:"contact_id" db:"contact_id,uuid"`

	// Owner (OwnerType + OwnerID) is reused as-is from the conversation
	// assignment precedent. NEVER cleared by closing a Case (design §7)
	// -- this is a load-bearing invariant for /continue's authorization
	// (design §5.3).
	commonidentity.Owner

	Status       Status     `json:"status"        db:"status"`
	OpenedAt     *time.Time `json:"opened_at"      db:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at"      db:"closed_at"`
	ClosedReason string     `json:"closed_reason"  db:"closed_reason"`
	ClosedByType string     `json:"closed_by_type" db:"closed_by_type"`
	ClosedByID   *uuid.UUID `json:"closed_by_id"   db:"closed_by_id,uuid"`

	// PreviousCaseID chains re-contact: nil for the first Case with a
	// given peer, set to the prior (now-closed) Case's ID on re-contact.
	PreviousCaseID *uuid.UUID `json:"previous_case_id" db:"previous_case_id,uuid"`

	// TagIDs mirrors bin-queue-manager's Queue.TagIDs storage exactly
	// (VOIP-1254): a plain JSON column, no junction table, no reverse
	// lookup. Case tag usage is low-frequency and agent-driven (not a
	// routing hot path), so this is strictly lighter than Queue's own
	// use of the same pattern for real-time agent-tag matching.
	TagIDs []uuid.UUID `json:"tag_ids,omitempty" db:"tag_ids,json"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
}

// Status defines the Case lifecycle status.
type Status string

// Status constants.
const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
)

// ClosedReason constants. ClosedReasonMerged is reserved (schema-only,
// unused until the same-channel case-merge feature is designed -- see
// design doc §2 parked table).
const (
	ClosedReasonAgentClosed = "agent_closed"
	ClosedReasonTimeout     = "timeout"
	ClosedReasonMerged      = "merged"
)

// ClosedByType constants.
const (
	ClosedByTypeAgent  = "agent"
	ClosedByTypeSystem = "system"
)
