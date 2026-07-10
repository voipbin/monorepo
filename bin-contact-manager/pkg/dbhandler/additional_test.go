package dbhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// Test_AddressUpdate_IsPrimary tests updating the is_primary field of an address
func Test_AddressUpdate_IsPrimary(t *testing.T) {
	tests := []struct {
		name    string
		contact *contact.Contact
		address *contact.Address
		update  map[string]any

		responseCurTime *time.Time
	}{
		{
			name: "update address is_primary",
			contact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1"),
					CustomerID: uuid.FromStringOrNil("a2a2a2a2-a2a2-a2a2-a2a2-a2a2a2a2a2a2"),
				},
				FirstName: "Address",
				LastName:  "Update",
				Source:    "manual",
			},
			address: &contact.Address{
				Address: commonaddress.Address{
					Type: contact.AddressTypeTel,
					Target: "+155****1111",
				},
				ID: uuid.FromStringOrNil("a3a3a3a3-a3a3-a3a3-a3a3-a3a3a3a3a3a3"),
				CustomerID: uuid.FromStringOrNil("a2a2a2a2-a2a2-a2a2-a2a2-a2a2a2a2a2a2"),
				ContactID: uuid.FromStringOrNil("a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1"),
				IsPrimary: false,
			},
			update: map[string]any{
				"is_primary": true,
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

			// Create address
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.AddressCreate(ctx, tt.address); err != nil {
				t.Errorf("AddressCreate() error = %v", err)
			}

			// Update address
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.AddressUpdate(ctx, tt.address.ID, tt.update); err != nil {
				t.Errorf("AddressUpdate() error = %v", err)
			}

			// Verify the update
			res, err := h.AddressGet(ctx, tt.address.CustomerID, tt.address.ID)
			if err != nil {
				t.Errorf("AddressGet() error = %v", err)
			}

			if res.IsPrimary != tt.update["is_primary"].(bool) {
				t.Errorf("Address IsPrimary = %v, want %v", res.IsPrimary, tt.update["is_primary"])
			}
		})
	}
}

// Test_AddressResetPrimary tests resetting primary addresses
func Test_AddressResetPrimary(t *testing.T) {
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
	customerID := uuid.FromStringOrNil("a5a5a5a5-a5a5-a5a5-a5a5-a5a5a5a5a5a5")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
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

	// Create primary address
	addr1 := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeTel,
			Target: "+155****1111",
		},
		ID: uuid.FromStringOrNil("a6a6a6a6-a6a6-a6a6-a6a6-a6a6a6a6a6a6"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, addr1); err != nil {
		t.Errorf("AddressCreate() error = %v", err)
	}

	// Reset primary
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressResetPrimary(ctx, contactID); err != nil {
		t.Errorf("AddressResetPrimary() error = %v", err)
	}

	// Verify primary is reset
	res, err := h.AddressGet(ctx, customerID, addr1.ID)
	if err != nil {
		t.Errorf("AddressGet() error = %v", err)
	}

	if res.IsPrimary {
		t.Error("Address should not be primary after reset")
	}
}

// Test_AddressUpdate_Target tests updating the target field of an address
func Test_AddressUpdate_Target(t *testing.T) {
	tests := []struct {
		name    string
		contact *contact.Contact
		address *contact.Address
		update  map[string]any

		responseCurTime *time.Time
	}{
		{
			name: "update address target and is_primary",
			contact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a7a7a7a7-a7a7-a7a7-a7a7-a7a7a7a7a7a7"),
					CustomerID: uuid.FromStringOrNil("a8a8a8a8-a8a8-a8a8-a8a8-a8a8a8a8a8a8"),
				},
				FirstName: "Email",
				LastName:  "Update",
				Source:    "manual",
			},
			address: &contact.Address{
				Address: commonaddress.Address{
					Type: contact.AddressTypeEmail,
					Target: "old@example.com",
				},
				ID: uuid.FromStringOrNil("a9a9a9a9-a9a9-a9a9-a9a9-a9a9a9a9a9a9"),
				CustomerID: uuid.FromStringOrNil("a8a8a8a8-a8a8-a8a8-a8a8-a8a8a8a8a8a8"),
				ContactID: uuid.FromStringOrNil("a7a7a7a7-a7a7-a7a7-a7a7-a7a7a7a7a7a7"),
				IsPrimary: false,
			},
			update: map[string]any{
				"is_primary": true,
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

			// Create address
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.AddressCreate(ctx, tt.address); err != nil {
				t.Errorf("AddressCreate() error = %v", err)
			}

			// Update address
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ContactSet(ctx, gomock.Any())
			if err := h.AddressUpdate(ctx, tt.address.ID, tt.update); err != nil {
				t.Errorf("AddressUpdate() error = %v", err)
			}

			// Verify the update
			res, err := h.AddressGet(ctx, tt.address.CustomerID, tt.address.ID)
			if err != nil {
				t.Errorf("AddressGet() error = %v", err)
			}

			if res.IsPrimary != tt.update["is_primary"].(bool) {
				t.Errorf("Address IsPrimary = %v, want %v", res.IsPrimary, tt.update["is_primary"])
			}
		})
	}
}

// Test_AddressResetPrimary_Email tests resetting primary email-type addresses
func Test_AddressResetPrimary_Email(t *testing.T) {
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
	customerID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
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

	// Create email address with primary
	addr1 := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeEmail,
			Target: "primary@example.com",
		},
		ID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, addr1); err != nil {
		t.Errorf("AddressCreate() error = %v", err)
	}

	// Reset primary
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressResetPrimary(ctx, contactID); err != nil {
		t.Errorf("AddressResetPrimary() error = %v", err)
	}

	// Verify primary is reset
	res, err := h.AddressGet(ctx, customerID, addr1.ID)
	if err != nil {
		t.Errorf("AddressGet() error = %v", err)
	}

	if res.IsPrimary {
		t.Error("Email address should not be primary after reset")
	}
}

// Test_AddressGet_NotFound tests getting a non-existent address (tel type)
func Test_AddressGet_NotFound_Tel(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb0")
	nonExistentID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1")

	_, err := h.AddressGet(ctx, customerID, nonExistentID)
	if err != ErrNotFound {
		t.Errorf("AddressGet() expected ErrNotFound, got: %v", err)
	}
}

// Test_AddressGet_NotFound_Email tests getting a non-existent address (email type)
func Test_AddressGet_NotFound_Email(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-ccccccccccc0")
	nonExistentID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-ccccccccccc1")

	_, err := h.AddressGet(ctx, customerID, nonExistentID)
	if err != ErrNotFound {
		t.Errorf("AddressGet() expected ErrNotFound, got: %v", err)
	}
}

// Test_AddressUpdate_MultipleFields tests updating multiple address fields at once
func Test_AddressUpdate_MultipleFields(t *testing.T) {
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
	customerID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-ddddddddddd2")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
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

	// Create address
	addr := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeTel,
			Target: "+155****1111",
		},
		ID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-ddddddddddd3"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: false,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, addr); err != nil {
		t.Errorf("AddressCreate() error = %v", err)
	}

	// Update multiple fields (target + is_primary)
	updates := map[string]any{
		"target":     "+155****9999",
		"is_primary": true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressUpdate(ctx, addr.ID, updates); err != nil {
		t.Errorf("AddressUpdate() error = %v", err)
	}

	// Verify updates
	res, err := h.AddressGet(ctx, customerID, addr.ID)
	if err != nil {
		t.Errorf("AddressGet() error = %v", err)
	}

	if !res.IsPrimary {
		t.Error("Address IsPrimary should be true")
	}
	if res.Target != "+155****9999" {
		t.Errorf("Address Target = %v, want +155****9999", res.Target)
	}
}

// Test_AddressUpdate_EmailTarget tests updating an email address's target field
func Test_AddressUpdate_EmailTarget(t *testing.T) {
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
	customerID := uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2")
	c := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
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

	// Create email address
	addr := &contact.Address{
		Address: commonaddress.Address{
			Type: contact.AddressTypeEmail,
			Target: "old@example.com",
		},
		ID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee3"),
		CustomerID: customerID,
		ContactID: contactID,
		IsPrimary: false,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressCreate(ctx, addr); err != nil {
		t.Errorf("AddressCreate() error = %v", err)
	}

	// Update multiple fields
	updates := map[string]any{
		"target":     "new@example.com",
		"is_primary": true,
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)))
	mockCache.EXPECT().ContactSet(ctx, gomock.Any())
	if err := h.AddressUpdate(ctx, addr.ID, updates); err != nil {
		t.Errorf("AddressUpdate() error = %v", err)
	}

	// Verify updates
	res, err := h.AddressGet(ctx, customerID, addr.ID)
	if err != nil {
		t.Errorf("AddressGet() error = %v", err)
	}

	if res.Target != "new@example.com" {
		t.Errorf("Address Target = %v, want new@example.com", res.Target)
	}

	if !res.IsPrimary {
		t.Error("Address IsPrimary should be true")
	}
}
