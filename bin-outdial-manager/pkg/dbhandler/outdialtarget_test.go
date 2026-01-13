package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

func Test_OutdialTargetCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:          dbTest,
		cache:       mockCache,
		utilHandler: utilhandler.NewUtilHandler(),
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

			mockCache.EXPECT().OutdialTargetSet(ctx, gomock.Any().Return(nil)
			if err := h.OutdialTargetCreate(context.Background(), tt.outdialTarget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID.Return(nil, fmt.Errorf(""))
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

func Test_OutdialTargetGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:          dbTest,
		cache:       mockCache,
		utilHandler: utilhandler.NewUtilHandler(),
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

			filters := map[outdialtarget.Field]any{
				outdialtarget.FieldOutdialID: tt.outdialID,
				outdialtarget.FieldDeleted:   false,
			}
			targets, err := h.OutdialTargetGets(ctx, GetCurTime(), tt.limit, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Clear timestamps for comparison
			for _, target := range targets {
				target.TMCreate = ""
				target.TMUpdate = ""
			}

			// Check length first
			if len(targets) != len(tt.expectRes) {
				t.Errorf("Wrong length. expect: %d, got: %d", len(tt.expectRes), len(targets))
				return
			}

			// Check that all expected records are in results (order-independent)
			for _, expected := range tt.expectRes {
				found := false
				for _, target := range targets {
					if target.ID == expected.ID && target.Name == expected.Name {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected record not found. ID: %v, Name: %s", expected.ID, expected.Name)
				}
			}
		})
	}
}

func Test_OutdialTargetUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name          string
		outdialTarget *outdialtarget.OutdialTarget

		fields map[outdialtarget.Field]any

		expectRes *outdialtarget.OutdialTarget
	}{
		{
			"destinations update",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("026169e4-b01e-11ec-a313-bf2c237db05c"),
			},

			map[outdialtarget.Field]any{
				outdialtarget.FieldDestination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},

			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("026169e4-b01e-11ec-a313-bf2c237db05c"),

				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
		},
		{
			"status update",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("2843869a-b020-11ec-a2cb-27a25801b7ff"),
			},

			map[outdialtarget.Field]any{
				outdialtarget.FieldStatus: outdialtarget.StatusProgressing,
			},

			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("2843869a-b020-11ec-a2cb-27a25801b7ff"),
				Status: outdialtarget.StatusProgressing,
			},
		},
		{
			"basic info update",
			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("25943184-b09c-11ec-b638-b7a468abbfe7"),
				Name:   "test name",
				Detail: "test detail",
			},

			map[outdialtarget.Field]any{
				outdialtarget.FieldName:   "update name",
				outdialtarget.FieldDetail: "update detail",
			},

			&outdialtarget.OutdialTarget{
				ID:     uuid.FromStringOrNil("25943184-b09c-11ec-b638-b7a468abbfe7"),
				Name:   "update name",
				Detail: "update detail",
			},
		},
		{
			"data update",
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("9de7b3ee-b020-11ec-8959-7faf2dbb2434"),
			},

			map[outdialtarget.Field]any{
				outdialtarget.FieldData: "test data",
			},

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
			if err := h.OutdialTargetUpdate(context.Background(), tt.outdialTarget.ID, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID.Return(nil, fmt.Errorf(""))
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

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID.Return(nil, fmt.Errorf(""))
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

			mockCache.EXPECT().OutdialTargetGet(gomock.Any(), tt.outdialTarget.ID.Return(nil, fmt.Errorf(""))
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
