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

// Test_PhoneNumberUpdate tests updating a phone number
func Test_PhoneNumberUpdate(t *testing.T) {
	tests := []struct {
		name    string
		contact *contact.Contact
		phone   *contact.PhoneNumber
		update  map[string]any

		responseCurTime *time.Time
	}{
		{
			name: "update phone number type",
			contact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1"),
					CustomerID: uuid.FromStringOrNil("a2a2a2a2-a2a2-a2a2-a2a2-a2a2a2a2a2a2"),
				},
				FirstName: "Phone",
				LastName:  "Update",
				Source:    "manual",
			},
			phone: &contact.PhoneNumber{
				ID:         uuid.FromStringOrNil("a3a3a3a3-a3a3-a3a3-a3a3-a3a3a3a3a3a3"),
				CustomerID: uuid.FromStringOrNil("a2a2a2a2-a2a2-a2a2-a2a2-a2a2a2a2a2a2"),
				ContactID:  uuid.FromStringOrNil("a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1"),
				Number:     "+1-555-111-1111",
				NumberE164: "+15551111111",
				Type:       "mobile",
				IsPrimary:  false,
			},
			update: map[string]any{
				"type": "work",
			},

			responseCurTime: timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Create the contact first
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.ContactCreate(ctx, tt.contact); err != nil {
				t.Errorf("ContactCreate() error = %v", err)
			}

			// Create phone number
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.PhoneNumberCreate(ctx, tt.phone); err != nil {
				t.Errorf("PhoneNumberCreate() error = %v", err)
			}

			// Update phone number
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.PhoneNumberUpdate(ctx, tt.phone.ID, tt.update); err != nil {
				t.Errorf("PhoneNumberUpdate() error = %v", err)
			}

			// Verify the update
			res, err := h.PhoneNumberGet(ctx, tt.phone.ID)
			if err != nil {
				t.Errorf("PhoneNumberGet() error = %v", err)
			}

			if res.Type != tt.update["type"].(string) {
				t.Errorf("PhoneNumber Type = %v, want %v", res.Type, tt.update["type"])
			}
		})
	}
}

// Test_PhoneNumberResetPrimary tests resetting primary phone numbers
func Test_PhoneNumberResetPrimary(t *testing.T) {
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

	contactID := uuid.FromStringOrNil("a4a4a4a4-a4a4-a4a4-a4a4-a4a4a4a4a4a4")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("a5a5a5a5-a5a5-a5a5-a5a5-a5a5a5a5a5a5"),
		},
		FirstName: "Primary",
		LastName:  "Reset",
		Source:    "manual",
	}

	// Create contact
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Errorf("ContactCreate() error = %v", err)
	}

	// Create two phone numbers, both primary
	phone1 := &contact.PhoneNumber{
		ID:         uuid.FromStringOrNil("a6a6a6a6-a6a6-a6a6-a6a6-a6a6a6a6a6a6"),
		CustomerID: c.CustomerID,
		ContactID:  contactID,
		Number:     "+1-555-111-1111",
		NumberE164: "+15551111111",
		Type:       "mobile",
		IsPrimary:  true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.PhoneNumberCreate(ctx, phone1); err != nil {
		t.Errorf("PhoneNumberCreate() error = %v", err)
	}

	// Reset primary
	if err := h.PhoneNumberResetPrimary(ctx, contactID); err != nil {
		t.Errorf("PhoneNumberResetPrimary() error = %v", err)
	}

	// Verify primary is reset
	res, err := h.PhoneNumberGet(ctx, phone1.ID)
	if err != nil {
		t.Errorf("PhoneNumberGet() error = %v", err)
	}

	if res.IsPrimary {
		t.Error("PhoneNumber should not be primary after reset")
	}
}

// Test_EmailUpdate tests updating an email
func Test_EmailUpdate(t *testing.T) {
	tests := []struct {
		name    string
		contact *contact.Contact
		email   *contact.Email
		update  map[string]any

		responseCurTime *time.Time
	}{
		{
			name: "update email type",
			contact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a7a7a7a7-a7a7-a7a7-a7a7-a7a7a7a7a7a7"),
					CustomerID: uuid.FromStringOrNil("a8a8a8a8-a8a8-a8a8-a8a8-a8a8a8a8a8a8"),
				},
				FirstName: "Email",
				LastName:  "Update",
				Source:    "manual",
			},
			email: &contact.Email{
				ID:         uuid.FromStringOrNil("a9a9a9a9-a9a9-a9a9-a9a9-a9a9a9a9a9a9"),
				CustomerID: uuid.FromStringOrNil("a8a8a8a8-a8a8-a8a8-a8a8-a8a8a8a8a8a8"),
				ContactID:  uuid.FromStringOrNil("a7a7a7a7-a7a7-a7a7-a7a7-a7a7a7a7a7a7"),
				Address:    "test@example.com",
				Type:       "work",
				IsPrimary:  false,
			},
			update: map[string]any{
				"type": "personal",
			},

			responseCurTime: timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Create the contact first
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.ContactCreate(ctx, tt.contact); err != nil {
				t.Errorf("ContactCreate() error = %v", err)
			}

			// Create email
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.EmailCreate(ctx, tt.email); err != nil {
				t.Errorf("EmailCreate() error = %v", err)
			}

			// Update email
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.EmailUpdate(ctx, tt.email.ID, tt.update); err != nil {
				t.Errorf("EmailUpdate() error = %v", err)
			}

			// Verify the update
			res, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("EmailGet() error = %v", err)
			}

			if res.Type != tt.update["type"].(string) {
				t.Errorf("Email Type = %v, want %v", res.Type, tt.update["type"])
			}
		})
	}
}

// Test_EmailResetPrimary tests resetting primary emails
func Test_EmailResetPrimary(t *testing.T) {
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

	contactID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"),
		},
		FirstName: "Email",
		LastName:  "Reset",
		Source:    "manual",
	}

	// Create contact
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Errorf("ContactCreate() error = %v", err)
	}

	// Create email with primary
	email1 := &contact.Email{
		ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"),
		CustomerID: c.CustomerID,
		ContactID:  contactID,
		Address:    "primary@example.com",
		Type:       "work",
		IsPrimary:  true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.EmailCreate(ctx, email1); err != nil {
		t.Errorf("EmailCreate() error = %v", err)
	}

	// Reset primary
	if err := h.EmailResetPrimary(ctx, contactID); err != nil {
		t.Errorf("EmailResetPrimary() error = %v", err)
	}

	// Verify primary is reset
	res, err := h.EmailGet(ctx, email1.ID)
	if err != nil {
		t.Errorf("EmailGet() error = %v", err)
	}

	if res.IsPrimary {
		t.Error("Email should not be primary after reset")
	}
}

// Test_PhoneNumberGet_NotFound tests getting a non-existent phone number
func Test_PhoneNumberGet_NotFound(t *testing.T) {
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

	nonExistentID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1")

	_, err := h.PhoneNumberGet(ctx, nonExistentID)
	if err != ErrNotFound {
		t.Errorf("PhoneNumberGet() expected ErrNotFound, got: %v", err)
	}
}

// Test_EmailGet_NotFound tests getting a non-existent email
func Test_EmailGet_NotFound(t *testing.T) {
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

	nonExistentID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-ccccccccccc1")

	_, err := h.EmailGet(ctx, nonExistentID)
	if err != ErrNotFound {
		t.Errorf("EmailGet() expected ErrNotFound, got: %v", err)
	}
}

// Test_PhoneNumberUpdate_MultipleFields tests updating multiple phone number fields at once
func Test_PhoneNumberUpdate_MultipleFields(t *testing.T) {
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

	contactID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-ddddddddddd1")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-ddddddddddd2"),
		},
		FirstName: "Multi",
		LastName:  "Update",
		Source:    "manual",
	}

	// Create contact
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Errorf("ContactCreate() error = %v", err)
	}

	// Create phone
	phone := &contact.PhoneNumber{
		ID:         uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-ddddddddddd3"),
		CustomerID: c.CustomerID,
		ContactID:  contactID,
		Number:     "+1-555-111-1111",
		NumberE164: "+15551111111",
		Type:       "mobile",
		IsPrimary:  false,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.PhoneNumberCreate(ctx, phone); err != nil {
		t.Errorf("PhoneNumberCreate() error = %v", err)
	}

	// Update multiple fields
	updates := map[string]any{
		"type":       "work",
		"is_primary": true,
	}

	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.PhoneNumberUpdate(ctx, phone.ID, updates); err != nil {
		t.Errorf("PhoneNumberUpdate() error = %v", err)
	}

	// Verify updates
	res, err := h.PhoneNumberGet(ctx, phone.ID)
	if err != nil {
		t.Errorf("PhoneNumberGet() error = %v", err)
	}

	if res.Type != "work" {
		t.Errorf("PhoneNumber Type = %v, want work", res.Type)
	}
	if !res.IsPrimary {
		t.Error("PhoneNumber IsPrimary should be true")
	}
}

// Test_EmailUpdate_MultipleFields tests updating multiple email fields at once
func Test_EmailUpdate_MultipleFields(t *testing.T) {
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

	contactID := uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2"),
		},
		FirstName: "Multi",
		LastName:  "Email",
		Source:    "manual",
	}

	// Create contact
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.ContactCreate(ctx, c); err != nil {
		t.Errorf("ContactCreate() error = %v", err)
	}

	// Create email
	email := &contact.Email{
		ID:         uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee3"),
		CustomerID: c.CustomerID,
		ContactID:  contactID,
		Address:    "old@example.com",
		Type:       "work",
		IsPrimary:  false,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.EmailCreate(ctx, email); err != nil {
		t.Errorf("EmailCreate() error = %v", err)
	}

	// Update multiple fields
	updates := map[string]any{
		"address":    "new@example.com",
		"type":       "personal",
		"is_primary": true,
	}

	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.EmailUpdate(ctx, email.ID, updates); err != nil {
		t.Errorf("EmailUpdate() error = %v", err)
	}

	// Verify updates
	res, err := h.EmailGet(ctx, email.ID)
	if err != nil {
		t.Errorf("EmailGet() error = %v", err)
	}

	if res.Address != "new@example.com" {
		t.Errorf("Email Address = %v, want new@example.com", res.Address)
	}
	if res.Type != "personal" {
		t.Errorf("Email Type = %v, want personal", res.Type)
	}
	if !res.IsPrimary {
		t.Error("Email IsPrimary should be true")
	}
}
