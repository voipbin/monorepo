package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

func Test_OutdialCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name      string
		outdial   *outdial.Outdial
		expectRes *outdial.Outdial
	}{
		{
			"have no actions",
			&outdial.Outdial{
				ID:         uuid.FromStringOrNil("f1c1fede-abf7-11ec-afd7-c7d38010c6da"),
				CustomerID: uuid.FromStringOrNil("f1f9e8d0-abf7-11ec-a318-e3437a92791f"),
				CampaignID: uuid.FromStringOrNil("f22a533a-abf7-11ec-b485-4fb99c030404"),

				Name:   "test outdial name",
				Detail: "test outdial detail",

				Data: "test uuid data string",

				TMCreate: "2020-04-18 03:22:17.995000",
			},
			&outdial.Outdial{
				ID:         uuid.FromStringOrNil("f1c1fede-abf7-11ec-afd7-c7d38010c6da"),
				CustomerID: uuid.FromStringOrNil("f1f9e8d0-abf7-11ec-a318-e3437a92791f"),
				CampaignID: uuid.FromStringOrNil("f22a533a-abf7-11ec-b485-4fb99c030404"),

				Name:   "test outdial name",
				Detail: "test outdial detail",

				Data: "test uuid data string",

				TMCreate: "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialCreate(context.Background(), tt.outdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialGet(gomock.Any(), tt.outdial.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialGet(context.Background(), tt.outdial.ID)
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

func Test_OutdialGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name       string
		customerID uuid.UUID
		limit      uint64
		outdials   []outdial.Outdial
		expectRes  []*outdial.Outdial
	}{
		{
			"normal",
			uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
			10,
			[]outdial.Outdial{
				{
					ID:         uuid.FromStringOrNil("8c2eff6c-abf8-11ec-aef8-9757be01931e"),
					CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					Name:       "test1",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("8c807824-abf8-11ec-bf9f-9f7c1fd38a43"),
					CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					Name:       "test2",
					TMDelete:   DefaultTimeStamp,
				},
			},
			[]*outdial.Outdial{
				{
					ID:         uuid.FromStringOrNil("8c807824-abf8-11ec-bf9f-9f7c1fd38a43"),
					CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					Name:       "test2",
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("8c2eff6c-abf8-11ec-aef8-9757be01931e"),
					CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					Name:       "test1",
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, flow := range tt.outdials {
				mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
				if err := h.OutdialCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			flows, err := h.OutdialGetsByCustomerID(ctx, tt.customerID, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, flow := range flows {
				flow.TMCreate = ""
			}

			if reflect.DeepEqual(flows, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, flows)
			}
		})
	}
}

func Test_OutdialUpdateBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name    string
		outdial *outdial.Outdial

		outdialName string
		detail      string

		expectRes *outdial.Outdial
	}{
		{
			"test normal",
			&outdial.Outdial{
				ID: uuid.FromStringOrNil("802c0b50-abf9-11ec-bed8-3f61478b7331"),
			},

			"test name",
			"test detail",

			&outdial.Outdial{
				ID:     uuid.FromStringOrNil("802c0b50-abf9-11ec-bed8-3f61478b7331"),
				Name:   "test name",
				Detail: "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialCreate(context.Background(), tt.outdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialUpdateBasicInfo(context.Background(), tt.outdial.ID, tt.outdialName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialGet(gomock.Any(), tt.outdial.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialGet(context.Background(), tt.outdial.ID)
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

func Test_OutdialUpdateCampaignID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name    string
		outdial *outdial.Outdial

		campaignID uuid.UUID

		expectRes *outdial.Outdial
	}{
		{
			"test normal",
			&outdial.Outdial{
				ID:     uuid.FromStringOrNil("d985ffe4-abf9-11ec-9a44-232311136ad4"),
				Name:   "test name",
				Detail: "test detail",
			},

			uuid.FromStringOrNil("d9a443b4-abf9-11ec-835b-d75f14d69cb2"),

			&outdial.Outdial{
				ID:         uuid.FromStringOrNil("d985ffe4-abf9-11ec-9a44-232311136ad4"),
				Name:       "test name",
				Detail:     "test detail",
				CampaignID: uuid.FromStringOrNil("d9a443b4-abf9-11ec-835b-d75f14d69cb2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialCreate(context.Background(), tt.outdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialUpdateCampaignID(context.Background(), tt.outdial.ID, tt.campaignID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialGet(gomock.Any(), tt.outdial.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialGet(context.Background(), tt.outdial.ID)
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

func Test_OutdialUpdateData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name    string
		outdial *outdial.Outdial

		data string

		expectRes *outdial.Outdial
	}{
		{
			"test normal",
			&outdial.Outdial{
				ID:     uuid.FromStringOrNil("0db4b65c-abfa-11ec-9e15-0b454feb2f7e"),
				Name:   "test name",
				Detail: "test detail",
			},

			"test data string",

			&outdial.Outdial{
				ID:     uuid.FromStringOrNil("0db4b65c-abfa-11ec-9e15-0b454feb2f7e"),
				Name:   "test name",
				Detail: "test detail",
				Data:   "test data string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialCreate(context.Background(), tt.outdial); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			if err := h.OutdialUpdateData(context.Background(), tt.outdial.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialGet(gomock.Any(), tt.outdial.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialSet(gomock.Any(), gomock.Any())
			res, err := h.OutdialGet(context.Background(), tt.outdial.ID)
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
