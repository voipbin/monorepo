package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

func Test_OutdialTargetCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget
		expectRes     *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("26f03072-b01b-11ec-8b95-b7c2633990d7"),
				OutdialID: uuid.FromStringOrNil("274f60f6-b01b-11ec-98c7-8377c4689265"),

				Name:   "test outdialtarget name",
				Detail: "test outdialtarget detail",

				Data:   "test uuid data string",
				Status: outdialtarget.StatusIdle,

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				TMCreate: "2020-04-18 03:22:17.995000",
			},
			&outdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("26f03072-b01b-11ec-8b95-b7c2633990d7"),
				OutdialID: uuid.FromStringOrNil("274f60f6-b01b-11ec-98c7-8377c4689265"),

				Name:   "test outdialtarget name",
				Detail: "test outdialtarget detail",

				Data:   "test uuid data string",
				Status: outdialtarget.StatusIdle,

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},

				TMCreate: "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().OutdialTargetSet(ctx, gomock.Any()).Return(nil)
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(ctx, tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created outdial. outdial: %v", res)

			tt.expectRes.TMCreate = res.TMCreate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetGetsByOutdialID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name           string
		outdialID      uuid.UUID
		limit          uint64
		outdialTargets []outdialtarget.OutdialTarget
		expectRes      []*outdialtarget.OutdialTarget
	}{
		{
			"normal",
			uuid.FromStringOrNil("786650b0-b01d-11ec-988e-c377bbf8e597"),
			10,
			[]outdialtarget.OutdialTarget{
				{
					ID:        uuid.FromStringOrNil("791b1d2e-b01d-11ec-b6eb-d3f27ead80d3"),
					OutdialID: uuid.FromStringOrNil("786650b0-b01d-11ec-988e-c377bbf8e597"),
					Name:      "test1",
					TMDelete:  DefaultTimeStamp,
				},
				{
					ID:        uuid.FromStringOrNil("794b32de-b01d-11ec-9bc7-fbbf15a88054"),
					OutdialID: uuid.FromStringOrNil("786650b0-b01d-11ec-988e-c377bbf8e597"),
					Name:      "test2",
					TMDelete:  DefaultTimeStamp,
				},
			},
			[]*outdialtarget.OutdialTarget{
				{
					ID:        uuid.FromStringOrNil("794b32de-b01d-11ec-9bc7-fbbf15a88054"),
					OutdialID: uuid.FromStringOrNil("786650b0-b01d-11ec-988e-c377bbf8e597"),
					Name:      "test2",
					TMDelete:  DefaultTimeStamp,
				},
				{
					ID:        uuid.FromStringOrNil("791b1d2e-b01d-11ec-b6eb-d3f27ead80d3"),
					OutdialID: uuid.FromStringOrNil("786650b0-b01d-11ec-988e-c377bbf8e597"),
					Name:      "test1",
					TMDelete:  DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, target := range tt.outdialTargets {
				mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
				if err := h.OutdialTargetCreate(ctx, &target); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			targets, err := h.OutdialTargetGetsByOutdialID(ctx, tt.outdialID, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, target := range targets {
				target.TMCreate = ""
			}

			if reflect.DeepEqual(targets, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, targets)
			}
		})
	}
}

func Test_OutdialTargetUpdateDestinations(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		destination0 *commonaddress.Address
		destination1 *commonaddress.Address
		destination2 *commonaddress.Address
		destination3 *commonaddress.Address
		destination4 *commonaddress.Address

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"address 0",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("026169e4-b01e-11ec-a313-bf2c237db05c"),
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			nil,
			nil,
			nil,
			nil,

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("026169e4-b01e-11ec-a313-bf2c237db05c"),

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
		},
		{
			"address 0-1",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("49c10130-b133-11ec-9f36-63a041749f9a"),
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},

			nil,
			nil,
			nil,

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("49c10130-b133-11ec-9f36-63a041749f9a"),

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination1: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
		},
		{
			"address 0-4",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("6f352e6e-b133-11ec-a3df-377e1b8025fd"),
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000005",
			},

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("6f352e6e-b133-11ec-a3df-377e1b8025fd"),

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination1: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination2: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
				Destination3: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000004",
				},
				Destination4: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000005",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetUpdateDestinations(
				context.Background(),
				tt.outdialTarget.ID,
				tt.destination0,
				tt.destination1,
				tt.destination2,
				tt.destination3,
				tt.destination4,
			); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetUpdateStatus(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		status outdialtarget.Status

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("2843869a-b020-11ec-a2cb-27a25801b7ff"),
			},

			outdialtarget.StatusProgressing,

			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("2843869a-b020-11ec-a2cb-27a25801b7ff"),
				Status: outdialtarget.StatusProgressing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetUpdateStatus(context.Background(), tt.outdialTarget.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetUpdateBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		outdialTargetName string
		detail            string

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("25943184-b09c-11ec-b638-b7a468abbfe7"),
				Name:   "test name",
				Detail: "test detail",
			},

			"update name",
			"update detail",

			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("25943184-b09c-11ec-b638-b7a468abbfe7"),
				Name:   "update name",
				Detail: "update detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetUpdateBasicInfo(context.Background(), tt.outdialTarget.ID, tt.outdialTargetName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetUpdateData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		data string

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("9de7b3ee-b020-11ec-8959-7faf2dbb2434"),
			},

			"test data",

			&outdialtarget.OutdialTarget{
				ID:   uuid.FromStringOrNil("9de7b3ee-b020-11ec-8959-7faf2dbb2434"),
				Data: "test data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetUpdateData(context.Background(), tt.outdialTarget.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget
		expectRes     *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("31c09b8c-b09c-11ec-b035-3bb8e78f59ea"),
			},
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("31c09b8c-b09c-11ec-b035-3bb8e78f59ea"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetDelete(context.Background(), tt.outdialTarget.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetUpdateProgressing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"normal",
			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("9ca25534-b0d4-11ec-ac00-a7008273cb03"),
				Status: outdialtarget.StatusIdle,
			},

			&outdialtarget.OutdialTarget{
				ID:        uuid.FromStringOrNil("9ca25534-b0d4-11ec-ac00-a7008273cb03"),
				Status:    outdialtarget.StatusProgressing,
				TryCount0: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			if err := h.OutdialTargetUpdateProgressing(context.Background(), tt.outdialTarget.ID, 0); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialTargetGet(context.Background(), tt.outdialTarget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_OutdialTargetGetAvailable(t *testing.T) {

// 	tests := []struct {
// 		name           string
// 		outdialTargets []*outdialtarget.OutdialTarget

// 		outdialID uuid.UUID
// 		tryCount0 int
// 		tryCount1 int
// 		tryCount2 int
// 		tryCount3 int
// 		tryCount4 int
// 		tmUpdate  string
// 		limit     uint64

// 		expectRes []*outdialtarget.OutdialTarget
// 	}{
// 		{
// 			"1 record",
// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("531cef62-b1ad-11ec-b35b-6baeb43808a6"),
// 					OutdialID: uuid.FromStringOrNil("6854c278-b1a9-11ec-a2ac-3315b086e86b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TMUpdate: "2022-04-01 01:22:17.995000",
// 				},
// 			},

// 			uuid.FromStringOrNil("6854c278-b1a9-11ec-a2ac-3315b086e86b"),
// 			1,
// 			0,
// 			0,
// 			0,
// 			0,
// 			"2022-04-03 01:22:17.995000",
// 			1,

// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("531cef62-b1ad-11ec-b35b-6baeb43808a6"),
// 					OutdialID: uuid.FromStringOrNil("6854c278-b1a9-11ec-a2ac-3315b086e86b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TMUpdate: "2022-04-01 01:22:17.995000",
// 				},
// 			},
// 		},
// 		{
// 			"2 records",
// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("d5d42942-b1ae-11ec-bf29-ff3839164805"),
// 					OutdialID: uuid.FromStringOrNil("e0559644-b1ae-11ec-9a02-f3031df53892"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 				{
// 					ID:        uuid.FromStringOrNil("f120b7f6-b1ae-11ec-aec6-134dcb0d87e0"),
// 					OutdialID: uuid.FromStringOrNil("e0559644-b1ae-11ec-9a02-f3031df53892"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000002",
// 					},
// 					TryCount0: 2,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},

// 			uuid.FromStringOrNil("e0559644-b1ae-11ec-9a02-f3031df53892"),
// 			3,
// 			0,
// 			0,
// 			0,
// 			0,
// 			"2022-04-03 01:22:17.995000",
// 			1,

// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("d5d42942-b1ae-11ec-bf29-ff3839164805"),
// 					OutdialID: uuid.FromStringOrNil("e0559644-b1ae-11ec-9a02-f3031df53892"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},
// 		},
// 		{
// 			"3 records",
// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("63c77f9c-b1af-11ec-88af-27dedd49efe4"),
// 					OutdialID: uuid.FromStringOrNil("64609e84-b1af-11ec-b713-8b0b5de2ef2b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 				{
// 					ID:        uuid.FromStringOrNil("63f39442-b1af-11ec-a341-33e7cceae24a"),
// 					OutdialID: uuid.FromStringOrNil("64609e84-b1af-11ec-b713-8b0b5de2ef2b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000002",
// 					},
// 					TryCount0: 2,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 				{
// 					ID:        uuid.FromStringOrNil("6430cb64-b1af-11ec-8baa-7b72d71a40b6"),
// 					OutdialID: uuid.FromStringOrNil("64609e84-b1af-11ec-b713-8b0b5de2ef2b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000003",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},

// 			uuid.FromStringOrNil("64609e84-b1af-11ec-b713-8b0b5de2ef2b"),
// 			3,
// 			0,
// 			0,
// 			0,
// 			0,
// 			"2022-04-03 01:22:17.995000",
// 			1,

// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("63c77f9c-b1af-11ec-88af-27dedd49efe4"),
// 					OutdialID: uuid.FromStringOrNil("64609e84-b1af-11ec-b713-8b0b5de2ef2b"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},
// 		},
// 		{
// 			"2 records with 1 invalid tm update",
// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("20a530ac-b2a8-11ec-baff-7b0c0eefeed6"),
// 					OutdialID: uuid.FromStringOrNil("3fdf22b6-b2a8-11ec-8c5d-3732536fe5dc"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 0,
// 					TMUpdate:  "2022-04-04 01:22:17.995000",
// 				},
// 				{
// 					ID:        uuid.FromStringOrNil("2143ff98-b2a8-11ec-b471-6b2b04d9be92"),
// 					OutdialID: uuid.FromStringOrNil("3fdf22b6-b2a8-11ec-8c5d-3732536fe5dc"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 3,
// 					TryCount1: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},

// 			uuid.FromStringOrNil("3fdf22b6-b2a8-11ec-8c5d-3732536fe5dc"),
// 			3,
// 			2,
// 			0,
// 			0,
// 			0,
// 			"2022-04-03 01:22:17.995000",
// 			1,

// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("2143ff98-b2a8-11ec-b471-6b2b04d9be92"),
// 					OutdialID: uuid.FromStringOrNil("3fdf22b6-b2a8-11ec-8c5d-3732536fe5dc"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 3,
// 					TryCount1: 1,
// 					TMUpdate:  "2022-04-01 01:22:17.995000",
// 				},
// 			},
// 		},
// 		{
// 			"no record",
// 			[]*outdialtarget.OutdialTarget{
// 				{
// 					ID:        uuid.FromStringOrNil("a444f730-b1ae-11ec-bf46-a3eb12bf058c"),
// 					OutdialID: uuid.FromStringOrNil("a53291c0-b1ae-11ec-b650-3f34598e1b89"),
// 					Status:    outdialtarget.StatusIdle,
// 					Destination0: &cmaddress.Address{
// 						Type:   cmaddress.TypeTel,
// 						Target: "+821100000001",
// 					},
// 					TryCount0: 1,
// 					TMUpdate:  "2022-04-01 03:22:17.995000",
// 				},
// 			},

// 			uuid.FromStringOrNil("a53291c0-b1ae-11ec-b650-3f34598e1b89"),
// 			1,
// 			0,
// 			0,
// 			0,
// 			0,
// 			"2022-04-01 01:22:17.995000",
// 			1,

// 			[]*outdialtarget.OutdialTarget{},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockCache := cachehandler.NewMockCacheHandler(mc)
// 			h := &handler{
// 				db:    dbTest,
// 				cache: mockCache,
// 			}

// 			ctx := context.Background()
// 			for _, target := range tt.outdialTargets {
// 				mockCache.EXPECT().OutdialTargetSet(gomock.Any(), gomock.Any())
// 				if err := h.OutdialTargetCreate(ctx, target); err != nil {
// 					t.Errorf("Wrong match. expect: ok, got: %v", err)
// 				}
// 			}

// 			res, err := h.OutdialTargetGetAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.tmUpdate, tt.limit)
// 			if err != nil {
// 				t.Errorf("Wroong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
