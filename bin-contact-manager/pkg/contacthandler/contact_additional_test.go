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

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_UpdatePhoneNumber tests updating a phone number
func Test_UpdatePhoneNumber(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		phoneID   uuid.UUID
		fields    map[string]any

		responseContact *contact.Contact
	}{
		{
			name: "normal update",

			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			phoneID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			fields: map[string]any{
				"number":     "+1-555-999-8888",
				"type":       "work",
				"is_primary": false,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
			},
		},
		{
			name: "set primary",

			contactID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			phoneID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			fields: map[string]any{
				"is_primary": true,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
					CustomerID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				},
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

			// If setting primary, expect reset call
			if isPrimary, ok := tt.fields["is_primary"]; ok {
				if primary, isBool := isPrimary.(bool); isBool && primary {
					mockDB.EXPECT().PhoneNumberResetPrimary(ctx, tt.contactID).Return(nil)
				}
			}

			mockDB.EXPECT().PhoneNumberUpdate(ctx, tt.phoneID, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.UpdatePhoneNumber(ctx, tt.contactID, tt.phoneID, tt.fields)
			if err != nil {
				t.Errorf("UpdatePhoneNumber() error = %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("UpdatePhoneNumber() ID = %v, want %v", res.ID, tt.contactID)
			}
		})
	}
}

// Test_UpdatePhoneNumber_Error tests error cases
func Test_UpdatePhoneNumber_Error(t *testing.T) {
	tests := []struct {
		name      string
		contactID uuid.UUID
		phoneID   uuid.UUID
		fields    map[string]any
		setupMock func(*dbhandler.MockDBHandler, context.Context, uuid.UUID, uuid.UUID, map[string]any)
	}{
		{
			name:      "reset primary error",
			contactID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			phoneID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			fields:    map[string]any{"is_primary": true},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, phoneID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().PhoneNumberResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))
			},
		},
		{
			name:      "update error",
			contactID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			phoneID:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			fields:    map[string]any{"number": "+1-555-123-4567"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, phoneID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().PhoneNumberUpdate(ctx, phoneID, gomock.Any()).Return(fmt.Errorf("update error"))
			},
		},
		{
			name:      "get after update error",
			contactID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			phoneID:   uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
			fields:    map[string]any{"number": "+1-555-123-4567"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, phoneID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().PhoneNumberUpdate(ctx, phoneID, gomock.Any()).Return(nil)
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

			tt.setupMock(mockDB, ctx, tt.contactID, tt.phoneID, tt.fields)

			_, err := h.UpdatePhoneNumber(ctx, tt.contactID, tt.phoneID, tt.fields)
			if err == nil {
				t.Error("UpdatePhoneNumber() expected error")
			}
		})
	}
}

// Test_UpdateEmail tests updating an email
func Test_UpdateEmail(t *testing.T) {
	tests := []struct {
		name string

		contactID uuid.UUID
		emailID   uuid.UUID
		fields    map[string]any

		responseContact *contact.Contact
	}{
		{
			name: "normal update",

			contactID: uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
			emailID:   uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),
			fields: map[string]any{
				"address":    "updated@example.com",
				"type":       "personal",
				"is_primary": false,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
					CustomerID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
				},
			},
		},
		{
			name: "set primary",

			contactID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			emailID:   uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			fields: map[string]any{
				"is_primary": true,
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
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

			// If setting primary, expect reset call
			if isPrimary, ok := tt.fields["is_primary"]; ok {
				if primary, isBool := isPrimary.(bool); isBool && primary {
					mockDB.EXPECT().EmailResetPrimary(ctx, tt.contactID).Return(nil)
				}
			}

			mockDB.EXPECT().EmailUpdate(ctx, tt.emailID, gomock.Any()).Return(nil)
			mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

			res, err := h.UpdateEmail(ctx, tt.contactID, tt.emailID, tt.fields)
			if err != nil {
				t.Errorf("UpdateEmail() error = %v", err)
			}

			if res.ID != tt.contactID {
				t.Errorf("UpdateEmail() ID = %v, want %v", res.ID, tt.contactID)
			}
		})
	}
}

// Test_UpdateEmail_Error tests error cases
func Test_UpdateEmail_Error(t *testing.T) {
	tests := []struct {
		name      string
		contactID uuid.UUID
		emailID   uuid.UUID
		fields    map[string]any
		setupMock func(*dbhandler.MockDBHandler, context.Context, uuid.UUID, uuid.UUID, map[string]any)
	}{
		{
			name:      "reset primary error",
			contactID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
			emailID:   uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
			fields:    map[string]any{"is_primary": true},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, emailID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().EmailResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))
			},
		},
		{
			name:      "update error",
			contactID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
			emailID:   uuid.FromStringOrNil("10101010-1010-1010-1010-101010101010"),
			fields:    map[string]any{"address": "test@example.com"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, emailID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().EmailUpdate(ctx, emailID, gomock.Any()).Return(fmt.Errorf("update error"))
			},
		},
		{
			name:      "get after update error",
			contactID: uuid.FromStringOrNil("20202020-2020-2020-2020-202020202020"),
			emailID:   uuid.FromStringOrNil("30303030-3030-3030-3030-303030303030"),
			fields:    map[string]any{"address": "test@example.com"},
			setupMock: func(mockDB *dbhandler.MockDBHandler, ctx context.Context, contactID, emailID uuid.UUID, fields map[string]any) {
				mockDB.EXPECT().EmailUpdate(ctx, emailID, gomock.Any()).Return(nil)
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

			tt.setupMock(mockDB, ctx, tt.contactID, tt.emailID, tt.fields)

			_, err := h.UpdateEmail(ctx, tt.contactID, tt.emailID, tt.fields)
			if err == nil {
				t.Error("UpdateEmail() expected error")
			}
		})
	}
}

// Test_UpdatePhoneNumber_WithNumber tests number_e164 derivation
func Test_UpdatePhoneNumber_WithNumber(t *testing.T) {
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
	phoneID := uuid.FromStringOrNil("50505050-5050-5050-5050-505050505050")
	fields := map[string]any{
		"number": "+1-555-123-4567",
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("60606060-6060-6060-6060-606060606060"),
		},
	}

	// Expect the update to include derived number_e164
	mockDB.EXPECT().PhoneNumberUpdate(ctx, phoneID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[string]any) error {
		if fields["number_e164"] != "+15551234567" {
			return fmt.Errorf("expected derived number_e164 to be +15551234567, got %v", fields["number_e164"])
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.UpdatePhoneNumber(ctx, contactID, phoneID, fields)
	if err != nil {
		t.Errorf("UpdatePhoneNumber() error = %v", err)
	}
}

// Test_UpdateEmail_WithAddress tests email normalization
func Test_UpdateEmail_WithAddress(t *testing.T) {
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
	emailID := uuid.FromStringOrNil("80808080-8080-8080-8080-808080808080")
	fields := map[string]any{
		"address": "  TEST@EXAMPLE.COM  ",
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("90909090-9090-9090-9090-909090909090"),
		},
	}

	// Expect the update to include normalized email
	mockDB.EXPECT().EmailUpdate(ctx, emailID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[string]any) error {
		if fields["address"] != "test@example.com" {
			return fmt.Errorf("expected normalized email to be test@example.com, got %v", fields["address"])
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactUpdated, gomock.Any())

	_, err := h.UpdateEmail(ctx, contactID, emailID, fields)
	if err != nil {
		t.Errorf("UpdateEmail() error = %v", err)
	}
}

// Test_AddPhoneNumber_ResetPrimaryError tests error handling for reset primary
func Test_AddPhoneNumber_ResetPrimaryError(t *testing.T) {
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
	phone := &contact.PhoneNumber{
		Number:     "+1-555-123-4567",
		NumberE164: "+15551234567",
		Type:       "mobile",
		IsPrimary:  true,
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("b0b0b0b0-b0b0-b0b0-b0b0-b0b0b0b0b0b0"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("c0c0c0c0-c0c0-c0c0-c0c0-c0c0c0c0c0c0"))
	mockDB.EXPECT().PhoneNumberResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))

	_, err := h.AddPhoneNumber(ctx, contactID, phone)
	if err == nil {
		t.Error("AddPhoneNumber() expected error for reset primary failure")
	}
}

// Test_AddEmail_ResetPrimaryError tests error handling for reset primary
func Test_AddEmail_ResetPrimaryError(t *testing.T) {
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

	contactID := uuid.FromStringOrNil("d0d0d0d0-d0d0-d0d0-d0d0-d0d0d0d0d0d0")
	email := &contact.Email{
		Address:   "test@example.com",
		Type:      "work",
		IsPrimary: true,
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: uuid.FromStringOrNil("e0e0e0e0-e0e0-e0e0-e0e0-e0e0e0e0e0e0"),
		},
	}

	mockDB.EXPECT().ContactGet(ctx, contactID).Return(responseContact, nil)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("f0f0f0f0-f0f0f0-f0f0-f0f0-f0f0f0f0f0f0"))
	mockDB.EXPECT().EmailResetPrimary(ctx, contactID).Return(fmt.Errorf("reset error"))

	_, err := h.AddEmail(ctx, contactID, email)
	if err == nil {
		t.Error("AddEmail() expected error for reset primary failure")
	}
}
