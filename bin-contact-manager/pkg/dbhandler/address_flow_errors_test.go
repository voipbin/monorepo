package dbhandler

// Error path and retry tests for the address write flows that wrap the
// transaction level writers in a begin, commit and retry loop (AddressCreate,
// AddressClaim, AddressUpdate, AddressDelete, AddressResetPrimary). They use
// sqlmock only.

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	preLockSQL   = "SELECT type, target, contact_id FROM contact_addresses"
	customerSQL  = "SELECT customer_id FROM contact_addresses"
	staleRowSQL  = "SELECT id, contact_id, tm_create FROM contact_addresses"
	tombstoneSQL = "SELECT tm_delete FROM contact_contacts"
	lockSQL      = "FROM contact_address_ownership_periods"
)

// flowCase is one scenario of a flow test table.
type flowCase struct {
	name     string
	setup    func(mock sqlmock.Sqlmock)
	call     func(h *handler) error
	wantErr  error // compared with ==
	wantText string
	wantIs   error
}

func runFlowCases(t *testing.T, tests []flowCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			err := tt.call(h)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
			} else if tt.wantText == "" {
				if err != nil {
					t.Errorf("error = %v, want nil", err)
				}
			} else {
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func preLockRows(addrType commonaddress.Type, target string, contactID uuid.UUID) *sqlmock.Rows {
	var contactValue any
	if contactID != uuid.Nil {
		contactValue = contactID.Bytes()
	}
	return sqlmock.NewRows([]string{"type", "target", "contact_id"}).AddRow(string(addrType), target, contactValue)
}

func customerRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"customer_id"}).AddRow(txTestCustomerID.Bytes())
}

func fullAddressRows(contactID uuid.UUID, target string) *sqlmock.Rows {
	var contactValue any
	if contactID != uuid.Nil {
		contactValue = contactID.Bytes()
	}
	return sqlmock.NewRows(addressRowColumns()).AddRow(
		txTestAddressID.Bytes(), txTestCustomerID.Bytes(), contactValue,
		string(commonaddress.TypeTel), target, "", "", "", false,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	)
}

func Test_AddressCreate_Flow(t *testing.T) {
	ctx := context.Background()
	unresolved := func() *contact.Address {
		return &contact.Address{
			Address:    commonaddress.Address{Type: commonaddress.TypeTel, Target: txTestOldTarget},
			ID:         txTestAddressID,
			CustomerID: txTestCustomerID,
		}
	}
	resolved := func() *contact.Address {
		a := unresolved()
		a.ContactID = txTestContactID
		return a
	}

	runFlowCases(t, []flowCase{
		{
			name:  "begin failure",
			setup: func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errCaseBoom) },
			call: func(h *handler) error {
				return h.AddressCreate(ctx, unresolved())
			},
			wantText: "could not begin transaction. AddressCreate",
			wantIs:   errCaseBoom,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO contact_addresses").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			call: func(h *handler) error {
				return h.AddressCreate(ctx, unresolved())
			},
			wantText: "could not commit transaction. AddressCreate",
			wantIs:   errCaseBoom,
		},
		{
			name: "sustained deadlock exhausts the retries",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					mock.ExpectBegin()
					mock.ExpectExec("INSERT INTO contact_addresses").WillReturnError(mysqlDeadlock())
					mock.ExpectRollback()
				}
			},
			call: func(h *handler) error {
				return h.AddressCreate(ctx, unresolved())
			},
			wantText: "could not create address: exhausted retries under sustained deadlock",
			wantIs:   ErrDeadlock,
		},
		{
			name: "a failing repair after a duplicate target is returned",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call: func(h *handler) error {
				return h.AddressCreate(ctx, resolved())
			},
			wantText: "could not query. staleRowByTargetTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "a deadlock inside the repair is retried and the duplicate is then reported",
			setup: func(mock sqlmock.Sqlmock) {
				// first create attempt hits the duplicate target
				mock.ExpectBegin()
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
				// the repair deadlocks
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnError(mysqlDeadlock())
				mock.ExpectRollback()
				// the create is attempted again and reports the duplicate, the repair is one-shot
				mock.ExpectBegin()
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
			},
			call: func(h *handler) error {
				return h.AddressCreate(ctx, resolved())
			},
			wantErr: ErrDuplicateTarget,
		},
	})
}

func Test_AddressClaim_Flow(t *testing.T) {
	ctx := context.Background()
	claimer := txTestContactID
	owner := txTestOtherID
	call := func(force bool) func(h *handler) error {
		return func(h *handler) error {
			return h.AddressClaim(ctx, txTestCustomerID, txTestAddressID, claimer, force)
		}
	}
	existing := func(mock sqlmock.Sqlmock, ownerID uuid.UUID) {
		mock.ExpectQuery("FROM contact_addresses WHERE id").WillReturnRows(fullAddressRows(ownerID, txTestOldTarget))
	}
	attemptStart := func(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
		mock.ExpectBegin()
		mock.ExpectQuery(preLockSQL).WillReturnRows(rows)
	}

	runFlowCases(t, []flowCase{
		{
			name: "address lookup failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses WHERE id").WillReturnError(errCaseBoom)
			},
			call:     call(false),
			wantText: "could not query. AddressGet",
			wantIs:   errCaseBoom,
		},
		{
			name: "address lookup scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses WHERE id").WillReturnRows(twoColumnRow())
			},
			call:     call(false),
			wantText: "could not scan. AddressGet",
		},
		{
			name: "unknown address is not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses WHERE id").WillReturnRows(sqlmock.NewRows(addressRowColumns()))
			},
			call:    call(false),
			wantErr: ErrNotFound,
		},
		{
			name: "an address that already belongs to the contact needs no write",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, claimer)
			},
			// no error and no further expectation: the mock fails the call on any extra query
			call: call(false),
		},
		{
			name: "begin failure",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				mock.ExpectBegin().WillReturnError(errCaseBoom)
			},
			call:     call(false),
			wantText: "could not begin transaction. AddressClaim",
			wantIs:   errCaseBoom,
		},
		{
			name: "pre-lock read failure",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				mock.ExpectBegin()
				mock.ExpectQuery(preLockSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     call(false),
			wantText: "could not query. addressTypeTargetContactByID",
			wantIs:   errCaseBoom,
		},
		{
			name: "a target that keeps changing exhausts the retries",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					attemptStart(mock, preLockRows(commonaddress.TypeTel, "+15550009999", owner))
					mock.ExpectRollback()
				}
			},
			call:     call(false),
			wantText: "could not claim address: exhausted retries under sustained deadlock",
			wantIs:   ErrStaleTarget,
		},
		{
			name: "tombstone lookup failure",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, owner))
				mock.ExpectQuery(tombstoneSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     call(false),
			wantText: "could not query. contactTombstoneTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "a live owner without force is a conflict",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, owner))
				mock.ExpectQuery(tombstoneSQL).WillReturnRows(sqlmock.NewRows([]string{"tm_delete"}).AddRow(nil))
				mock.ExpectRollback()
			},
			call:    call(false),
			wantErr: ErrConflict,
		},
		{
			name: "a forced claim stops when the owner period cannot be closed",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, owner))
				mock.ExpectQuery(tombstoneSQL).WillReturnRows(sqlmock.NewRows([]string{"tm_delete"}).AddRow(nil))
				mock.ExpectQuery(lockSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     call(true),
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "a failing repair of a deleted owner is returned",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, owner)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, owner))
				mock.ExpectQuery(tombstoneSQL).WillReturnRows(sqlmock.NewRows([]string{"tm_delete"}).AddRow(time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)))
				mock.ExpectQuery(staleRowSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     call(false),
			wantText: "could not query. staleRowByTargetTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "the claim write failure is returned",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, uuid.Nil)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, uuid.Nil))
				mock.ExpectQuery(lockSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     call(false),
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				existing(mock, uuid.Nil)
				attemptStart(mock, preLockRows(commonaddress.TypeTel, txTestOldTarget, uuid.Nil))
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			call:     call(false),
			wantText: "could not commit transaction. AddressClaim",
			wantIs:   errCaseBoom,
		},
	})
}

func Test_AddressUpdate_Flow(t *testing.T) {
	ctx := context.Background()
	update := func(fields map[string]any) func(h *handler) error {
		return func(h *handler) error {
			return h.AddressUpdate(ctx, txTestAddressID, fields)
		}
	}
	prelude := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
		mock.ExpectQuery(customerSQL).WillReturnRows(customerRows())
	}
	inTx := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
	}
	rename := map[string]any{"name": "renamed"}

	runFlowCases(t, []flowCase{
		{
			name: "unknown address is not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}))
			},
			call:    update(rename),
			wantErr: ErrNotFound,
		},
		{
			name: "customer lookup failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(customerSQL).WillReturnError(errCaseBoom)
			},
			call:     update(rename),
			wantText: "could not query. addressCustomerID",
			wantIs:   errCaseBoom,
		},
		{
			name: "customer lookup without a row",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(customerSQL).WillReturnRows(sqlmock.NewRows([]string{"customer_id"}))
			},
			call:    update(rename),
			wantErr: ErrNotFound,
		},
		{
			name: "customer lookup scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(customerSQL).WillReturnRows(twoColumnRow())
			},
			call:     update(rename),
			wantText: "could not scan customer_id. addressCustomerID",
		},
		{
			name: "customer lookup parse failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(customerSQL).WillReturnRows(badUUIDRow("customer_id"))
			},
			call:     update(rename),
			wantText: "could not parse customer_id. addressCustomerID",
		},
		{
			name: "begin failure",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				mock.ExpectBegin().WillReturnError(errCaseBoom)
			},
			call:     update(rename),
			wantText: "could not begin transaction. AddressUpdate",
			wantIs:   errCaseBoom,
		},
		{
			name: "re-read inside the transaction fails",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				mock.ExpectBegin()
				mock.ExpectQuery(preLockSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     update(rename),
			wantText: "could not query. addressTypeTargetContactByID",
			wantIs:   errCaseBoom,
		},
		{
			name: "a target that keeps changing exhausts the retries",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					prelude(mock)
					mock.ExpectBegin()
					mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, "+15550009999", txTestContactID))
					mock.ExpectRollback()
				}
			},
			call:     update(rename),
			wantText: "could not update address: exhausted retries under sustained deadlock",
			wantIs:   ErrStaleTarget,
		},
		{
			name: "primary reset failure",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				inTx(mock)
				mock.ExpectExec("UPDATE contact_addresses SET is_primary").WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     update(map[string]any{"is_primary": true}),
			wantText: "could not execute. AddressResetPrimaryTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				inTx(mock)
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			call:     update(rename),
			wantText: "could not commit transaction. AddressUpdate",
			wantIs:   errCaseBoom,
		},
		{
			name: "a failing repair after a duplicate target is returned",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				inTx(mock)
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     update(map[string]any{"target": txTestNewHigh}),
			wantText: "could not query. staleRowByTargetTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "a deadlock inside the repair is retried",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				inTx(mock)
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnError(mysqlDeadlock())
				mock.ExpectRollback()
				// the update is attempted again and its duplicate is reported, the repair is one-shot
				prelude(mock)
				inTx(mock)
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
			},
			call:    update(map[string]any{"target": txTestNewHigh}),
			wantErr: ErrDuplicateTarget,
		},
		{
			name: "a repair that frees the target retries the update at once",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				inTx(mock)
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
				mock.ExpectRollback()
				// the repair finds no stale row and commits, the update then succeeds
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "tm_create"}))
				mock.ExpectCommit()
				prelude(mock)
				inTx(mock)
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			call:     update(map[string]any{"target": txTestNewHigh}),
			wantText: "",
		},
	})
}

func Test_AddressDelete_Flow(t *testing.T) {
	ctx := context.Background()
	del := func(h *handler) error { return h.AddressDelete(ctx, txTestAddressID) }
	prelude := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
		mock.ExpectQuery(customerSQL).WillReturnRows(customerRows())
	}

	runFlowCases(t, []flowCase{
		{
			name: "unknown address is not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}))
			},
			call:    del,
			wantErr: ErrNotFound,
		},
		{
			name: "customer lookup failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(customerSQL).WillReturnError(errCaseBoom)
			},
			call:     del,
			wantText: "could not query. addressCustomerID",
			wantIs:   errCaseBoom,
		},
		{
			name: "begin failure",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				mock.ExpectBegin().WillReturnError(errCaseBoom)
			},
			call:     del,
			wantText: "could not begin transaction. AddressDelete",
			wantIs:   errCaseBoom,
		},
		{
			name: "re-read inside the transaction fails",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				mock.ExpectBegin()
				mock.ExpectQuery(preLockSQL).WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     del,
			wantText: "could not query. addressTypeTargetContactByID",
			wantIs:   errCaseBoom,
		},
		{
			name: "a target that keeps changing exhausts the retries",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					prelude(mock)
					mock.ExpectBegin()
					mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, "+15550009999", txTestContactID))
					mock.ExpectRollback()
				}
			},
			call:     del,
			wantText: "could not delete address: exhausted retries under sustained deadlock",
			wantIs:   ErrStaleTarget,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				prelude(mock)
				mock.ExpectBegin()
				mock.ExpectQuery(preLockSQL).WillReturnRows(preLockRows(commonaddress.TypeTel, txTestOldTarget, txTestContactID))
				mock.ExpectQuery(lockSQL).WillReturnRows(emptyPeriods())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			call:     del,
			wantText: "could not commit transaction. AddressDelete",
			wantIs:   errCaseBoom,
		},
	})
}

func Test_AddressResetPrimary_Flow(t *testing.T) {
	ctx := context.Background()
	reset := func(h *handler) error { return h.AddressResetPrimary(ctx, txTestContactID) }

	runFlowCases(t, []flowCase{
		{
			name:     "begin failure",
			setup:    func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errCaseBoom) },
			call:     reset,
			wantText: "could not begin transaction. AddressResetPrimary",
			wantIs:   errCaseBoom,
		},
		{
			name: "update failure rolls back",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE contact_addresses SET is_primary").WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			call:     reset,
			wantText: "could not execute. AddressResetPrimaryTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE contact_addresses SET is_primary").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			call:     reset,
			wantText: "could not commit transaction. AddressResetPrimary",
			wantIs:   errCaseBoom,
		},
	})
}
