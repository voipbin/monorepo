package dbhandler

// Regression test for the round-3 code review finding: AddressCreate's
// outer retry loop previously shared a single attempt counter between
// genuine deadlock retries (design §5.3, addressMaxDeadlockRetries) and
// the one-shot stale-row duplicate-key repair-and-retry sequence (design
// §4 round-27/28/32). If addressMaxDeadlockRetries-1 deadlocks had
// already consumed the budget, a successful repair on the final attempt
// would exit the loop WITHOUT ever retrying the actual write -- silently
// reporting "exhausted retries under sustained deadlock" even though no
// deadlock caused the final outcome. Fixed in commit eacc72270 by
// tracking deadlockAttempts and repairAttempted as independent counters.
//
// Uses sqlmock (not the real SQLite in-memory DB) to inject a real
// *mysql_driver.MySQLError deadlock (errno 1213) and duplicate-key error
// (errno 1062) -- SQLite has no equivalent driver error to synthesize
// these conditions. Same convention as kase_test.go's
// Test_CaseGetOpenByPeer.

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressCreate_DeadlockBudget_RepairDoesNotConsumeIt drives
// AddressCreate through addressMaxDeadlockRetries-1 (2) genuine
// deadlocks, then a duplicate-key collision on the FINAL allotted
// attempt whose repair succeeds. Before commit eacc72270, this exact
// sequence would have exhausted deadlockAttempts on the repair's own
// loop turn and returned "exhausted retries under sustained deadlock"
// without ever retrying the actual INSERT. After the fix, the repair
// does not consume budget, and the retried INSERT is given its own
// (unbudgeted) attempt and succeeds.
func Test_AddressCreate_DeadlockBudget_RepairDoesNotConsumeIt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
	mockCache.EXPECT().ContactSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, ErrNotFound).AnyTimes()

	h := &handler{utilHandler: mockUtil, db: db, cache: mockCache}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("79000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("79000000-0000-0000-0000-000000000002")
	deadContactID := uuid.FromStringOrNil("79000000-0000-0000-0000-000000000003")
	addrID := uuid.FromStringOrNil("79000000-0000-0000-0000-000000000004")
	target := "+155****9001"

	periodCols := []string{"id", "contact_id", "valid_from", "valid_to"}
	deadlockErr := &mysql_driver.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock; try restarting transaction"}
	dupErr := &mysql_driver.MySQLError{Number: 1062, Message: "Duplicate entry for key 'idx_contact_addresses_identifier'"}

	// --- Attempt 1: genuine deadlock (deadlockAttempts: 0 -> 1) ---
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM contact_address_ownership_periods WHERE").
		WillReturnRows(sqlmock.NewRows(periodCols)) // no rows -- Step 5, first registration
	mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO contact_addresses").
		WillReturnError(deadlockErr)
	mock.ExpectRollback()

	// --- Attempt 2: genuine deadlock (deadlockAttempts: 1 -> 2) ---
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM contact_address_ownership_periods WHERE").
		WillReturnRows(sqlmock.NewRows(periodCols))
	mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO contact_addresses").
		WillReturnError(deadlockErr)
	mock.ExpectRollback()

	// --- Attempt 3 (deadlockAttempts still 2 -- the "last budgeted"
	// attempt under the OLD shared-counter bug): duplicate-key
	// collision, occupying row belongs to a tombstoned Contact. ---
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM contact_address_ownership_periods WHERE").
		WillReturnRows(sqlmock.NewRows(periodCols))
	mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO contact_addresses").
		WillReturnError(dupErr)
	mock.ExpectRollback()

	// --- Repair (its OWN fresh transaction, design §4 round-32; does
	// NOT consume deadlockAttempts). Occupying row's owner is
	// tombstoned -> close a fabricated period for it and hard-delete
	// the stale row. ---
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, contact_id, tm_create FROM contact_addresses WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "tm_create"}).
			AddRow(addrID.Bytes(), deadContactID.Bytes(), curTime.Add(-48*time.Hour).Format("2006-01-02 15:04:05.999999-07:00")))
	mock.ExpectQuery("SELECT tm_delete FROM contact_contacts WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"tm_delete"}).
			AddRow(curTime.Add(-1 * time.Hour).Format("2006-01-02 15:04:05.999999-07:00")))
	mock.ExpectQuery("SELECT valid_to FROM contact_address_ownership_periods WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"valid_to"})) // no prior closed period
	mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
		WillReturnResult(sqlmock.NewResult(1, 1)) // fabricated closed period for the dead owner
	mock.ExpectExec("DELETE FROM contact_addresses WHERE").
		WillReturnResult(sqlmock.NewResult(1, 1)) // vacate the slot (AddressCreate path: hard-delete)
	mock.ExpectCommit()

	// --- Attempt 4 (the retried create, in a fresh transaction --
	// this is the attempt that the round-3 bug would have skipped
	// entirely). Now the vacated slot lets the INSERT succeed. ---
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM contact_address_ownership_periods WHERE").
		WillReturnRows(sqlmock.NewRows(periodCols).
			AddRow(uuid.Must(uuid.NewV4()).Bytes(), deadContactID.Bytes(),
				curTime.Add(-48*time.Hour).Format("2006-01-02 15:04:05.999999-07:00"),
				curTime.Add(-1*time.Hour).Format("2006-01-02 15:04:05.999999-07:00"))) // the repair's fabricated closed period -- Step 4 (StepReassign)
	mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
		WillReturnResult(sqlmock.NewResult(1, 1)) // new open period for the real caller
	mock.ExpectExec("INSERT INTO contact_addresses").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// contactUpdateToCache's read-back (contact.go's contactGetFromDB +
	// AddressListByContactID + TagAssignmentListByContactID) -- loosely
	// matched, not the subject of this test.
	mock.ExpectQuery("SELECT .* FROM contact_contacts WHERE").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "customer_id", "first_name", "last_name", "display_name", "company", "job_title",
			"source", "external_id", "notes", "tm_create", "tm_update", "tm_delete",
		}).AddRow(contactID.Bytes(), customerID.Bytes(), "", "", "", "", "", "manual", "", "", curTime, curTime, nil))
	mock.ExpectQuery("SELECT .* FROM contact_addresses WHERE").
		WillReturnRows(sqlmock.NewRows(addressRowColumns()))
	mock.ExpectQuery("SELECT .* FROM contact_tag_assignments WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"tag_id"}))

	a := &contact.Address{
		Address:    commonaddress.Address{Type: contact.AddressTypeTel, Target: target},
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v, want success (repair-then-retry must not be starved by the deadlock budget)", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
