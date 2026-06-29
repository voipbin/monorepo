package dbhandler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// insertTestAddress inserts a row directly into contact_addresses (explicit SQL,
// no PrepareFields, avoids the generated primary_contact_uk column).
func insertTestAddress(t *testing.T, db *sql.DB, id, customerID, contactID uuid.UUID, addrType, target string) {
	t.Helper()
	q := `INSERT INTO contact_addresses (id, customer_id, contact_id, type, target, is_primary, tm_create)
		  VALUES (?,?,?,?,?,?,?)`
	_, err := db.Exec(q,
		id.Bytes(), customerID.Bytes(), contactID.Bytes(),
		addrType, target,
		0, // is_primary = false
		time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05.000000"),
	)
	if err != nil {
		t.Fatalf("insertTestAddress() error = %v", err)
	}
}

func Test_AddressListByContact(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("ab1b2c3d-0001-0001-0001-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0001-0001-0001-000000000002")

	// Create the parent contact first
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "List",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Insert two addresses directly (no AddressCreate yet)
	addrID1 := uuid.FromStringOrNil("ab1b2c3d-0001-0001-0001-000000000003")
	addrID2 := uuid.FromStringOrNil("ab1b2c3d-0001-0001-0001-000000000004")
	insertTestAddress(t, dbTest, addrID1, customerID, contactID, "tel", "+15551001001")
	insertTestAddress(t, dbTest, addrID2, customerID, contactID, "email", "test@example.com")

	// AddressListByContact → both returned
	res, err := h.AddressListByContact(ctx, customerID, contactID)
	if err != nil {
		t.Fatalf("AddressListByContact() error = %v", err)
	}
	if len(res) < 2 {
		t.Errorf("AddressListByContact() len = %d, want >= 2", len(res))
	}

	// Verify types present
	types := make(map[string]string)
	for _, ap := range res {
		types[ap.Type] = ap.Target
	}
	if types["tel"] != "+15551001001" {
		t.Errorf("AddressListByContact() tel target = %q, want +15551001001", types["tel"])
	}
	if types["email"] != "test@example.com" {
		t.Errorf("AddressListByContact() email target = %q, want test@example.com", types["email"])
	}
}

func Test_AddressGetByID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("ab1b2c3d-0002-0002-0002-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0002-0002-0002-000000000002")
	addrID := uuid.FromStringOrNil("ab1b2c3d-0002-0002-0002-000000000003")

	// Create the parent contact
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "GetByID",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	insertTestAddress(t, dbTest, addrID, customerID, contactID, "tel", "+15551002002")

	// found case
	ap, err := h.AddressGetByID(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGetByID() error = %v", err)
	}
	if ap.Type != "tel" {
		t.Errorf("AddressGetByID() Type = %q, want tel", ap.Type)
	}
	if ap.Target != "+15551002002" {
		t.Errorf("AddressGetByID() Target = %q, want +15551002002", ap.Target)
	}

	// not found: wrong id
	_, err = h.AddressGetByID(ctx, customerID, uuid.FromStringOrNil("ab1b2c3d-ffff-ffff-ffff-ffffffffffff"))
	if err != ErrNotFound {
		t.Errorf("AddressGetByID() expected ErrNotFound, got: %v", err)
	}

	// not found: wrong customerID (cross-tenant guard)
	wrongCustomer := uuid.FromStringOrNil("ab1b2c3d-eeee-eeee-eeee-eeeeeeeeeeee")
	_, err = h.AddressGetByID(ctx, wrongCustomer, addrID)
	if err != ErrNotFound {
		t.Errorf("AddressGetByID() cross-tenant: expected ErrNotFound, got: %v", err)
	}
}
