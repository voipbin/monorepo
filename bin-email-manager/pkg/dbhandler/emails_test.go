package dbhandler

import (
	"context"
	"fmt"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func Test_EmailCreate(t *testing.T) {

	tests := []struct {
		name  string
		email *email.Email

		responseCurTime *time.Time

		expectResCreate *email.Email
	}{
		{
			name: "normal",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4077be56-ffe2-11ef-b6ae-37bfd020683e"),
					CustomerID: uuid.FromStringOrNil("40cd3a16-ffe2-11ef-9218-b3c1ca232b13"),
				},

				ActiveflowID: uuid.FromStringOrNil("f1d1d566-002c-11f0-9d12-17bdd7d9190b"),

				ProviderType:        email.ProviderTypeSendgrid,
				ProviderReferenceID: "40fb0b80-ffe2-11ef-b114-f79bfa4b4ab9",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "test@voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeEmail,
						Target: "test1@voipbin.net",
					},
				},
				Status:  email.StatusProcessed,
				Subject: "test subject",
				Content: "test content",

				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("74401462-ffe3-11ef-9aa6-ef244c549f23"),
					},
				},
			},

			responseCurTime: timePtr(time.Date(2025, 3, 13, 3, 22, 17, 995000000, time.UTC)),

			expectResCreate: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4077be56-ffe2-11ef-b6ae-37bfd020683e"),
					CustomerID: uuid.FromStringOrNil("40cd3a16-ffe2-11ef-9218-b3c1ca232b13"),
				},

				ActiveflowID: uuid.FromStringOrNil("f1d1d566-002c-11f0-9d12-17bdd7d9190b"),

				ProviderType:        email.ProviderTypeSendgrid,
				ProviderReferenceID: "40fb0b80-ffe2-11ef-b114-f79bfa4b4ab9",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "test@voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeEmail,
						Target: "test1@voipbin.net",
					},
				},
				Status:  email.StatusProcessed,
				Subject: "test subject",
				Content: "test content",

				Attachments: []email.Attachment{
					{
						ReferenceType: email.AttachmentReferenceTypeRecording,
						ReferenceID:   uuid.FromStringOrNil("74401462-ffe3-11ef-9aa6-ef244c549f23"),
					},
				},

				TMCreate: timePtr(time.Date(2025, 3, 13, 3, 22, 17, 995000000, time.UTC)),
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any())
			if errCreate := h.EmailCreate(ctx, tt.email); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockCache.EXPECT().EmailGet(gomock.Any(), tt.email.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			resCreate, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectResCreate, resCreate) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.email, resCreate)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			if errUpdate := h.EmailUpdateStatus(ctx, tt.email.ID, email.StatusDelivered); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
			}

			mockCache.EXPECT().EmailGet(gomock.Any(), tt.email.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			resUpdate, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			expectResUpdate := tt.expectResCreate
			expectResUpdate.Status = email.StatusDelivered
			expectResUpdate.TMUpdate = tt.responseCurTime

			if !reflect.DeepEqual(expectResUpdate, resUpdate) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.email, resUpdate)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			if errUpdate := h.EmailDelete(ctx, tt.email.ID); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
			}

			mockCache.EXPECT().EmailGet(gomock.Any(), tt.email.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			resDelete, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			expectResDelete := tt.expectResCreate
			expectResDelete.TMDelete = tt.responseCurTime

			if !reflect.DeepEqual(expectResDelete, resDelete) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.email, resUpdate)
			}

		})
	}
}

func Test_EmailList(t *testing.T) {

	tests := []struct {
		name   string
		emails []email.Email

		size    uint64
		filters map[email.Field]any

		expectRes []*email.Email
	}{
		{
			name: "normal",
			emails: []email.Email{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b79dec8a-ffe7-11ef-a9ee-e3856d9170c1"),
						CustomerID: uuid.FromStringOrNil("b7cff8ba-ffe7-11ef-ab26-737ee1b185a8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b7f922da-ffe7-11ef-adec-9b348249b70d"),
						CustomerID: uuid.FromStringOrNil("b7cff8ba-ffe7-11ef-ab26-737ee1b185a8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("0782db16-ffe8-11ef-9a8a-2fc48f725c3f"),
						CustomerID: uuid.FromStringOrNil("b8429a78-ffe7-11ef-bf43-f72206e52204"),
					},
				},
			},

			size: 10,
			filters: map[email.Field]any{
				email.FieldCustomerID: uuid.FromStringOrNil("b7cff8ba-ffe7-11ef-ab26-737ee1b185a8"),
				email.FieldDeleted:    false,
			},

			expectRes: []*email.Email{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b7f922da-ffe7-11ef-adec-9b348249b70d"),
						CustomerID: uuid.FromStringOrNil("b7cff8ba-ffe7-11ef-ab26-737ee1b185a8"),
					},
					Source:       &commonaddress.Address{},
					Destinations: []commonaddress.Address{},
					Attachments:  []email.Attachment{},
					TMUpdate:     nil,
					TMDelete:     nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b79dec8a-ffe7-11ef-a9ee-e3856d9170c1"),
						CustomerID: uuid.FromStringOrNil("b7cff8ba-ffe7-11ef-ab26-737ee1b185a8"),
					},
					Source:       &commonaddress.Address{},
					Destinations: []commonaddress.Address{},
					Attachments:  []email.Attachment{},
					TMUpdate:     nil,
					TMDelete:     nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, e := range tt.emails {
				mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
				mockCache.EXPECT().EmailSet(ctx, gomock.Any())
				if err := h.EmailCreate(ctx, &e); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.EmailList(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, f := range res {
				f.TMCreate = nil
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_EmailUpdateProviderReferenceID(t *testing.T) {

	tests := []struct {
		name  string
		email *email.Email

		id                  uuid.UUID
		providerReferenceID string

		responseCurTime *time.Time

		expectRes *email.Email
	}{
		{
			name: "normal",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83bf937c-0010-11f0-98ab-f7954e7eb6d8"),
				},
			},

			id:                  uuid.FromStringOrNil("83bf937c-0010-11f0-98ab-f7954e7eb6d8"),
			providerReferenceID: "8409255a-0010-11f0-8fe6-c7156be65533",

			responseCurTime: timePtr(time.Date(2025, 3, 13, 3, 22, 17, 995000000, time.UTC)),

			expectRes: &email.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("83bf937c-0010-11f0-98ab-f7954e7eb6d8"),
				},
				ProviderReferenceID: "8409255a-0010-11f0-8fe6-c7156be65533",
				Source:              &commonaddress.Address{},
				Destinations:        []commonaddress.Address{},
				Attachments:         []email.Attachment{},

				TMCreate: timePtr(time.Date(2025, 3, 13, 3, 22, 17, 995000000, time.UTC)),
				TMUpdate: timePtr(time.Date(2025, 3, 13, 3, 22, 17, 995000000, time.UTC)),
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().EmailSet(ctx, gomock.Any())
			if err := h.EmailCreate(ctx, tt.email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			if errUpdate := h.EmailUpdateProviderReferenceID(ctx, tt.id, tt.providerReferenceID); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
			}

			mockCache.EXPECT().EmailGet(gomock.Any(), tt.email.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.EmailGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailGet_NotFound(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			name: "returns_error_when_not_found",
			id:   uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().EmailGet(ctx, tt.id).Return(nil, fmt.Errorf("not found"))

			_, err := h.EmailGet(ctx, tt.id)
			if err == nil {
				t.Errorf("Expected error when email not found")
			}
		})
	}
}

func Test_EmailList_WithEmptyResult(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		size    uint64
		filters map[email.Field]any
	}{
		{
			name:  "returns_empty_or_small_list_when_filtered",
			token: "",
			size:  10,
			filters: map[email.Field]any{
				email.FieldCustomerID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
				email.FieldDeleted:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())

			res, err := h.EmailList(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res == nil {
				t.Errorf("Expected non-nil result")
			}

			// Just verify the list is returned, don't check the exact count
			// since other tests might have created data
		})
	}
}

func Test_EmailUpdate_WithMultipleFields(t *testing.T) {
	tests := []struct {
		name   string
		email  *email.Email
		fields map[email.Field]any
	}{
		{
			name: "updates_multiple_fields",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				Status:  email.StatusInitiated,
				Subject: "Original Subject",
			},
			fields: map[email.Field]any{
				email.FieldStatus:  email.StatusDelivered,
				email.FieldSubject: "Updated Subject",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			// Create the email first
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(ctx, gomock.Any())
			if err := h.EmailCreate(ctx, tt.email); err != nil {
				t.Fatalf("Failed to create email: %v", err)
			}

			// Update the email
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.EmailUpdate(ctx, tt.email.ID, tt.fields); err != nil {
				t.Errorf("Unexpected error from EmailUpdate: %v", err)
			}

			// Verify the update
			mockCache.EXPECT().EmailGet(ctx, tt.email.ID).Return(nil, fmt.Errorf("not in cache"))
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Fatalf("Failed to get email: %v", err)
			}

			if res.Status != email.StatusDelivered {
				t.Errorf("Wrong Status. expect: %s, got: %s", email.StatusDelivered, res.Status)
			}
			if res.Subject != "Updated Subject" {
				t.Errorf("Wrong Subject. expect: Updated Subject, got: %s", res.Subject)
			}
		})
	}
}

func Test_NewHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates_new_handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			if h == nil {
				t.Errorf("Expected non-nil handler")
			}
		})
	}
}

func Test_EmailGetFromCache_Success(t *testing.T) {
	tests := []struct {
		name  string
		email *email.Email
	}{
		{
			name: "gets_email_from_cache",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				},
				Status: email.StatusDelivered,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			// Return the email from cache
			mockCache.EXPECT().EmailGet(ctx, tt.email.ID).Return(tt.email, nil)

			res, err := h.EmailGet(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.ID != tt.email.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.email.ID, res.ID)
			}
		})
	}
}

func Test_EmailCreate_CacheFails(t *testing.T) {
	tests := []struct {
		name  string
		email *email.Email
	}{
		{
			name: "succeeds_even_when_cache_fails",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
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
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))

			// Should succeed even if cache fails
			err := h.EmailCreate(ctx, tt.email)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func Test_EmailUpdate_CacheFails(t *testing.T) {
	tests := []struct {
		name   string
		email  *email.Email
		fields map[email.Field]any
	}{
		{
			name: "succeeds_even_when_cache_update_fails",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
					CustomerID: uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888"),
				},
				Status: email.StatusInitiated,
			},
			fields: map[email.Field]any{
				email.FieldStatus: email.StatusDelivered,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			// Create the email
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(ctx, gomock.Any())
			if err := h.EmailCreate(ctx, tt.email); err != nil {
				t.Fatalf("Failed to create email: %v", err)
			}

			// Update with cache failing
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache error"))

			err := h.EmailUpdate(ctx, tt.email.ID, tt.fields)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func Test_EmailDelete_CacheFails(t *testing.T) {
	tests := []struct {
		name  string
		email *email.Email
	}{
		{
			name: "succeeds_even_when_cache_fails_on_delete",
			email: &email.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			// Create the email
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(ctx, gomock.Any())
			if err := h.EmailCreate(ctx, tt.email); err != nil {
				t.Fatalf("Failed to create email: %v", err)
			}

			// Delete with cache failing
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().EmailSet(gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache error"))

			err := h.EmailDelete(ctx, tt.email.ID)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func Test_EmailList_WithToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		size  uint64
	}{
		{
			name:  "lists_emails_with_custom_token",
			token: "2025-01-01 00:00:00.000000",
			size:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			res, err := h.EmailList(ctx, tt.token, tt.size, map[email.Field]any{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res == nil {
				t.Errorf("Expected non-nil result")
			}
		})
	}
}
