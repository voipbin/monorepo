package casehandler

//go:generate mockgen -package casehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
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
	GetOrCreate(ctx context.Context, customerID uuid.UUID, self, peer commonaddress.Address, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)

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

	// UpdateContact implements design VOIP-1253: attaches or detaches a
	// Case's Contact via a direct Case.contact_id write.
	UpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*kase.Case, error)

	// Assign implements the square-talk Cases menu design §3.2: writes
	// Case.Owner (owner_type/owner_id). Tenant-checked via CaseGetByID
	// (mirroring Continue's pattern); no authorization decision inside
	// this function -- per §1.4 there is none to make, any caller who
	// reaches this function (already authenticated as an agent of the
	// tenant at the API layer) may assign to any (ownerType, ownerID).
	Assign(ctx context.Context, customerID, id uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) (*kase.Case, error)

	// CaseListUnresolved is design §6's agent-facing unresolved queue.
	CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error)

	// CaseListAll returns every Case (all tenants), for case-control's
	// `--all` reconcile-contact sweep. CLI-only usage -- never exposed
	// via a customer-facing RPC/route.
	CaseListAll(ctx context.Context) ([]*kase.Case, error)

	// CaseNoteCreate / CaseNoteDelete / CaseNoteListByCase implement
	// design §3.5: internal, agent-facing case annotations, physically
	// and transport-isolated from customer-facing data. Both create and
	// delete publish their event via the plain PublishEvent() primitive
	// -- NEVER PublishWebhookEvent().
	CaseNoteCreate(ctx context.Context, customerID, caseID uuid.UUID, authorType string, authorID *uuid.UUID, text string) (*casenote.CaseNote, error)
	CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error
	CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error)

	// CaseTagAdd / CaseTagRemove / CaseTagList implement design §7
	// round-22: case-scoped tag assignment. CaseTagAdd validates tag_id
	// existence via bin-tag-manager's TagV1TagGet before assigning.
	CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error
	CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error
	CaseTagList(ctx context.Context, customerID, caseID uuid.UUID) ([]uuid.UUID, error)

	// CaseList / CaseGet implement design §9's Phase 5 RPC/REST list and
	// get surface. CaseList is a customer-scoped, optionally
	// status/owner-filtered list (thin delegation to dbhandler.CaseList).
	// CaseGet is CaseGetByID with an explicit tenant check -- unlike the
	// raw dbhandler.CaseGetByID primitive, which is unscoped and relied
	// upon by internal callers that already verify ownership themselves
	// (see verifyCaseOwnership), CaseGet is the public, safe-by-default
	// entry point for the customer-facing GET /v1/cases/{id} route.
	CaseList(ctx context.Context, customerID uuid.UUID, size uint64, token string, status string, ownerType commonidentity.OwnerType, ownerID uuid.UUID, contactID uuid.UUID) ([]*kase.Case, string, error)
	CaseGet(ctx context.Context, customerID, id uuid.UUID) (*kase.Case, error)

	// Create implements design VOIP-1243 §3: a plain, explicit Case
	// creation -- NOT get-or-create. No transaction, no peer lock, no
	// retry loop, no previous_case_id chaining, no owner
	// auto-assignment. Reuses the existing dbhandler.CaseInsert
	// primitive and translates its sentinel errors
	// (dbhandler.ErrDuplicate / dbhandler.ErrDeadlock) into typed
	// cerrors.AlreadyExists / cerrors.Unavailable respectively.
	Create(ctx context.Context, customerID uuid.UUID, self, peer commonaddress.Address, referenceType, name, detail string) (*kase.Case, error)
}

type caseHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	// peerLocks/peerLocksMu implement VOIP-1232's per-(customer_id,
	// peer_type, peer_target, reference_type) in-process serialization
	// lock: a map of buffered channels (capacity 1) used as keyed
	// mutexes, guarded by peerLocksMu for map access. See peerlock.go.
	// caseHandler is a process-wide singleton (constructed once by
	// NewCaseHandler at service startup), so this map is genuinely
	// shared across every concurrent GetOrCreate call for the life of
	// the process.
	peerLocks   map[string]chan struct{}
	peerLocksMu sync.RWMutex
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
		peerLocks:     make(map[string]chan struct{}),
	}
}
