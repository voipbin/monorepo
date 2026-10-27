package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-contact-manager/models/kase"
)

func newTxTestCase(t *testing.T, customerID uuid.UUID, target string) *kase.Case {
	t.Helper()

	opened := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	return &kase.Case{
		ID:            uuid.Must(uuid.NewV4()),
		CustomerID:    customerID,
		Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: target},
		ReferenceType: "call",
		Owner:         commonidentity.Owner{OwnerType: commonidentity.OwnerTypeNone},
		Status:        kase.StatusOpen,
		OpenedAt:      &opened,
		TMCreate:      &opened,
		TMUpdate:      &opened,
	}
}

func beginTestTx(t *testing.T, h DBHandler) *sql.Tx {
	t.Helper()

	tx, err := h.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return tx
}

// The sqlite test database has a single connection, so a test that holds a
// transaction open reads through the transaction only, and reads through
// h.db only after Commit or Rollback.
func Test_CaseInsertTx_CommitAndRollback(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name         string
		commit       bool
		expectStored bool
	}{
		{name: "commit keeps the row", commit: true, expectStored: true},
		{name: "rollback discards the row", commit: false, expectStored: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTxTestCase(t, customerID, "+155501"+uuid.Must(uuid.NewV4()).String()[:4])

			tx := beginTestTx(t, h)

			if err := h.CaseInsertTx(ctx, tx, c); err != nil {
				t.Fatalf("CaseInsertTx() error = %v", err)
			}

			// Visible inside the transaction.
			got, err := h.CaseGetByIDTx(ctx, tx, c.ID)
			if err != nil {
				t.Fatalf("CaseGetByIDTx() error = %v", err)
			}
			if got.ID != c.ID || got.CustomerID != customerID {
				t.Errorf("CaseGetByIDTx() = %v/%v, want %v/%v", got.ID, got.CustomerID, c.ID, customerID)
			}

			if tt.commit {
				if err := tx.Commit(); err != nil {
					t.Fatalf("Commit() error = %v", err)
				}
			} else if err := tx.Rollback(); err != nil {
				t.Fatalf("Rollback() error = %v", err)
			}

			stored, err := h.CaseGetByID(ctx, c.ID)
			if tt.expectStored {
				if err != nil || stored.ID != c.ID {
					t.Errorf("CaseGetByID() = %v, %v, want the committed row", stored, err)
				}
				return
			}
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("CaseGetByID() error = %v, want ErrNotFound after rollback", err)
			}
		})
	}
}

func Test_CaseGetByIDTx_NotFound(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	tx := beginTestTx(t, h)

	got, err := h.CaseGetByIDTx(ctx, tx, uuid.Must(uuid.NewV4()))
	if !errors.Is(err, ErrNotFound) || got != nil {
		t.Errorf("CaseGetByIDTx() = %v, %v, want nil, ErrNotFound", got, err)
	}
}

func Test_CaseUpdateContactIDTx_And_ClearContactIDTx(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())

	c := newTxTestCase(t, customerID, "+1555020001")
	if err := h.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// Rolled back: contact_id stays NULL.
	tx := beginTestTx(t, h)
	if err := h.CaseUpdateContactIDTx(ctx, tx, customerID, c.ID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactIDTx() error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	got, err := h.CaseGetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.ContactID != nil {
		t.Errorf("contact_id after rollback = %v, want nil", got.ContactID)
	}

	// A different customer id never matches the row.
	tx = beginTestTx(t, h)
	if err := h.CaseUpdateContactIDTx(ctx, tx, otherCustomerID, c.ID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactIDTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	got, err = h.CaseGetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.ContactID != nil {
		t.Errorf("contact_id after a cross-customer update = %v, want nil", got.ContactID)
	}

	// Committed: contact_id is set.
	tx = beginTestTx(t, h)
	if err := h.CaseUpdateContactIDTx(ctx, tx, customerID, c.ID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactIDTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	got, err = h.CaseGetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.ContactID == nil || *got.ContactID != contactID {
		t.Fatalf("contact_id after commit = %v, want %v", got.ContactID, contactID)
	}

	// Clear with the same id and a different customer id leaves the row unchanged.
	tx = beginTestTx(t, h)
	if err := h.CaseClearContactIDTx(ctx, tx, otherCustomerID, c.ID); err != nil {
		t.Fatalf("CaseClearContactIDTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	got, err = h.CaseGetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.ContactID == nil || *got.ContactID != contactID {
		t.Errorf("contact_id after a cross-customer clear = %v, want %v", got.ContactID, contactID)
	}

	// Clear with the owning customer id.
	tx = beginTestTx(t, h)
	if err := h.CaseClearContactIDTx(ctx, tx, customerID, c.ID); err != nil {
		t.Fatalf("CaseClearContactIDTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	got, err = h.CaseGetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.ContactID != nil {
		t.Errorf("contact_id after clear = %v, want nil", got.ContactID)
	}
}

func Test_CaseListAll_SpansCustomers(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	cases := []*kase.Case{
		newTxTestCase(t, uuid.Must(uuid.NewV4()), "+1555030001"),
		newTxTestCase(t, uuid.Must(uuid.NewV4()), "+1555030002"),
	}
	for _, c := range cases {
		if err := h.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert() error = %v", err)
		}
	}

	res, err := h.CaseListAll(ctx)
	if err != nil {
		t.Fatalf("CaseListAll() error = %v", err)
	}

	found := map[uuid.UUID]uuid.UUID{}
	for _, c := range res {
		found[c.ID] = c.CustomerID
	}
	for _, c := range cases {
		if got, ok := found[c.ID]; !ok || got != c.CustomerID {
			t.Errorf("CaseListAll() missing case %s of customer %s", c.ID, c.CustomerID)
		}
	}
}

func Test_BeginTx_Success(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)

	tx, err := h.BeginTx(context.Background())
	if err != nil || tx == nil {
		t.Fatalf("BeginTx() = %v, %v, want a transaction", tx, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback() error = %v", err)
	}
}
