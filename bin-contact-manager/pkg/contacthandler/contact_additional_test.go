package contacthandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_UpdateAddress tests updating an address
func Test_UpdateAddress(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		addressID uuid.UUID
		fields    map[string]any

		responseContact *contact.Contact
		responseAddress *contact.Address
	}{
		{
			name: "normal update",

			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			addressID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			fields: map[string]any{
				"target":     "+1-555-999-8888",
				"is_primary": false,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
			},
			responseAddress: &contact.Address{
				Address: commonaddress.Address{
					Type:   contact.AddressTypeTel,
					Target: "+15559998888",
				},
				ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				CustomerID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				ContactID:  uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				IsPrimary:  false,
			},
		},
		{
			name: "set primary",

			contactID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			addressID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			fields: map[string]any{
				"is_primary": true,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				},
			},
			responseAddress: &contact.Address{
				Address: commonaddress.Address{
					Type:   contact.AddressTypeEmail,
					Target: "test@example.com",
				},
				ID:         uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				CustomerID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				ContactID:  uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				IsPrimary:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// First ContactGet to get customer_id
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			// AddressGet to verify existence + get type
			mockDB.EXPECT().AddressGet(ctx, tt.responseContact.CustomerID, tt.addressID).Return(tt.responseAddress, nil)

			// If setting primary, expect reset call
			if isPrimary, ok := tt.fields["is_primary"]; ok {
				if primary, isBool := isPrimary.(bool); isBool && primary {
					mockDB.EXPECT().AddressResetPrimary(ctx, tt.contactID).Return(nil)
				}
			}

			mockDB.EXPECT().AddressUpdate(ctx, tt.addressID, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.UpdateAddress(ctx, tt.contactID, tt.addressID, tt.fields)
			if err != nil {
				t.Errorf("UpdateAddress() error = %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("UpdateAddress() ID = %v, want %v", res.ID, tt.contactID)
			}
		})
	}
}

// Test_UpdateAddress_Error tests error cases
func Test_UpdateAddress_Error(t *testing.T) {
	tests := []struct {
		name      string
		contactID uuid.UUID
		addressID uuid.UUID
		fields    map[string]any
		setupMock func(*dbhandler.MockDBHandler, context.Context, uuid.UUID, uuid.UUID, map[string]any)
	}{
		{
			name:      "contact get error",
			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			addressID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			fields:    map[string]any{"is_primary": true},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("contact not found"))
			},
		},
		{
			name:      "address get error",
			contactID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			addressID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			fields:    map[string]any{"is_primary": true},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) {
				responseContact := &contact.Contact{
					Identity: commonidentity.Identity{
						ID:         contactID,
						CustomerID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
					},
				}
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
				mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(nil, fmt.Errorf("address not found"))
			},
		},
		{
			name:      "reset primary error",
			contactID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
			addressID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
			fields:    map[string]any{"is_primary": true},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) {
				responseContact := &contact.Contact{
					Identity: commonidentity.Identity{
						ID:         contactID,
						CustomerID: uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),
					},
				}
				responseAddress := &contact.Address{
					Address: commonaddress.Address{
						Type: contact.AddressTypeTel,
					},
					ID:        addressID,
					IsPrimary: false,
				}
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
				mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(responseAddress, nil)
				mockDB.EXPECT().AddressResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))
			},
		},
		{
			name:      "update error",
			contactID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			addressID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			fields:    map[string]any{"target": "+15551234567"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) {
				responseContact := &contact.Contact{
					Identity: commonidentity.Identity{
						ID:         contactID,
						CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
					},
				}
				responseAddress := &contact.Address{
					Address: commonaddress.Address{
						Type: contact.AddressTypeTel,
					},
					ID: addressID,
				}
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
				mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(responseAddress, nil)
				mockDB.EXPECT().AddressUpdate(ctx, addressID, gomock.Any()).Return(fmt.Errorf("update error"))
			},
		},
		{
			name:      "get after update error",
			contactID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			addressID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
			fields:    map[string]any{"target": "test@example.com"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) {
				responseContact := &contact.Contact{
					Identity: commonidentity.Identity{
						ID:         contactID,
						CustomerID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
					},
				}
				responseAddress := &contact.Address{
					Address: commonaddress.Address{
						Type: contact.AddressTypeEmail,
					},
					ID: addressID,
				}
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
				mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(responseAddress, nil)
				mockDB.EXPECT().AddressUpdate(ctx, addressID, gomock.Any()).Return(nil)
				mockDB.EXPECT().ContactGet(ctx, contactID).Return(nil, fmt.Errorf("get error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := contactHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			tt.setupMock(mockDB, ctx, tt.contactID, tt.addressID, tt.fields)

			_, err := h.UpdateAddress(ctx, tt.contactID, tt.addressID, tt.fields)
			if err == nil {
				t.Error("UpdateAddress() expected error")
			}
		})
	}
}

// Test_UpdateAddress_WithTarget tests target normalization on update
func Test_UpdateAddress_WithTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("40404040-4040-4040-4040-404040404040")
	addressID := uuid.FromStringOrNil("50505050-5050-5050-5050-505050505050")
	fields := map[string]any{
		"target": "+15551234567",
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("60606060-6060-6060-6060-606060606060"),
		},
	}
	responseAddress := &contact.Address{
		Address: commonaddress.Address{
			Type:   contact.AddressTypeTel,
			Target: "+15551234567",
		},
		ID:        addressID,
		ContactID: contactID,
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(responseAddress, nil)
	// Expect the update to include the normalized number
	mockDB.EXPECT().AddressUpdate(ctx, addressID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[string]any) error {
		if fields["target"] != "+15551234567" {
			return fmt.Errorf("expected normalized number to be +155****4567, got %v", fields["target"])
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.UpdateAddress(ctx, contactID, addressID, fields)
	if err != nil {
		t.Errorf("UpdateAddress() error = %v", err)
	}
}

// Test_UpdateAddress_WithEmailTarget tests email normalization on update
func Test_UpdateAddress_WithEmailTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("70707070-7070-7070-7070-707070707070")
	addressID := uuid.FromStringOrNil("80808080-8080-8080-8080-808080808080")
	fields := map[string]any{
		"target": "  TEST@EXAMPLE.COM  ",
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("90909090-9090-9090-9090-909090909090"),
		},
	}
	responseAddress := &contact.Address{
		Address: commonaddress.Address{
			Type:   contact.AddressTypeEmail,
			Target: "test@example.com",
		},
		ID:        addressID,
		ContactID: contactID,
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockDB.EXPECT().AddressGet(ctx, responseContact.CustomerID, addressID).Return(responseAddress, nil)
	// Expect the update to include normalized email
	mockDB.EXPECT().AddressUpdate(ctx, addressID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[string]any) error {
		if fields["target"] != "test@example.com" {
			return fmt.Errorf("expected normalized email to be test@example.com, got %v", fields["target"])
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.UpdateAddress(ctx, contactID, addressID, fields)
	if err != nil {
		t.Errorf("UpdateAddress() error = %v", err)
	}
}

// Test_AddAddress_ResetPrimaryError tests error handling for reset primary
func Test_AddAddress_ResetPrimaryError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := contactHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	contactID := uuid.FromStringOrNil("a0a0a0a0-a0a0-a0a0-a0a0-a0a0a0a0a0a0")
	addr := &contact.Address{
		Address: commonaddress.Address{
			Type:   contact.AddressTypeTel,
			Target: "+15554567890",
		},
		IsPrimary: true,
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("c0c0c0c0-c0c0-c0c0-c0c0-c0c0c0c0c0c0"))
	mockDB.EXPECT().AddressResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))

	_, err := h.AddAddress(ctx, contactID, addr)
	if err == nil {
		t.Error("AddAddress() expected error for reset primary failure")
	}
}
