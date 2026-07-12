package dbhandler

// Regression tests for B5: AddressUpdateTx/AddressDeleteTx's UPDATE/DELETE
// statements previously did not check RowsAffected. The post-lock re-read
// each performs (addressTypeTargetContactByID) is a plain SELECT, not
// SELECT ... FOR UPDATE -- it confirms the row's shape a moment before
// the write, not atomically with it. If a concurrent delete/re-target
// removes or moves the exact row in that gap, the UPDATE/DELETE affects
// zero rows and, without an explicit check, returns a silent success
// (having already written a period-table side effect for a row that no
// longer matches). AddressClaimTx already defended against this with a
// RowsAffected guard; AddressUpdateTx/AddressDeleteTx never adopted it.
//
// Uses sqlmock to force a 0-affected-rows outcome that the real SQLite
// test harness's serialized single-connection execution cannot easily
// reproduce (there is no true concurrent second writer in-process).

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressUpdate_ConcurrentDeleteRace_ZeroRowsAffected simulates a
// concurrent AddressDelete winning the race after AddressUpdate's
// post-lock re-read but before its own UPDATE executes: the UPDATE
// affects zero rows. Before the B5 fix this returned nil (silent
// success); after the fix it returns ErrStaleTarget, which the outer
// retry loop treats as retry-eligible.
func Test_AddressUpdate_ConcurrentDeleteRace_ZeroRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()

	h := &handler{utilHandler: mockUtil, db: db, cache: mockCache}
	ctx := context.Background()

	addrID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000002")
	contactID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000003")

	// Every retry iteration's pre-lock read (address.go's AddressUpdate
	// loop) still finds the row present -- the concurrent delete has not
	// committed from this reader's point of view until AFTER the final
	// retry's own re-read, matching a real narrow TOCTOU window.
	for i := 0; i < addressMaxDeadlockRetries; i++ {
		mock.ExpectQuery("SELECT type, target, contact_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}).
				AddRow(string(commonaddress.TypeTel), "+155****7001", contactID.Bytes()))
		mock.ExpectQuery("SELECT customer_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(customerID.Bytes()))

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT type, target, contact_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}).
				AddRow(string(commonaddress.TypeTel), "+155****7001", contactID.Bytes()))
		mock.ExpectExec("UPDATE contact_addresses SET").
			WillReturnResult(sqlmock.NewResult(0, 0)) // the concurrent delete already removed this row
		mock.ExpectRollback()
	}

	err = h.AddressUpdate(ctx, addrID, map[string]any{"name": "Updated Name"})
	if err == nil {
		t.Fatal("AddressUpdate() error = nil, want an exhausted-retries error (0 rows affected must never be a silent success)")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// Test_AddressDelete_ConcurrentDeleteRace_ZeroRowsAffected simulates the
// same TOCTOU race on the delete path: a concurrent delete/re-target
// already removed the row by the time this DELETE executes.
func Test_AddressDelete_ConcurrentDeleteRace_ZeroRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()

	h := &handler{utilHandler: mockUtil, db: db, cache: mockCache}
	ctx := context.Background()

	addrID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000004")
	customerID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000005")
	contactID := uuid.FromStringOrNil("77000000-0000-0000-0000-000000000006")
	periodCols := []string{"id", "contact_id", "valid_from", "valid_to"}

	for i := 0; i < addressMaxDeadlockRetries; i++ {
		mock.ExpectQuery("SELECT type, target, contact_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}).
				AddRow(string(commonaddress.TypeTel), "+155****7002", contactID.Bytes()))
		mock.ExpectQuery("SELECT customer_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(customerID.Bytes()))

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT type, target, contact_id FROM contact_addresses WHERE").
			WithArgs(addrID.Bytes()).
			WillReturnRows(sqlmock.NewRows([]string{"type", "target", "contact_id"}).
				AddRow(string(commonaddress.TypeTel), "+155****7002", contactID.Bytes()))
		mock.ExpectQuery("SELECT .* FROM contact_address_ownership_periods WHERE").
			WillReturnRows(sqlmock.NewRows(periodCols).
				AddRow(uuid.Must(uuid.NewV4()).Bytes(), contactID.Bytes(), nil, nil)) // this contact's own open period
		mock.ExpectExec("UPDATE contact_address_ownership_periods SET").
			WillReturnResult(sqlmock.NewResult(1, 1)) // close it (closeOwnOpenPeriodTx runs BEFORE the row delete)
		mock.ExpectExec("DELETE FROM contact_addresses WHERE").
			WillReturnResult(sqlmock.NewResult(0, 0)) // the concurrent winner already removed this row
		mock.ExpectRollback()
	}

	err = h.AddressDelete(ctx, addrID)
	if err == nil {
		t.Fatal("AddressDelete() error = nil, want an exhausted-retries error (0 rows affected must never be a silent success)")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
