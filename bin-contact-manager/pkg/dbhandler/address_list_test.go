package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

func insertTestAddressAt(t *testing.T, id, customerID, contactID uuid.UUID, addrType, target string, isPrimary int, createdAt time.Time) {
	t.Helper()

	q := `INSERT INTO contact_addresses (id, customer_id, contact_id, type, target, is_primary, tm_create)
		  VALUES (?,?,?,?,?,?,?)`
	if _, err := dbTest.Exec(q,
		id.Bytes(), customerID.Bytes(), contactID.Bytes(),
		addrType, target, isPrimary,
		createdAt.Format("2006-01-02 15:04:05.000000"),
	); err != nil {
		t.Fatalf("insertTestAddressAt() error = %v", err)
	}
}

func Test_AddressListAllByContactID_IncludesEveryTypeOfThatContactOnly(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())
	otherContactID := uuid.Must(uuid.NewV4())
	base := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	telID := uuid.Must(uuid.NewV4())
	emailID := uuid.Must(uuid.NewV4())
	webID := uuid.Must(uuid.NewV4())
	otherID := uuid.Must(uuid.NewV4())
	insertTestAddressAt(t, telID, customerID, contactID, "tel", "+1555040001", 0, base)
	insertTestAddressAt(t, emailID, customerID, contactID, "email", "listall@example.com", 1, base.Add(time.Minute))
	insertTestAddressAt(t, webID, customerID, contactID, "web_session", "session-token-1", 0, base.Add(2*time.Minute))
	insertTestAddressAt(t, otherID, customerID, otherContactID, "tel", "+1555040002", 0, base)

	res, err := h.AddressListAllByContactID(ctx, contactID)
	if err != nil {
		t.Fatalf("AddressListAllByContactID() error = %v", err)
	}

	// Ordered by is_primary desc, tm_create asc, and the other contact's
	// address is never returned.
	wantIDs := []uuid.UUID{emailID, telID, webID}
	if len(res) != len(wantIDs) {
		t.Fatalf("AddressListAllByContactID() returned %d rows, want %d", len(res), len(wantIDs))
	}
	for i, want := range wantIDs {
		if res[i].ID != want {
			t.Errorf("row %d id = %s, want %s", i, res[i].ID, want)
		}
		if res[i].ContactID != contactID {
			t.Errorf("row %d contact id = %s, want %s", i, res[i].ContactID, contactID)
		}
	}
	if res[2].Type != commonaddress.TypeWebSession {
		t.Errorf("row 2 type = %s, want %s", res[2].Type, commonaddress.TypeWebSession)
	}

	empty, err := h.AddressListAllByContactID(ctx, uuid.Must(uuid.NewV4()))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("AddressListAllByContactID(unknown contact) = %v, %v, want an empty non-nil slice", empty, err)
	}
}

func Test_AddressList_Filters(t *testing.T) {
	resetTestDB(t)

	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	contactA := uuid.Must(uuid.NewV4())
	contactB := uuid.Must(uuid.NewV4())
	base := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)

	aTel := uuid.Must(uuid.NewV4())
	aEmail := uuid.Must(uuid.NewV4())
	bTel := uuid.Must(uuid.NewV4())
	insertTestAddressAt(t, aTel, customerID, contactA, "tel", "+1555050001", 0, base)
	insertTestAddressAt(t, aEmail, customerID, contactA, "email", "filters-a@example.com", 0, base.Add(time.Minute))
	insertTestAddressAt(t, bTel, customerID, contactB, "tel", "+1555050002", 0, base.Add(2*time.Minute))

	tests := []struct {
		name      string
		filters   map[string]any
		pageToken string
		pageSize  uint64
		expectIDs []uuid.UUID
	}{
		{
			name:      "contact_id filter excludes the other contact",
			filters:   map[string]any{"contact_id": contactA},
			expectIDs: []uuid.UUID{aEmail, aTel},
		},
		{
			name:      "type filter keeps only that type",
			filters:   map[string]any{"type": "tel"},
			expectIDs: []uuid.UUID{bTel, aTel},
		},
		{
			name:      "contact_id and type combine",
			filters:   map[string]any{"contact_id": contactA, "type": "email"},
			expectIDs: []uuid.UUID{aEmail},
		},
		{
			name:      "nil contact_id and empty type are ignored",
			filters:   map[string]any{"contact_id": uuid.Nil, "type": ""},
			expectIDs: []uuid.UUID{bTel, aEmail, aTel},
		},
		{
			name:      "page token returns rows created before it",
			filters:   map[string]any{},
			pageToken: base.Add(time.Minute).Format("2006-01-02 15:04:05.000000"),
			expectIDs: []uuid.UUID{aTel},
		},
		{
			name:      "page size limits the rows",
			filters:   map[string]any{},
			pageSize:  1,
			expectIDs: []uuid.UUID{bTel},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.AddressList(ctx, customerID, tt.filters, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Fatalf("AddressList() error = %v", err)
			}
			if len(res) != len(tt.expectIDs) {
				t.Fatalf("AddressList() returned %d rows, want %d", len(res), len(tt.expectIDs))
			}
			for i, want := range tt.expectIDs {
				if res[i].ID != want {
					t.Errorf("row %d id = %s, want %s", i, res[i].ID, want)
				}
			}
		})
	}
}
