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

func Test_AddressListByContactID(t *testing.T) {
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
	insertTestAddress(t, dbTest, addrID1, customerID, contactID, "tel", "+155****1001")
	insertTestAddress(t, dbTest, addrID2, customerID, contactID, "email", "test@example.com")

	// AddressListByContactID → both returned
	res, err := h.AddressListByContactID(ctx, contactID)
	if err != nil {
		t.Fatalf("AddressListByContactID() error = %v", err)
	}
	if len(res) < 2 {
		t.Errorf("AddressListByContactID() len = %d, want >= 2", len(res))
	}

	// Verify types present
	types := make(map[string]string)
	for _, a := range res {
		types[a.Type] = a.Target
	}
	if types["tel"] != "+155****1001" {
		t.Errorf("AddressListByContactID() tel target = %q, want +155****1001", types["tel"])
	}
	if types["email"] != "test@example.com" {
		t.Errorf("AddressListByContactID() email target = %q, want test@example.com", types["email"])
	}
}

func Test_AddressGet(t *testing.T) {
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
		LastName:  "Get",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	insertTestAddress(t, dbTest, addrID, customerID, contactID, "tel", "+155****2002")

	// found case
	a, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if a.Type != "tel" {
		t.Errorf("AddressGet() Type = %q, want tel", a.Type)
	}
	if a.Target != "+155****2002" {
		t.Errorf("AddressGet() Target = %q, want +155****2002", a.Target)
	}

	// not found: wrong id
	_, err = h.AddressGet(ctx, customerID, uuid.FromStringOrNil("ab1b2c3d-ffff-ffff-ffff-ffffffffffff"))
	if err != ErrNotFound {
		t.Errorf("AddressGet() expected ErrNotFound, got: %v", err)
	}

	// not found: wrong customerID (cross-tenant guard)
	wrongCustomer := uuid.FromStringOrNil("ab1b2c3d-eeee-eeee-eeee-eeeeeeeeeeee")
	_, err = h.AddressGet(ctx, wrongCustomer, addrID)
	if err != ErrNotFound {
		t.Errorf("AddressGet() cross-tenant: expected ErrNotFound, got: %v", err)
	}
}

func Test_AddressCreate(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0003-0003-0003-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0003-0003-0003-000000000002")
	addrID := uuid.FromStringOrNil("ab1b2c3d-0003-0003-0003-000000000003")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	// Create the parent contact
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "Create",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Create address
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	a := &contact.Address{
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       contact.AddressTypeTel,
		Target:     "+155****3001",
		IsPrimary:  true,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Verify it was created
	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.Type != contact.AddressTypeTel {
		t.Errorf("AddressCreate() Type = %q, want %q", got.Type, contact.AddressTypeTel)
	}
	if got.Target != "+155****3001" {
		t.Errorf("AddressCreate() Target = %q, want +155****3001", got.Target)
	}
	if !got.IsPrimary {
		t.Errorf("AddressCreate() IsPrimary = false, want true")
	}
}

func Test_AddressUpdate(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0004-0004-0004-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0004-0004-0004-000000000002")
	addrID := uuid.FromStringOrNil("ab1b2c3d-0004-0004-0004-000000000003")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	// Create the parent contact
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "Update",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Create address
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	a := &contact.Address{
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       contact.AddressTypeTel,
		Target:     "+155****4001",
		IsPrimary:  false,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Update is_primary
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressUpdate(ctx, addrID, map[string]any{"is_primary": true}); err != nil {
		t.Fatalf("AddressUpdate() error = %v", err)
	}

	// Verify
	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if !got.IsPrimary {
		t.Errorf("AddressUpdate() IsPrimary = false, want true")
	}
}

func Test_AddressDelete(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0005-0005-0005-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0005-0005-0005-000000000002")
	addrID := uuid.FromStringOrNil("ab1b2c3d-0005-0005-0005-000000000003")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	// Create the parent contact
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "Delete",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Create address
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	a := &contact.Address{
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       contact.AddressTypeTel,
		Target:     "+155****5001",
		IsPrimary:  false,
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Delete
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressDelete(ctx, addrID); err != nil {
		t.Fatalf("AddressDelete() error = %v", err)
	}

	// Verify deleted
	_, err := h.AddressGet(ctx, customerID, addrID)
	if err != ErrNotFound {
		t.Errorf("AddressGet() after delete: expected ErrNotFound, got: %v", err)
	}
}

