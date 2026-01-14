package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

func Test_ProviderCreate(t *testing.T) {

	tests := []struct {
		name     string
		provider *provider.Provider

		responseCurTime string

		expectRes *provider.Provider
	}{
		{
			"empty item",
			&provider.Provider{
				ID: uuid.FromStringOrNil("55b05798-42d8-11ed-bf01-63eb9485e19d"),
			},

			"2020-04-18 03:22:17.995000",

			&provider.Provider{
				ID:          uuid.FromStringOrNil("55b05798-42d8-11ed-bf01-63eb9485e19d"),
				TechHeaders: map[string]string{},
				TMCreate:    "2020-04-18 03:22:17.995000",
				TMUpdate:    commondatabasehandler.DefaultTimeStamp,
				TMDelete:    commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			"all item",
			&provider.Provider{
				ID:          uuid.FromStringOrNil("bc376722-42d8-11ed-a4aa-93b57648abc4"),
				Type:        provider.TypeSIP,
				Hostname:    "test.com",
				TechPrefix:  "0011",
				TechPostfix: "1122",
				TechHeaders: map[string]string{
					"CUSTOMER_CODE": "11223344",
				},
				Name:   "provider name",
				Detail: "provider detail",
			},

			"2020-04-18 03:22:17.995000",

			&provider.Provider{
				ID:          uuid.FromStringOrNil("bc376722-42d8-11ed-a4aa-93b57648abc4"),
				Type:        provider.TypeSIP,
				Hostname:    "test.com",
				TechPrefix:  "0011",
				TechPostfix: "1122",
				TechHeaders: map[string]string{
					"CUSTOMER_CODE": "11223344",
				},
				Name:     "provider name",
				Detail:   "provider detail",
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			if err := h.ProviderCreate(ctx, tt.provider); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ProviderGet(ctx, tt.provider.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			res, err := h.ProviderGet(ctx, tt.provider.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderGets(t *testing.T) {

	tests := []struct {
		name string

		token     string
		limit     uint64
		providers []provider.Provider

		responseCurTime string

		expectRes []*provider.Provider
	}{
		{
			"normal",

			"2018-12-30 04:22:17.995000",
			10,
			[]provider.Provider{
				{
					ID:   uuid.FromStringOrNil("626446ac-42dd-11ed-863f-1f7b39fa9d61"),
					Name: "test1",
				},
				{
					ID:   uuid.FromStringOrNil("634cdf70-42dd-11ed-bd60-ebdb62f04c25"),
					Name: "test2",
				},
			},

			"2018-04-18 03:22:17.995000",

			[]*provider.Provider{
				{
					ID:          uuid.FromStringOrNil("634cdf70-42dd-11ed-bd60-ebdb62f04c25"),
					Name:        "test2",
					TechHeaders: map[string]string{},
					TMCreate:    "2018-04-18 03:22:17.995000",
					TMUpdate:    commondatabasehandler.DefaultTimeStamp,
					TMDelete:    commondatabasehandler.DefaultTimeStamp,
				},
				{
					ID:          uuid.FromStringOrNil("626446ac-42dd-11ed-863f-1f7b39fa9d61"),
					Name:        "test1",
					TechHeaders: map[string]string{},
					TMCreate:    "2018-04-18 03:22:17.995000",
					TMUpdate:    commondatabasehandler.DefaultTimeStamp,
					TMDelete:    commondatabasehandler.DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			for _, p := range tt.providers {
				mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
				if err := h.ProviderCreate(ctx, &p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			filters := map[provider.Field]any{}
			res, err := h.ProviderGets(ctx, tt.token, tt.limit, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < 1 {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", "bigger than 0", len(res))
			}
		})
	}
}

func Test_ProviderDelete(t *testing.T) {

	tests := []struct {
		name string

		provider *provider.Provider
	}{
		{
			"normal",

			&provider.Provider{
				ID:       uuid.FromStringOrNil("2396620a-432f-11ed-9c2e-37f76ce929df"),
				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			if err := h.ProviderCreate(ctx, tt.provider); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			if err := h.ProviderDelete(ctx, tt.provider.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ProviderGet(ctx, tt.provider.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().ProviderSet(ctx, gomock.Any()).Return(nil)
			res, err := h.ProviderGet(ctx, tt.provider.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == commondatabasehandler.DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}

func Test_ProviderUpdate(t *testing.T) {

	tests := []struct {
		name string

		provider *provider.Provider

		updateFields map[provider.Field]any

		responseCurTime string
		expectRes       *provider.Provider
	}{
		{
			"normal",

			&provider.Provider{
				ID:     uuid.FromStringOrNil("e8776eb6-432f-11ed-acde-b7089222dfd9"),
				Name:   "test name",
				Detail: "test detail",

				TMCreate: "2021-04-18 03:22:17.995000",
				TMUpdate: "2021-04-18 03:22:17.995000",
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},

			map[provider.Field]any{
				provider.FieldName:   "update name",
				provider.FieldDetail: "update detail",
			},

			"2021-04-18 03:22:17.995000",

			&provider.Provider{
				ID:          uuid.FromStringOrNil("e8776eb6-432f-11ed-acde-b7089222dfd9"),
				Name:        "update name",
				Detail:      "update detail",
				TechHeaders: map[string]string{},

				TMCreate: "2021-04-18 03:22:17.995000",
				TMUpdate: "2021-04-18 03:22:17.995000",
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			if err := h.ProviderCreate(ctx, tt.provider); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			if err := h.ProviderUpdate(ctx, tt.provider.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ProviderGet(ctx, tt.provider.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ProviderSet(ctx, gomock.Any())
			res, err := h.ProviderGet(ctx, tt.provider.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
