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

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

func Test_RouteCreate(t *testing.T) {

	tests := []struct {
		name  string
		route *route.Route

		responseCurTime string

		expectRes *route.Route
	}{
		{
			"empty item",
			&route.Route{
				ID: uuid.FromStringOrNil("df43b28c-4334-11ed-800b-1365aa60a589"),
			},

			"2020-04-18T03:22:17.995000Z",

			&route.Route{
				ID:       uuid.FromStringOrNil("df43b28c-4334-11ed-800b-1365aa60a589"),
				TMCreate: "2020-04-18T03:22:17.995000Z",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			"all item",
			&route.Route{
				ID:         uuid.FromStringOrNil("df888a56-4334-11ed-b4c2-8b00086523fb"),
				CustomerID: uuid.FromStringOrNil("efbf83de-4334-11ed-913a-07b6799785d2"),
				ProviderID: uuid.FromStringOrNil("efec350a-4334-11ed-a603-334dec19334f"),
				Priority:   1,
				Target:     "all",
			},

			"2020-04-18T03:22:17.995000Z",

			&route.Route{
				ID:         uuid.FromStringOrNil("df888a56-4334-11ed-b4c2-8b00086523fb"),
				CustomerID: uuid.FromStringOrNil("efbf83de-4334-11ed-913a-07b6799785d2"),
				ProviderID: uuid.FromStringOrNil("efec350a-4334-11ed-a603-334dec19334f"),
				Priority:   1,
				Target:     "all",
				TMCreate:   "2020-04-18T03:22:17.995000Z",
				TMUpdate:   commondatabasehandler.DefaultTimeStamp,
				TMDelete:   commondatabasehandler.DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			if err := h.RouteCreate(ctx, tt.route); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RouteGet(ctx, tt.route.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			res, err := h.RouteGet(ctx, tt.route.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteList(t *testing.T) {

	tests := []struct {
		name   string
		routes []route.Route

		limit uint64

		responseCurTime string
	}{
		{
			"normal",
			[]route.Route{
				{
					ID:         uuid.FromStringOrNil("d0c0df3a-6806-11ee-a3c0-17e6aa842a38"),
					CustomerID: uuid.FromStringOrNil("d0ef49f6-6806-11ee-b833-83dd4074c332"),
				},
				{
					ID:         uuid.FromStringOrNil("d1181228-6806-11ee-a6c5-67b6e351be48"),
					CustomerID: uuid.FromStringOrNil("d13e89da-6806-11ee-aeb5-8f493b0bc47e"),
				},
			},

			10,
			"2020-04-18T03:22:17.995000Z",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
			for _, r := range tt.routes {
				mockCache.EXPECT().RouteSet(ctx, gomock.Any())
				if err := h.RouteCreate(ctx, &r); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			filters := map[route.Field]any{}
			res, err := h.RouteList(ctx, utilhandler.TimeGetCurTime(), tt.limit, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < 1 {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", "bigger than 0", len(res))
			}
		})
	}
}

func Test_RouteListByCustomerID(t *testing.T) {

	tests := []struct {
		name   string
		routes []route.Route

		customerID uuid.UUID
		limit      uint64

		responseCurTime string

		expectRes []*route.Route
	}{
		{
			"normal",
			[]route.Route{
				{
					ID:         uuid.FromStringOrNil("4004e982-4336-11ed-99fc-53e93440d555"),
					CustomerID: uuid.FromStringOrNil("3fc93770-4336-11ed-a641-73b648571f6b"),
					TMCreate:   "2020-04-18T03:22:17.995000Z",
					TMUpdate:   "2020-04-18T03:22:17.995000Z",
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("40335d58-4336-11ed-b1ec-57f4e8d28783"),
					CustomerID: uuid.FromStringOrNil("3fc93770-4336-11ed-a641-73b648571f6b"),
					TMCreate:   "2020-04-18T03:22:17.995000Z",
					TMUpdate:   "2020-04-18T03:22:17.995000Z",
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("3fc93770-4336-11ed-a641-73b648571f6b"),
			10,

			"2020-04-18T03:22:17.995000Z",

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("40335d58-4336-11ed-b1ec-57f4e8d28783"),
					CustomerID: uuid.FromStringOrNil("3fc93770-4336-11ed-a641-73b648571f6b"),
					TMCreate:   "2020-04-18T03:22:17.995000Z",
					TMUpdate:   commondatabasehandler.DefaultTimeStamp,
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("4004e982-4336-11ed-99fc-53e93440d555"),
					CustomerID: uuid.FromStringOrNil("3fc93770-4336-11ed-a641-73b648571f6b"),
					TMCreate:   "2020-04-18T03:22:17.995000Z",
					TMUpdate:   commondatabasehandler.DefaultTimeStamp,
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
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
			for _, r := range tt.routes {
				mockCache.EXPECT().RouteSet(ctx, gomock.Any())
				if err := h.RouteCreate(ctx, &r); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			filters := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
			}
			res, err := h.RouteList(ctx, utilhandler.TimeGetCurTime(), tt.limit, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_RouteListByCustomerIDWithTarget(t *testing.T) {

	tests := []struct {
		name   string
		routes []route.Route

		customerID uuid.UUID
		target     string
		token      string

		responseCurTime string

		expectRes []*route.Route
	}{
		{
			"normal",
			[]route.Route{
				{
					ID:         uuid.FromStringOrNil("b0af5b76-4337-11ed-b068-3394fb21fec1"),
					CustomerID: uuid.FromStringOrNil("b048bb00-4337-11ed-96f0-4b0f7dc31ba1"),
					Target:     "all",
					Priority:   2,
				},
				{
					ID:         uuid.FromStringOrNil("b07cc8aa-4337-11ed-9c1f-6f01ee46218f"),
					CustomerID: uuid.FromStringOrNil("b048bb00-4337-11ed-96f0-4b0f7dc31ba1"),
					Target:     "all",
					Priority:   1,
				},
			},

			uuid.FromStringOrNil("b048bb00-4337-11ed-96f0-4b0f7dc31ba1"),
			"all",
			"2020-05-18T03:22:18.995000Z", // token should be after the TMCreate

			"2020-05-18T03:22:17.995000Z",

			// Results are ordered by TMCreate DESC, ID DESC
			// Since both have same TMCreate, they're ordered by ID DESC
			// b0af5b76 > b07cc8aa lexicographically, so b0af5b76 comes first
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("b0af5b76-4337-11ed-b068-3394fb21fec1"),
					CustomerID: uuid.FromStringOrNil("b048bb00-4337-11ed-96f0-4b0f7dc31ba1"),
					Target:     "all",
					Priority:   2,
					TMCreate:   "2020-05-18T03:22:17.995000Z",
					TMUpdate:   commondatabasehandler.DefaultTimeStamp,
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("b07cc8aa-4337-11ed-9c1f-6f01ee46218f"),
					CustomerID: uuid.FromStringOrNil("b048bb00-4337-11ed-96f0-4b0f7dc31ba1"),
					Target:     "all",
					Priority:   1,
					TMCreate:   "2020-05-18T03:22:17.995000Z",
					TMUpdate:   commondatabasehandler.DefaultTimeStamp,
					TMDelete:   commondatabasehandler.DefaultTimeStamp,
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
			for _, r := range tt.routes {
				mockCache.EXPECT().RouteSet(ctx, gomock.Any())
				if err := h.RouteCreate(ctx, &r); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			filters := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
				route.FieldTarget:     tt.target,
			}
			res, err := h.RouteList(ctx, tt.token, 1000, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_RouteDelete(t *testing.T) {

	tests := []struct {
		name  string
		route *route.Route
	}{
		{
			"normal",
			&route.Route{
				ID:       uuid.FromStringOrNil("76fc1f26-4338-11ed-bd70-1ba6021f2c4c"),
				TMCreate: "2020-04-18T03:22:17.995000Z",
				TMUpdate: "2020-04-18T03:22:17.995000Z",
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

			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			if err := h.RouteCreate(ctx, tt.route); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RouteDelete(ctx, tt.route.ID)
			if err := h.RouteDelete(ctx, tt.route.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RouteGet(ctx, tt.route.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().RouteSet(ctx, gomock.Any()).Return(nil)
			res, err := h.RouteGet(ctx, tt.route.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == commondatabasehandler.DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}
		})
	}
}

func Test_RouteUpdate(t *testing.T) {

	tests := []struct {
		name string

		data *route.Route

		id           uuid.UUID
		updateFields map[route.Field]any

		responseCurTime string

		expectRes *route.Route
	}{
		{
			name: "normal",

			data: &route.Route{
				ID:         uuid.FromStringOrNil("e8776eb6-432f-11ed-acde-b7089222dfd9"),
				CustomerID: uuid.FromStringOrNil("af531e46-4339-11ed-940d-1f29614aec4f"),
				ProviderID: uuid.FromStringOrNil("af23f6a2-4339-11ed-ba82-83f201c80803"),
				Priority:   1,
				Target:     "all",
			},

			id: uuid.FromStringOrNil("e8776eb6-432f-11ed-acde-b7089222dfd9"),
			updateFields: map[route.Field]any{
				route.FieldName:       "update name",
				route.FieldDetail:     "update detail",
				route.FieldProviderID: uuid.FromStringOrNil("f7855bcc-6b54-11ee-a216-bbb1db932bc9"),
				route.FieldPriority:   2,
				route.FieldTarget:     "+82",
			},

			responseCurTime: "2020-05-18T03:22:17.995000Z",

			expectRes: &route.Route{
				ID:         uuid.FromStringOrNil("e8776eb6-432f-11ed-acde-b7089222dfd9"),
				CustomerID: uuid.FromStringOrNil("af531e46-4339-11ed-940d-1f29614aec4f"),
				Name:       "update name",
				Detail:     "update detail",
				ProviderID: uuid.FromStringOrNil("f7855bcc-6b54-11ee-a216-bbb1db932bc9"),
				Priority:   2,
				Target:     "+82",
				TMCreate:   "2020-05-18T03:22:17.995000Z",
				TMUpdate:   "2020-05-18T03:22:17.995000Z",
				TMDelete:   commondatabasehandler.DefaultTimeStamp,
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
			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			if err := h.RouteCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			if err := h.RouteUpdate(ctx, tt.id, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().RouteGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().RouteSet(ctx, gomock.Any())
			res, err := h.RouteGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
