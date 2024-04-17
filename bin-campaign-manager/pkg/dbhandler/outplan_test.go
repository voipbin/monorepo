package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

func Test_OutplanCreate(t *testing.T) {
	tests := []struct {
		name    string
		outplan *outplan.Outplan

		responseCurTime string
		expectRes       *outplan.Outplan
	}{
		{
			"normal",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("504dbdd4-b3b5-11ec-b050-8f20b3a62441"),
				CustomerID: uuid.FromStringOrNil("50745688-b3b5-11ec-91bd-c3d3ee057cb1"),
				Name:       "test name",
				Detail:     "test detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
			},

			"2020-04-18 03:22:17.995000",

			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("504dbdd4-b3b5-11ec-b050-8f20b3a62441"),
				CustomerID: uuid.FromStringOrNil("50745688-b3b5-11ec-91bd-c3d3ee057cb1"),
				Name:       "test name",
				Detail:     "test detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutplanCreate(context.Background(), tt.outplan); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutplanGet(gomock.Any(), tt.outplan.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			res, err := h.OutplanGet(ctx, tt.outplan.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created outdial. outdial: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanDelete(t *testing.T) {

	tests := []struct {
		name    string
		outplan *outplan.Outplan

		responseCurTime string
		expectRes       *outplan.Outplan
	}{
		{
			"normal",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("9a72c25e-b47f-11ec-8c84-fbce9a6f9ddf"),
				CustomerID: uuid.FromStringOrNil("9aa97862-b47f-11ec-a611-5379cfa62666"),

				Name:   "test name",
				Detail: "test detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
			},

			"2020-04-18 03:22:17.995000",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("9a72c25e-b47f-11ec-8c84-fbce9a6f9ddf"),
				CustomerID: uuid.FromStringOrNil("9aa97862-b47f-11ec-a611-5379cfa62666"),

				Name:   "test name",
				Detail: "test detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,

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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			if err := h.OutplanCreate(context.Background(), tt.outplan); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			if err := h.OutplanDelete(context.Background(), tt.outplan.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutplanGet(gomock.Any(), tt.outplan.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			res, err := h.OutplanGet(context.Background(), tt.outplan.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_OutplanGetsByCustomerID(t *testing.T) {
	tests := []struct {
		name     string
		outplans []*outplan.Outplan

		customerID uuid.UUID
		token      string
		limit      uint64

		responseCurtime string
		expectRes       []*outplan.Outplan
	}{
		{
			"1 item",
			[]*outplan.Outplan{
				{
					ID:         uuid.FromStringOrNil("0b2e5afe-b3b7-11ec-90fb-0f96dcc8665c"),
					CustomerID: uuid.FromStringOrNil("0e4af5f8-b3b7-11ec-b721-578bb8a6f432"),

					Name:   "test name",
					Detail: "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,
				},
			},

			uuid.FromStringOrNil("0e4af5f8-b3b7-11ec-b721-578bb8a6f432"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*outplan.Outplan{
				{
					ID:         uuid.FromStringOrNil("0b2e5afe-b3b7-11ec-90fb-0f96dcc8665c"),
					CustomerID: uuid.FromStringOrNil("0e4af5f8-b3b7-11ec-b721-578bb8a6f432"),

					Name:   "test name",
					Detail: "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			"2 items",
			[]*outplan.Outplan{
				{
					ID:         uuid.FromStringOrNil("373b3dc8-b3b8-11ec-b5ef-dfeccd01e42e"),
					CustomerID: uuid.FromStringOrNil("37671b14-b3b8-11ec-a203-532a1edfa496"),
					Name:       "test name",
					Detail:     "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,
				},
				{
					ID:         uuid.FromStringOrNil("3792fa72-b3b8-11ec-94f5-ff4b74330ee9"),
					CustomerID: uuid.FromStringOrNil("37671b14-b3b8-11ec-a203-532a1edfa496"),
					Name:       "test name",
					Detail:     "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,
				},
			},

			uuid.FromStringOrNil("37671b14-b3b8-11ec-a203-532a1edfa496"),
			"2022-04-18 03:22:17.995000",
			100,

			"2020-04-18 03:22:17.995000",
			[]*outplan.Outplan{
				{
					ID:         uuid.FromStringOrNil("3792fa72-b3b8-11ec-94f5-ff4b74330ee9"),
					CustomerID: uuid.FromStringOrNil("37671b14-b3b8-11ec-a203-532a1edfa496"),
					Name:       "test name",
					Detail:     "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("373b3dc8-b3b8-11ec-b5ef-dfeccd01e42e"),
					CustomerID: uuid.FromStringOrNil("37671b14-b3b8-11ec-a203-532a1edfa496"),
					Name:       "test name",
					Detail:     "test detail",

					Source: &commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},

					DialTimeout: 30000,
					TryInterval: 600000,

					MaxTryCount0: 3,
					MaxTryCount1: 3,
					MaxTryCount2: 3,
					MaxTryCount3: 3,
					MaxTryCount4: 3,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
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

			for _, p := range tt.outplans {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurtime)
				mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
				if err := h.OutplanCreate(ctx, p); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.OutplanGetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created outdial. outdial: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name    string
		outplan *outplan.Outplan

		outplanName string
		detail      string

		responseCurTime string
		expectRes       *outplan.Outplan
	}{
		{
			"normal",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("b231e8a0-b3d2-11ec-b78a-57bdcb8f39c3"),
				CustomerID: uuid.FromStringOrNil("0e4af5f8-b3b7-11ec-b721-578bb8a6f432"),
				Name:       "test name",
				Detail:     "test detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
			},

			"update name",
			"update detail",

			"2020-04-18 03:22:17.995000",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("b231e8a0-b3d2-11ec-b78a-57bdcb8f39c3"),
				CustomerID: uuid.FromStringOrNil("0e4af5f8-b3b7-11ec-b721-578bb8a6f432"),
				Name:       "update name",
				Detail:     "update detail",

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				DialTimeout: 30000,
				TryInterval: 600000,

				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     "2020-04-18 03:22:17.995000",
				TMDelete:     DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutplanCreate(context.Background(), tt.outplan); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutplanUpdateBasicInfo(ctx, tt.outplan.ID, tt.outplanName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutplanGet(gomock.Any(), tt.outplan.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			res, err := h.OutplanGet(ctx, tt.outplan.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateDialInfo(t *testing.T) {
	tests := []struct {
		name    string
		outplan *outplan.Outplan

		source       *commonaddress.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		responseCurTime string
		expectRes       *outplan.Outplan
	}{
		{
			"normal",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("78f2b8de-b3ce-11ec-b4f4-e7c49d54d606"),
				CustomerID: uuid.FromStringOrNil("d24cd7f8-b3b9-11ec-9c73-071ce4f4b4ed"),
				Name:       "test name",
				Detail:     "test detail",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialTimeout:  30000,
				TryInterval:  600000,
				MaxTryCount0: 3,
				MaxTryCount1: 3,
				MaxTryCount2: 3,
				MaxTryCount3: 3,
				MaxTryCount4: 3,
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			60000,
			300000,
			2,
			2,
			2,
			2,
			2,

			"2020-04-18 03:22:17.995000",
			&outplan.Outplan{
				ID:         uuid.FromStringOrNil("78f2b8de-b3ce-11ec-b4f4-e7c49d54d606"),
				CustomerID: uuid.FromStringOrNil("d24cd7f8-b3b9-11ec-9c73-071ce4f4b4ed"),
				Name:       "test name",
				Detail:     "test detail",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DialTimeout:  60000,
				TryInterval:  300000,
				MaxTryCount0: 2,
				MaxTryCount1: 2,
				MaxTryCount2: 2,
				MaxTryCount3: 2,
				MaxTryCount4: 2,
				TMCreate:     "2020-04-18 03:22:17.995000",
				TMUpdate:     "2020-04-18 03:22:17.995000",
				TMDelete:     DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutplanCreate(ctx, tt.outplan); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().OutplanSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutplanUpdateDialInfo(ctx, tt.outplan.ID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutplanGet(gomock.Any(), tt.outplan.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutplanSet(gomock.Any(), gomock.Any())
			res, err := h.OutplanGet(ctx, tt.outplan.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
