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

// Test_AddressCreate_Unresolved verifies AddressCreate writes a real SQL
// NULL for contact_id when a.ContactID == uuid.Nil (not the all-zero-byte
// uuid.Nil.Bytes()), and does not call contactUpdateToCache in that case.
func Test_AddressCreate_Unresolved(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0006-0006-0006-000000000001")
	addrID := uuid.FromStringOrNil("ab1b2c3d-0006-0006-0006-000000000002")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	// No mockCache.ContactSet expectation — contactUpdateToCache must NOT be
	// called for an unresolved (uuid.Nil) contact_id.
	mockUtil.EXPECT().TimeNow().Return(curTime)
	a := &contact.Address{
		ID:         addrID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
		Type:       contact.AddressTypeTel,
		Target:     "+155****6001",
	}
	if err := h.AddressCreate(ctx, a); err != nil {
		t.Fatalf("AddressCreate() error = %v", err)
	}

	// Verify the raw column is SQL NULL, not 16 zero bytes.
	var contactIDBytes []byte
	row := dbTest.QueryRow("SELECT contact_id FROM contact_addresses WHERE id = ?", addrID.Bytes())
	if err := row.Scan(&contactIDBytes); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v", err)
	}
	if contactIDBytes != nil {
		t.Errorf("contact_id column = %x, want SQL NULL (nil)", contactIDBytes)
	}

	// Verify the read path reports uuid.Nil.
	got, err := h.AddressGet(ctx, customerID, addrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != uuid.Nil {
		t.Errorf("AddressGet() ContactID = %v, want uuid.Nil", got.ContactID)
	}
}

// Test_AddressList_Unresolved verifies AddressList's unresolved filter only
// returns rows where contact_id IS NULL.
func Test_AddressList_Unresolved(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0007-0007-0007-000000000001")
	contactID := uuid.FromStringOrNil("ab1b2c3d-0007-0007-0007-000000000002")
	resolvedAddrID := uuid.FromStringOrNil("ab1b2c3d-0007-0007-0007-000000000003")
	unresolvedAddrID := uuid.FromStringOrNil("ab1b2c3d-0007-0007-0007-000000000004")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Address",
		LastName:  "ListUnresolved",
		Source:    "manual",
	}
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// resolved address
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, &contact.Address{
		ID:         resolvedAddrID,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       contact.AddressTypeTel,
		Target:     "+155****7001",
	}); err != nil {
		t.Fatalf("AddressCreate() resolved error = %v", err)
	}

	// unresolved address (no ContactSet expected)
	mockUtil.EXPECT().TimeNow().Return(curTime)
	if err := h.AddressCreate(ctx, &contact.Address{
		ID:         unresolvedAddrID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
		Type:       contact.AddressTypeTel,
		Target:     "+155****7002",
	}); err != nil {
		t.Fatalf("AddressCreate() unresolved error = %v", err)
	}

	res, err := h.AddressList(ctx, customerID, map[string]any{"unresolved": true}, "", 0)
	if err != nil {
		t.Fatalf("AddressList() error = %v", err)
	}

	foundUnresolved := false
	for _, a := range res {
		if a.ID == resolvedAddrID {
			t.Errorf("AddressList(unresolved=true) unexpectedly included resolved address %v", a.ID)
		}
		if a.ID == unresolvedAddrID {
			foundUnresolved = true
			if a.ContactID != uuid.Nil {
				t.Errorf("AddressList(unresolved=true) returned ContactID = %v, want uuid.Nil", a.ContactID)
			}
		}
	}
	if !foundUnresolved {
		t.Errorf("AddressList(unresolved=true) did not include unresolved address %v", unresolvedAddrID)
	}
}

// Test_AddressClaim covers the claim guard: successful claim of an
// unresolved address, conflict on an address already resolved to a
// different contact, and idempotent success on re-claiming the same
// contact.
func Test_AddressClaim(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("ab1b2c3d-0008-0008-0008-000000000001")
	contactID1 := uuid.FromStringOrNil("ab1b2c3d-0008-0008-0008-000000000002")
	contactID2 := uuid.FromStringOrNil("ab1b2c3d-0008-0008-0008-000000000003")
	unresolvedAddrID := uuid.FromStringOrNil("ab1b2c3d-0008-0008-0008-000000000004")
	resolvedAddrID := uuid.FromStringOrNil("ab1b2c3d-0008-0008-0008-000000000005")
	curTime := timePtr(time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC))

	// Two contacts to claim into / already claimed by.
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, &contact.Contact{
		Identity:  commonidentity.Identity{ID: contactID1, CustomerID: customerID},
		FirstName: "Claim", LastName: "One", Source: "manual",
	}); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, &contact.Contact{
		Identity:  commonidentity.Identity{ID: contactID2, CustomerID: customerID},
		FirstName: "Claim", LastName: "Two", Source: "manual",
	}); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// unresolved address
	mockUtil.EXPECT().TimeNow().Return(curTime)
	if err := h.AddressCreate(ctx, &contact.Address{
		ID:         unresolvedAddrID,
		CustomerID: customerID,
		ContactID:  uuid.Nil,
		Type:       contact.AddressTypeTel,
		Target:     "+155****8001",
	}); err != nil {
		t.Fatalf("AddressCreate() unresolved error = %v", err)
	}

	// already-resolved address (belongs to contactID1)
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, &contact.Address{
		ID:         resolvedAddrID,
		CustomerID: customerID,
		ContactID:  contactID1,
		Type:       contact.AddressTypeTel,
		Target:     "+155****8002",
	}); err != nil {
		t.Fatalf("AddressCreate() resolved error = %v", err)
	}

	// (a) claiming an unresolved address succeeds
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressClaim(ctx, customerID, unresolvedAddrID, contactID2); err != nil {
		t.Fatalf("AddressClaim() error = %v", err)
	}
	got, err := h.AddressGet(ctx, customerID, unresolvedAddrID)
	if err != nil {
		t.Fatalf("AddressGet() error = %v", err)
	}
	if got.ContactID != contactID2 {
		t.Errorf("AddressClaim() ContactID = %v, want %v", got.ContactID, contactID2)
	}

	// (b) claiming an address already resolved to a DIFFERENT contact -> ErrConflict
	if err := h.AddressClaim(ctx, customerID, resolvedAddrID, contactID2); err != ErrConflict {
		t.Errorf("AddressClaim() cross-contact claim: expected ErrConflict, got: %v", err)
	}

	// (c) re-claiming an address already resolved to the SAME contact -> idempotent success
	if err := h.AddressClaim(ctx, customerID, resolvedAddrID, contactID1); err != nil {
		t.Errorf("AddressClaim() idempotent same-contact claim: expected success, got: %v", err)
	}
}

