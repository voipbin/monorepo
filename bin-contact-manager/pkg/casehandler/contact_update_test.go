package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_UpdateContact_Attach verifies a successful attach
// (contact_id != uuid.Nil): CaseUpdateContactID is called and
// case_contact_attributed is published via the plain PublishEvent
// primitive.
func Test_UpdateContact_Attach(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9801-9801-9801-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0801", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	ct := &contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
	}
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactSet(ctx, gomock.Any()).AnyTimes()
	if err := db.ContactCreate(ctx, ct); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	mockNotify.EXPECT().PublishEvent(ctx, "case_contact_attributed", map[string]uuid.UUID{
		"case_id":    caseID,
		"contact_id": contactID,
	}).Times(1)

	res, err := h.UpdateContact(ctx, customerID, caseID, contactID)
	if err != nil {
		t.Fatalf("UpdateContact() error = %v", err)
	}
	if res.ContactID == nil || *res.ContactID != contactID {
		t.Errorf("expected contact_id: %v, got: %v", contactID, res.ContactID)
	}
}

// Test_UpdateContact_Detach verifies a successful detach
// (contact_id == uuid.Nil): CaseClearContactID is called and
// case_contact_detached is published.
func Test_UpdateContact_Detach(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9802-9802-9802-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0802", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// Attach first, so the detach path has something real to clear.
	if err := db.CaseUpdateContactID(ctx, customerID, caseID, contactID); err != nil {
		t.Fatalf("CaseUpdateContactID() error = %v", err)
	}

	mockNotify.EXPECT().PublishEvent(ctx, "case_contact_detached", map[string]uuid.UUID{
		"case_id":    caseID,
		"contact_id": uuid.Nil,
	}).Times(1)

	res, err := h.UpdateContact(ctx, customerID, caseID, uuid.Nil)
	if err != nil {
		t.Fatalf("UpdateContact() error = %v", err)
	}
	if res.ContactID != nil {
		t.Errorf("expected nil contact_id after detach, got: %v", *res.ContactID)
	}
}

// Test_UpdateContact_CaseOwnershipFailure verifies that when the case
// does not belong to the given customer, UpdateContact rejects before
// any DB write or event publish happens.
func Test_UpdateContact_CaseOwnershipFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000003")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9803-9803-9803-000000000004")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	victimCase := &kase.Case{
		ID: caseID, CustomerID: victimCustomerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0803", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, victimCase); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// No PublishEvent EXPECT is set: the ownership check must reject
	// before any of that machinery runs.
	if _, err := h.UpdateContact(ctx, attackerCustomerID, caseID, contactID); err != dbhandler.ErrNotFound {
		t.Errorf("expected ErrNotFound (tenant isolation must hide existence), got: %v", err)
	}

	res, err := h.CaseGet(ctx, victimCustomerID, caseID)
	if err != nil {
		t.Fatalf("CaseGet() (victim) error = %v", err)
	}
	if res.ContactID != nil {
		t.Errorf("BUG: attacker's cross-tenant UpdateContact call modified the victim's case: %+v", res)
	}
}

// Test_UpdateContact_ContactNotFound verifies that attaching a
// non-existent contact is rejected and no event is published.
func Test_UpdateContact_ContactNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9807-9807-9807-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9807-9807-9807-000000000002")
	missingContactID := uuid.FromStringOrNil("f1b2c3d4-9807-9807-9807-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0807", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	// No PublishEvent EXPECT is set: the contact-not-found check must
	// reject before any event is published.
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound).AnyTimes()
	if _, err := h.UpdateContact(ctx, customerID, caseID, missingContactID); err == nil {
		t.Errorf("expected an error for a non-existent contact, got nil")
	}

	res, err := h.CaseGet(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseGet() error = %v", err)
	}
	if res.ContactID != nil {
		t.Errorf("BUG: UpdateContact wrote a contact_id despite ContactGet failing: %+v", res)
	}
}

// Test_UpdateContact_ContactCrossTenant verifies that attaching a
// contact belonging to a different customer is rejected and no
// update/event happens -- this is the cross-tenant guard preserved
// from VOIP-1252's round-1 review finding.
func Test_UpdateContact_ContactCrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000001")
	otherCustomerID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000003")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9805-9805-9805-000000000004")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0805", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	ct := &contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: otherCustomerID},
	}
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactSet(ctx, gomock.Any()).AnyTimes()
	if err := db.ContactCreate(ctx, ct); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// No PublishEvent EXPECT is set: the cross-tenant contact check
	// must reject before any event is published.
	if _, err := h.UpdateContact(ctx, customerID, caseID, contactID); err == nil {
		t.Errorf("expected an error for a cross-tenant contact, got nil")
	}

	res, err := h.CaseGet(ctx, customerID, caseID)
	if err != nil {
		t.Fatalf("CaseGet() error = %v", err)
	}
	if res.ContactID != nil {
		t.Errorf("BUG: UpdateContact attached a cross-tenant contact: %+v", res)
	}
}

// Test_UpdateContact_NeverUsesPublishWebhookEvent is the mandatory
// negative test mirroring casenote_isolation_test.go's pattern:
// case_contact_attributed/detached MUST be published via the plain
// notifyHandler.PublishEvent() primitive -- NEVER PublishWebhookEvent().
// This is an internal state-change/audit event, not a customer-facing
// webhook.
func Test_UpdateContact_NeverUsesPublishWebhookEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9806-9806-9806-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****0806", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &now, TMCreate: &now, TMUpdate: &now,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	ct := &contact.Contact{
		Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
	}
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound).AnyTimes()
	mockCache.EXPECT().ContactSet(ctx, gomock.Any()).AnyTimes()
	if err := db.ContactCreate(ctx, ct); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// The mandatory negative assertion: PublishEvent exactly once,
	// PublishWebhookEvent NEVER. gomock fails the test if
	// PublishWebhookEvent is called without a matching EXPECT (Times(0)
	// makes any call a hard failure, not just an unasserted no-op).
	mockNotify.EXPECT().PublishEvent(ctx, "case_contact_attributed", gomock.Any()).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	if _, err := h.UpdateContact(ctx, customerID, caseID, contactID); err != nil {
		t.Fatalf("UpdateContact() error = %v", err)
	}
}
