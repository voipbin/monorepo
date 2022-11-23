package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/util"
)

func TestTagCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name        string
		tg          *tag.Tag
		expectAgent *tag.Tag
	}{
		{
			"test normal",
			&tag.Tag{
				ID:       uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:     "name1",
				Detail:   "detail1",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&tag.Tag{
				ID:       uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:     "name1",
				Detail:   "detail1",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			if err := h.TagCreate(ctx, tt.tg); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TagGet(gomock.Any(), tt.tg.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			res, err := h.TagGet(ctx, tt.tg.ID)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectAgent, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAgent, res)
			}
		})
	}
}

func TestTagGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name       string
		customerID uuid.UUID
		tag        []*tag.Tag
		size       uint64
		expectRes  []*tag.Tag
	}{
		{
			"test normal",
			uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
			[]*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name1",
					Detail:     "detail1",
					TMCreate:   "2020-04-18T03:22:17.995000",
				},
				{
					ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name2",
					Detail:     "detail2",
					TMCreate:   "2020-04-18T03:22:17.994000",
				},
			},
			2,
			[]*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name1",
					Detail:     "detail1",
					TMCreate:   "2020-04-18T03:22:17.995000",
				},
				{
					ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					Name:       "name2",
					Detail:     "detail2",
					TMCreate:   "2020-04-18T03:22:17.994000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, u := range tt.tag {
				mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
				if err := h.TagCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TagGets(ctx, tt.customerID, tt.size, util.GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestTagSetBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		id      uuid.UUID
		tagName string
		detail  string

		tags []*tag.Tag

		expectRes *tag.Tag
	}{
		{
			"test normal",

			uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
			"name1",
			"detail1",

			[]*tag.Tag{
				{
					ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("b7442490-7fe1-11ec-a66b-b7a03a06132f"),
					Name:       "name1",
					Detail:     "detail1",
					TMCreate:   "",
					TMUpdate:   "",
					TMDelete:   "",
				},
			},

			&tag.Tag{
				ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
				CustomerID: uuid.FromStringOrNil("b7442490-7fe1-11ec-a66b-b7a03a06132f"),
				Name:       "name1",
				Detail:     "detail1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, u := range tt.tags {
				mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
				if err := h.TagCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			err := h.TagSetBasicInfo(ctx, tt.id, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TagGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			res, err := h.TagGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			res.TMCreate = ""
			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestTagDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		id uuid.UUID

		tag *tag.Tag

		expectRes *tag.Tag
	}{
		{
			"test normal",

			uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),

			&tag.Tag{
				ID:         uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),
				CustomerID: uuid.FromStringOrNil("dd805a3e-7fe1-11ec-b37d-134362dec03c"),
				Name:       "name1",
				Detail:     "detail1",
				TMCreate:   "",
				TMUpdate:   "",
				TMDelete:   "",
			},

			&tag.Tag{
				ID:         uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),
				CustomerID: uuid.FromStringOrNil("dd805a3e-7fe1-11ec-b37d-134362dec03c"),
				Name:       "name1",
				Detail:     "detail1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any())
			if err := h.TagCreate(ctx, tt.tag); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

			res.TMCreate = ""
			res.TMUpdate = ""
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
