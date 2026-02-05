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
