package casehandler

//go:generate mockgen -package casehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// CaseHandler interface for Case business logic operations (design §4's
// get-or-create algorithm and §5's lifecycle).
type CaseHandler interface {
	// GetOrCreate implements design doc §4 exactly: reuse an open Case for
	// (customerID, peerType, peerTarget, referenceType) if one exists and
	// has not timed out; otherwise close-and-reopen (timeout) or insert a
	// fresh Case (first contact / no prior case), honoring an explicit
	// case_id hint (§4.3) when present and valid. caseIDHint may be nil.
	//
	// self is the local (business-side) endpoint address for this event,
	// used only for design §4.4's proactive-link write: when a brand-new
	// Case opens for a non-"conversation_message" referenceType (today:
	// "call"), GetOrCreate looks up the sibling message Conversation for
	// (self, peer) and, if found, stamps Metadata.ContactCaseID on it.
	// Pass a zero commonaddress.Address to opt out of this optimization
	// entirely (e.g. reference_type == "conversation_message" itself,
	// which never triggers this write per §4.4's scope).
	GetOrCreate(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)

	// Close implements design §5.1: idempotent, race-tolerant close.
	// Returns the ACTUALLY persisted closed_reason/closed_by, never the
	// caller's own intent, distinguishing a genuine close from a
	// race-lost double-close via CloseResult.AlreadyClosed.
	Close(ctx context.Context, customerID, id uuid.UUID, closedByType commonidentity.OwnerType, closedByID uuid.UUID) (*CloseResult, error)

	// Continue implements design §5.3: agent-initiated manual
	// continuation for accidental-close recovery. Requires the source
	// case closed; requires the caller be the owning agent or an
	// admin/manager (callerIsAdmin, decided upstream). Reuses the same
	// insertWithRetry primitive as GetOrCreate's insert branches.
	Continue(ctx context.Context, customerID, id uuid.UUID, callerType commonidentity.OwnerType, callerID uuid.UUID, callerIsAdmin bool) (*kase.Case, error)

	// ResolutionCreateCaseLevel / ResolutionDeleteCaseLevel implement
	// design §3.4's contact-attribution write paths: create/soft-delete a
	// case-level Resolution and, in the SAME transaction, re-derive and
	// write Case.contact_id via deriveCaseContactID.
	ResolutionCreateCaseLevel(ctx context.Context, customerID, caseID, contactID uuid.UUID, resolutionType, resolvedByType string, resolvedByID uuid.UUID) (*resolution.Resolution, error)
	ResolutionDeleteCaseLevel(ctx context.Context, customerID, caseID, id uuid.UUID) error

	// CaseListUnresolved is design §6's agent-facing unresolved queue.
	CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error)

	// CaseNoteCreate / CaseNoteDelete / CaseNoteListByCase implement
	// design §3.5: internal, agent-facing case annotations, physically
	// and transport-isolated from customer-facing data. Both create and
	// delete publish their event via the plain PublishEvent() primitive
	// -- NEVER PublishWebhookEvent().
	CaseNoteCreate(ctx context.Context, customerID, caseID uuid.UUID, authorType string, authorID *uuid.UUID, text string) (*casenote.CaseNote, error)
	CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error
	CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error)
}

type caseHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewCaseHandler returns CaseHandler interface
func NewCaseHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) CaseHandler {
	return &caseHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}
