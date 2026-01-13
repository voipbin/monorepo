package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AccesskeyCreate(t *testing.T) {

	tests := []struct {
		name      string
		accesskey *accesskey.Accesskey

		responseCurTime string
		expectRes       *accesskey.Accesskey
	}{
		{
			name: "all",

			accesskey: &accesskey.Accesskey{
				ID:         uuid.FromStringOrNil("64af434a-a757-11ef-bfa4-67b1b491a69b"),
				CustomerID: uuid.FromStringOrNil("64d2e7c8-a757-11ef-a0c0-1bd4aee0d0f2"),
				Name:       "test name",
				Detail:     "test detail",
				Token:      "test_token",
				TMExpire:   "2024-12-18 03:22:17.995000",
			},

			responseCurTime: "2024-04-18 03:22:17.995000",
			expectRes: &accesskey.Accesskey{
				ID:         uuid.FromStringOrNil("64af434a-a757-11ef-bfa4-67b1b491a69b"),
				CustomerID: uuid.FromStringOrNil("64d2e7c8-a757-11ef-a0c0-1bd4aee0d0f2"),
				Name:       "test name",
				Detail:     "test detail",
				Token:      "test_token",
				TMExpire:   "2024-12-18 03:22:17.995000",
				TMCreate:   "2024-04-18 03:22:17.995000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).AnyTimes()
			if err := h.AccesskeyCreate(ctx, tt.accesskey); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccesskeyGet(ctx, tt.accesskey.ID.Return(nil, fmt.Errorf(""))
			res, err := h.AccesskeyGet(ctx, tt.accesskey.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccesskeyDelete(t *testing.T) {

	tests := []struct {
		name      string
		accesskey *accesskey.Accesskey

		responseCurTime string
		expectRes       *accesskey.Accesskey
	}{
		{
			name: "normal",
			accesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("20a71b30-a759-11ef-b8fe-835b9e771719"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &accesskey.Accesskey{
				ID:       uuid.FromStringOrNil("20a71b30-a759-11ef-b8fe-835b9e771719"),
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: "2020-04-18 03:22:17.995000",
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccesskeyGet(ctx, tt.accesskey.ID.Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().AccesskeySet(ctx, gomock.Any().Return(nil).AnyTimes()
			if err := h.AccesskeyCreate(ctx, tt.accesskey); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			if err := h.AccesskeyDelete(ctx, tt.accesskey.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.AccesskeyGet(ctx, tt.accesskey.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccesskeyGets(t *testing.T) {

	tests := []struct {
		name       string
		accesskeys []*accesskey.Accesskey
		size       uint64
		token      string
		filters    map[accesskey.Field]any

		responseCurTime string
		expectRes       []*accesskey.Accesskey
	}{
		{
			name: "gets by customer_id",
			accesskeys: []*accesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("6b3fd5ba-a759-11ef-acd6-6f1a8cacd51f"),
					CustomerID: uuid.FromStringOrNil("6b9a880c-a759-11ef-93e6-f757c578bc3b"),
				},
				{
					ID:         uuid.FromStringOrNil("6b6f4818-a759-11ef-b6cc-0b5bb0dbad8a"),
					CustomerID: uuid.FromStringOrNil("6b9a880c-a759-11ef-93e6-f757c578bc3b"),
				},
			},
			size:  2,
			token: "2020-04-18 03:22:17.995001",
			filters: map[accesskey.Field]any{
				accesskey.FieldDeleted: false,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*accesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("6b3fd5ba-a759-11ef-acd6-6f1a8cacd51f"),
					CustomerID: uuid.FromStringOrNil("6b9a880c-a759-11ef-93e6-f757c578bc3b"),
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("6b6f4818-a759-11ef-b6cc-0b5bb0dbad8a"),
					CustomerID: uuid.FromStringOrNil("6b9a880c-a759-11ef-93e6-f757c578bc3b"),
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			name: "gets by token",
			accesskeys: []*accesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("cfd12b46-ab0f-11ef-a45f-ebb9ad8f8a2c"),
					CustomerID: uuid.FromStringOrNil("d03eb274-ab0f-11ef-aa02-d771ad6ee1b9"),
					Token:      "d07da3da-ab0f-11ef-8826-4f93ce3ceaa5",
				},
				{
					ID:         uuid.FromStringOrNil("d05cb4fe-ab0f-11ef-9a9c-57570390a427"),
					CustomerID: uuid.FromStringOrNil("d03eb274-ab0f-11ef-aa02-d771ad6ee1b9"),
					Token:      "d09df996-ab0f-11ef-862c-e3a5ac697296",
				},
			},
			size:  2,
			token: "2020-04-18 03:22:17.995001",
			filters: map[accesskey.Field]any{
				accesskey.FieldDeleted: false,
				accesskey.FieldToken:   "d09df996-ab0f-11ef-862c-e3a5ac697296",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*accesskey.Accesskey{
				{
					ID:         uuid.FromStringOrNil("d05cb4fe-ab0f-11ef-9a9c-57570390a427"),
					CustomerID: uuid.FromStringOrNil("d03eb274-ab0f-11ef-aa02-d771ad6ee1b9"),
					Token:      "d09df996-ab0f-11ef-862c-e3a5ac697296",
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, u := range tt.accesskeys {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().AccesskeySet(ctx, gomock.Any())
				if err := h.AccesskeyCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.AccesskeyGets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_AccesskeyUpdate(t *testing.T) {

	tests := []struct {
		name      string
		accesskey *accesskey.Accesskey

		updateFields map[accesskey.Field]any

		responseCurTime string
		expectRes       *accesskey.Accesskey
	}{
		{
			name: "all",
			accesskey: &accesskey.Accesskey{
				ID:     uuid.FromStringOrNil("b0935838-a75b-11ef-bb3b-e72be3c75a94"),
				Name:   "test4",
				Detail: "detail4",
			},

			updateFields: map[accesskey.Field]any{
				accesskey.FieldName:   "update name",
				accesskey.FieldDetail: "update detail",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &accesskey.Accesskey{
				ID:       uuid.FromStringOrNil("b0935838-a75b-11ef-bb3b-e72be3c75a94"),
				Name:     "update name",
				Detail:   "update detail",
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "empty",
			accesskey: &accesskey.Accesskey{
				ID:     uuid.FromStringOrNil("b12c9a5c-a75b-11ef-bc1a-97774b43f8cd"),
				Name:   "test4",
				Detail: "detail4",
			},

			updateFields: map[accesskey.Field]any{
				accesskey.FieldName:   "",
				accesskey.FieldDetail: "",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &accesskey.Accesskey{
				ID:       uuid.FromStringOrNil("b12c9a5c-a75b-11ef-bc1a-97774b43f8cd"),
				Name:     "",
				Detail:   "",
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccesskeySet(ctx, gomock.Any().Return(nil)
			if err := h.AccesskeyCreate(ctx, tt.accesskey); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().AccesskeySet(ctx, gomock.Any().Return(nil)
			if err := h.AccesskeyUpdate(ctx, tt.accesskey.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccesskeyGet(ctx, gomock.Any().Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccesskeySet(ctx, gomock.Any().Return(nil)
			res, err := h.AccesskeyGet(ctx, tt.accesskey.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
