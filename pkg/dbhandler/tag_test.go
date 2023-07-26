package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/tag-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/tag-manager.git/pkg/cachehandler"
)

func Test_TagCreate(t *testing.T) {

	tests := []struct {
		name string
		tag  *tag.Tag

		responseCurTime string
		expectRes       *tag.Tag
	}{
		{
			name: "normal",
			tag: &tag.Tag{
				ID:     uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:   "name1",
				Detail: "detail1",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &tag.Tag{
				ID:       uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:     "name1",
				Detail:   "detail1",
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			if err := h.TagCreate(ctx, tt.tag); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TagGet(ctx, tt.tag.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			res, err := h.TagGet(ctx, tt.tag.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TagGets(t *testing.T) {

	tests := []struct {
		name string
		tags []*tag.Tag

		customerID uuid.UUID
		size       uint64

		responseCurTime string
		expectRes       []*tag.Tag
	}{
		{
			name: "normal",
			tags: []*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name1",
					Detail:     "detail1",
				},
				{
					ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name2",
					Detail:     "detail2",
				},
			},

			customerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
			size:       2,

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name1",
					Detail:     "detail1",
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name2",
					Detail:     "detail2",
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

			for _, u := range tt.tags {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TagSet(ctx, gomock.Any())
				if err := h.TagCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TagGets(ctx, tt.customerID, tt.size, utilhandler.TimeGetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TagSetBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		tagName string
		detail  string

		tags []*tag.Tag

		responseCurTime string
		expectRes       *tag.Tag
	}{
		{
			name: "test normal",

			id:      uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
			tagName: "name1",
			detail:  "detail1",

			tags: []*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("b7442490-7fe1-11ec-a66b-b7a03a06132f"),
					Name:       "name1",
					Detail:     "detail1",
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &tag.Tag{
				ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
				CustomerID: uuid.FromStringOrNil("b7442490-7fe1-11ec-a66b-b7a03a06132f"),
				Name:       "name1",
				Detail:     "detail1",
				TMCreate:   "2020-04-18 03:22:17.995000",
				TMUpdate:   "2020-04-18 03:22:17.995000",
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

			for _, u := range tt.tags {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TagSet(ctx, gomock.Any())
				if err := h.TagCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			err := h.TagSetBasicInfo(ctx, tt.id, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TagGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			res, err := h.TagGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TagDelete(t *testing.T) {

	tests := []struct {
		name string
		tag  *tag.Tag

		id uuid.UUID

		responseCurTime string
		expectRes       *tag.Tag
	}{
		{
			name: "normal",
			tag: &tag.Tag{
				ID:         uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),
				CustomerID: uuid.FromStringOrNil("dd805a3e-7fe1-11ec-b37d-134362dec03c"),
				Name:       "name1",
				Detail:     "detail1",
			},

			id: uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &tag.Tag{
				ID:         uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),
				CustomerID: uuid.FromStringOrNil("dd805a3e-7fe1-11ec-b37d-134362dec03c"),
				Name:       "name1",
				Detail:     "detail1",
				TMCreate:   "2020-04-18 03:22:17.995000",
				TMUpdate:   "2020-04-18 03:22:17.995000",
				TMDelete:   "2020-04-18 03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			if err := h.TagCreate(ctx, tt.tag); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			err := h.TagDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TagGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			res, err := h.TagGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
