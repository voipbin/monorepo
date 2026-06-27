package dbhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressMigration_CrossTypeSinglePrimary is a regression test for VOIP-1207.
//
// The unified contact_addresses table enforces one-primary-per-contact across
// BOTH tel and email types via UNIQUE(customer_id, primary_contact_uk). The
// handler path (AddPhoneNumber/AddEmail) calls PhoneNumberResetPrimary /
// EmailResetPrimary before inserting a new primary, and both delegate to
// addressResetPrimaryForContact, which demotes every existing primary of either
// type. This test exercises that path end-to-end against the sqlite test DB:
// after adding a primary phone and then a primary email, the phone must be
// demoted so the email is the sole primary.
func Test_AddressMigration_CrossTypeSinglePrimary(t *testing.T) {
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
	curTime := timePtr(time.Date(2021, 5, 19, 1, 2, 3, 0, time.UTC))

	contactID := uuid.FromStringOrNil("c1c1c1c1-0001-0001-0001-000000000001")
	customerID := uuid.FromStringOrNil("c1c1c1c1-0002-0002-0002-000000000002")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
		FirstName: "Cross",
		LastName:  "Primary",
		Source:    "manual",
	}

	// Create contact.
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Fatalf("ContactCreate() error = %v", err)
	}

	// Handler path for AddPhoneNumber(primary): reset existing primaries, then create.
	if err := h.PhoneNumberResetPrimary(ctx, contactID); err != nil {
		t.Fatalf("PhoneNumberResetPrimary() error = %v", err)
	}
	phone := &contact.PhoneNumber{
		ID:         uuid.FromStringOrNil("c1c1c1c1-0003-0003-0003-000000000003"),
		CustomerID: customerID,
		ContactID:  contactID,
		Number:     "+155****1111",
		IsPrimary:  true,
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.PhoneNumberCreate(ctx, phone); err != nil {
		t.Fatalf("PhoneNumberCreate() error = %v", err)
	}

	// Phone should be primary at this point.
	gotPhone, err := h.PhoneNumberGet(ctx, phone.ID)
	if err != nil {
		t.Fatalf("PhoneNumberGet() error = %v", err)
	}
	if !gotPhone.IsPrimary {
		t.Fatalf("expected the phone to be primary after creation")
	}

	// Handler path for AddEmail(primary): reset existing primaries (cross-type),
	// which must demote the phone, then create the primary email.
	if err := h.EmailResetPrimary(ctx, contactID); err != nil {
		t.Fatalf("EmailResetPrimary() error = %v", err)
	}
	email := &contact.Email{
		ID:         uuid.FromStringOrNil("c1c1c1c1-0004-0004-0004-000000000004"),
		CustomerID: customerID,
		ContactID:  contactID,
		Address:    "primary@example.com",
		IsPrimary:  true,
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.EmailCreate(ctx, email); err != nil {
		t.Fatalf("EmailCreate() error = %v", err)
	}

	// Assert cross-type single primary: the phone must be demoted, the email primary.
	gotPhone, err = h.PhoneNumberGet(ctx, phone.ID)
	if err != nil {
		t.Fatalf("PhoneNumberGet() error = %v", err)
	}
	if gotPhone.IsPrimary {
		t.Errorf("expected the phone to be demoted after adding a primary email, but it is still primary")
	}

	gotEmail, err := h.EmailGet(ctx, email.ID)
	if err != nil {
		t.Fatalf("EmailGet() error = %v", err)
	}
	if !gotEmail.IsPrimary {
		t.Errorf("expected the email to be primary, but it is not")
	}
}
