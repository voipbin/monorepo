package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// DBHandler interface for contact-manager database operations
type DBHandler interface {
	// Contact operations
	ContactCreate(ctx context.Context, c *contact.Contact) error
	ContactGet(ctx context.Context, id uuid.UUID) (*contact.Contact, error)
	ContactList(ctx context.Context, size uint64, token string, filters map[contact.Field]any) ([]*contact.Contact, error)
	ContactUpdate(ctx context.Context, id uuid.UUID, fields map[contact.Field]any) error
	ContactDelete(ctx context.Context, id uuid.UUID) error
	ContactLookupByPhone(ctx context.Context, customerID uuid.UUID, phoneE164 string) (*contact.Contact, error)
	ContactLookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error)
	AddressLookupContactIDByTypeTarget(ctx context.Context, customerID uuid.UUID, addrType commonaddress.Type, target string) (uuid.UUID, error)
	ContactDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error

	// Address operations
	AddressCreate(ctx context.Context, a *contact.Address) error
	AddressGet(ctx context.Context, customerID, id uuid.UUID) (*contact.Address, error)
	AddressList(ctx context.Context, customerID uuid.UUID, filters map[string]any, pageToken string, pageSize uint64) ([]contact.Address, error)
	AddressListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.Address, error)
	AddressUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error
	AddressDelete(ctx context.Context, id uuid.UUID) error
	AddressResetPrimary(ctx context.Context, contactID uuid.UUID) error
	AddressClaim(ctx context.Context, customerID, addressID, contactID uuid.UUID) error

	// Address ownership-period operations (design
	// docs/plans/2026-07-11-contact-address-ownership-integrity-design.md
	// §4/§5.1, NOJIRA-contact-address-ownership-periods Phase 1)
	OwnershipPeriodsLockAndResolveTx(ctx context.Context, tx *sql.Tx, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) (int, []OwnershipPeriod, error)
	AddressCreateTx(ctx context.Context, tx *sql.Tx, a *contact.Address) error
	AddressUpdateTx(ctx context.Context, tx *sql.Tx, id, customerID, contactID uuid.UUID, oldType commonaddress.Type, oldTarget string, fields map[string]any) error
	AddressDeleteTx(ctx context.Context, tx *sql.Tx, addressID, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error
	AddressClaimTx(ctx context.Context, tx *sql.Tx, customerID, addressID, contactID uuid.UUID, addrType commonaddress.Type, target string) error
	AddressResetPrimaryTx(ctx context.Context, tx *sql.Tx, contactID uuid.UUID) error
	AddressDeleteCompensating(ctx context.Context, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error

	// Address ownership-period READ operations (design §6.1/§6.2,
	// NOJIRA-contact-address-ownership-periods Phase 2). Used exclusively
	// by interactionListByContact's STEP1 -- AddressListByContactID above
	// stays untouched, since it also backs the public Contact.Addresses
	// API field via ContactGet/ContactList (design §6.1).
	OwnershipPeriodsListByContactID(ctx context.Context, contactID uuid.UUID) ([]OwnershipPeriod, error)
	MissingPeriodOwnedAddresses(ctx context.Context, customerID, contactID uuid.UUID) ([]MissingPeriodAddress, error)

	// TagAssignment operations
	TagAssignmentCreate(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentDelete(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentListByContactID(ctx context.Context, contactID uuid.UUID) ([]uuid.UUID, error)

	// Interaction operations
	InteractionCreate(ctx context.Context, i *interaction.Interaction) error
	InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error)
	InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string, peerType, peerTarget string, addressSet []AddressPair, since time.Time) ([]*interaction.Interaction, error)
	InteractionListByOwnershipPeriods(ctx context.Context, customerID uuid.UUID, size uint64, token string, peerType, peerTarget string, bounds []OwnershipPeriodBound, since time.Time) ([]*interaction.Interaction, error)
	InteractionListByIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) ([]*interaction.Interaction, error)
	InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) ([]*interaction.Interaction, error)

	// Resolution operations
	ResolutionCreate(ctx context.Context, r *resolution.Resolution) error
	ResolutionCreateTx(ctx context.Context, tx *sql.Tx, r *resolution.Resolution) error
	ResolutionDelete(ctx context.Context, customerID, interactionID, id uuid.UUID) error
	ResolutionDeleteByCase(ctx context.Context, customerID, caseID, id uuid.UUID) error
	ResolutionDeleteByCaseTx(ctx context.Context, tx *sql.Tx, customerID, caseID, id uuid.UUID) error
	ResolutionListByInteraction(ctx context.Context, customerID, interactionID uuid.UUID) ([]*resolution.Resolution, error)
	ResolutionListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*resolution.Resolution, error)
	ResolutionListByCaseTx(ctx context.Context, tx *sql.Tx, customerID, caseID uuid.UUID) ([]*resolution.Resolution, error)
	ResolutionListByContact(ctx context.Context, customerID, contactID uuid.UUID) ([]*resolution.Resolution, error)

	// Case operations
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CaseInsert(ctx context.Context, c *kase.Case) error
	CaseInsertTx(ctx context.Context, tx *sql.Tx, c *kase.Case) error
	CaseGetByID(ctx context.Context, id uuid.UUID) (*kase.Case, error)
	CaseGetByIDTx(ctx context.Context, tx *sql.Tx, id uuid.UUID) (*kase.Case, error)
	CaseGetByIDForUpdate(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID) (*kase.Case, error)
	CaseGetOpenByPeer(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error)
	CaseUpdateStatusClosed(ctx context.Context, customerID, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error)
	CaseUpdateStatusClosedTx(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error)
	CaseUpdateTMUpdate(ctx context.Context, id uuid.UUID, tmUpdate *time.Time) error
	CaseUpdateTMUpdateTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, tmUpdate *time.Time) error
	CaseUpdateContactID(ctx context.Context, customerID, id, contactID uuid.UUID) error
	CaseUpdateContactIDTx(ctx context.Context, tx *sql.Tx, customerID, id, contactID uuid.UUID) error
	CaseClearContactIDTx(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID) error
	CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error)
	CaseListAll(ctx context.Context) ([]*kase.Case, error)

	// CaseNote operations (design §3.5)
	CaseNoteCreate(ctx context.Context, n *casenote.CaseNote) error
	CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error
	CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error)

	// CaseTagAssignment operations (design §7 round-22 correction)
	CaseTagAssignmentCreate(ctx context.Context, caseID, tagID uuid.UUID) error
	CaseTagAssignmentDelete(ctx context.Context, caseID, tagID uuid.UUID) error
	CaseTagAssignmentListByCaseID(ctx context.Context, caseID uuid.UUID) ([]uuid.UUID, error)
	CaseListByOwner(ctx context.Context, customerID uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) ([]*kase.Case, error)
	CaseGetLastClosedByPeer(ctx context.Context, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error)
	CaseGetLastClosedByPeerTx(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error)

	// CaseList returns Cases scoped to customerID, optionally filtered by
	// status (empty string = no status filter) and/or owner
	// (ownerType == commonidentity.OwnerTypeNone or ownerID == uuid.Nil =
	// no owner filter). Backs the Phase 5 RPC/REST GET /v1/cases?...
	// list surface (design §9).
	CaseList(ctx context.Context, customerID uuid.UUID, size uint64, token string, status string, ownerType commonidentity.OwnerType, ownerID uuid.UUID, contactID uuid.UUID) ([]*kase.Case, error)
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler

	// forUpdateSuffix is "FOR UPDATE" against MySQL and "" against the
	// SQLite in-memory test DB (SQLite has no FOR UPDATE syntax; its
	// whole-connection/file-level locking already serializes writers
	// coarsely enough for tests, which never run concurrent goroutines
	// against the same in-memory DB across separate connections anyway).
	// Detected once at construction via the driver's type name -- see
	// NewHandler.
	forUpdateSuffix string
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("record not found")
	ErrConflict = fmt.Errorf("address already claimed")

	// ErrDuplicateTarget is returned by AddressCreate when the insert
	// violates the unique index on contact_addresses(customer_id, type,
	// target). Distinct from ErrConflict, whose message ("address already
	// claimed") is specific to the ClaimAddress flow.
	ErrDuplicateTarget = fmt.Errorf("address already exists for this customer")

	// ErrDuplicate is returned by CaseInsert when the insert violates
	// uq_case_open_peer (contact_cases' partial-unique invariant: at most
	// one OPEN case per customer/peer/reference_type -- see
	// contact-case-management design §3.1). Callers (the get-or-create
	// algorithm, design §4) use this as the signal to retry with a locked
	// re-select rather than treating it as a generic infrastructure error.
	ErrDuplicate = fmt.Errorf("an open case already exists for this customer/peer/reference_type")

	// ErrDeadlock is returned by any Case* Tx-scoped operation (design §4's
	// get-or-create transaction) when the underlying driver reports a
	// MySQL deadlock (errno 1213, "Deadlock found when trying to get
	// lock; try restarting transaction"). VOIP-1232: concurrent
	// GetOrCreate calls racing to INSERT into contact_cases for the same
	// (customer_id, peer_type, peer_target, reference_type) tuple can
	// occasionally surface a 1213 instead of the clean 1062/ErrDuplicate
	// path, because the collision can occur during the insert-intention
	// gap-lock phase before either transaction commits. InnoDB
	// auto-rolls-back the ENTIRE transaction server-side when it reports
	// 1213 -- callers MUST NOT reuse the same *sql.Tx after seeing this
	// error; a correct retry restarts from a fresh BeginTx (design §4.2's
	// insert-retry loop is a different, narrower mechanism scoped to
	// ErrDuplicate only and does not cover this).
	ErrDeadlock = fmt.Errorf("deadlock detected; transaction was rolled back by the server")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	// Detect SQLite (used only by the in-memory test harness) vs. MySQL
	// (production) by the driver's concrete type name, so FOR UPDATE
	// locking clauses can be conditionally omitted where the driver
	// doesn't support the syntax. reflect avoids importing the sqlite3
	// driver package here (kept as a test-only dependency).
	driverType := fmt.Sprintf("%T", db.Driver())
	forUpdateSuffix := "FOR UPDATE"
	if strings.Contains(strings.ToLower(driverType), "sqlite") {
		forUpdateSuffix = ""
	}

	h := &handler{
		utilHandler:     utilhandler.NewUtilHandler(),
		db:              db,
		cache:           cache,
		forUpdateSuffix: forUpdateSuffix,
	}
	return h
}
