package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

func Test_OutdialCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:          dbTest,
		cache:       mockCache,
		utilHandler: utilhandler.NewUtilHandler(),
	}

	tests := []struct {
		name      string
		outdial   *outdial.Outdial
		expectRes *outdial.Outdial
	}{
		{
			"have no actions",
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1c1fede-abf7-11ec-afd7-c7d38010c6da"),
					CustomerID: uuid.FromStringOrNil("f1f9e8d0-abf7-11ec-a318-e3437a92791f"),
				},
				CampaignID: uuid.FromStringOrNil("f22a533a-abf7-11ec-b485-4fb99c030404"),

				Name:   "test outdial name",
				Detail: "test outdial detail",

				Data: "test uuid data string",

				TMCreate: "2020-04-18 03:22:17.995000",
			},
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1c1fede-abf7-11ec-afd7-c7d38010c6da"),
					CustomerID: uuid.FromStringOrNil("f1f9e8d0-abf7-11ec-a318-e3437a92791f"),
				},
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

func Test_OutdialList(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:          dbTest,
		cache:       mockCache,
		utilHandler: utilhandler.NewUtilHandler(),
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
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("8c2eff6c-abf8-11ec-aef8-9757be01931e"),
						CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					},
					Name:     "test1",
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("8c807824-abf8-11ec-bf9f-9f7c1fd38a43"),
						CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					},
					Name:     "test2",
					TMDelete: DefaultTimeStamp,
				},
			},
			[]*outdial.Outdial{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("8c807824-abf8-11ec-bf9f-9f7c1fd38a43"),
						CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					},
					Name:     "test2",
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("8c2eff6c-abf8-11ec-aef8-9757be01931e"),
						CustomerID: uuid.FromStringOrNil("8c5bc8e4-abf8-11ec-a5f5-fb2f8c29041b"),
					},
					Name:     "test1",
					TMDelete: DefaultTimeStamp,
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

			filters := map[outdial.Field]any{
				outdial.FieldCustomerID: tt.customerID,
				outdial.FieldDeleted:    false,
			}
			flows, err := h.OutdialList(ctx, GetCurTime(), tt.limit, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Clear timestamps for comparison
			for _, flow := range flows {
				flow.TMCreate = ""
				flow.TMUpdate = ""
			}

			// Check length first
			if len(flows) != len(tt.expectRes) {
				t.Errorf("Wrong length. expect: %d, got: %d", len(tt.expectRes), len(flows))
				return
			}

			// Check that all expected records are in results (order-independent)
			for _, expected := range tt.expectRes {
				found := false
				for _, flow := range flows {
					if flow.ID == expected.ID && flow.Name == expected.Name {
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

func Test_OutdialUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name    string
		outdial *outdial.Outdial

		fields map[outdial.Field]any

		expectRes *outdial.Outdial
	}{
		{
			"test basic info update",
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("802c0b50-abf9-11ec-bed8-3f61478b7331"),
				},
			},

			map[outdial.Field]any{
				outdial.FieldName:   "test name",
				outdial.FieldDetail: "test detail",
			},

			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("802c0b50-abf9-11ec-bed8-3f61478b7331"),
				},
				Name:   "test name",
				Detail: "test detail",
			},
		},
		{
			"test campaign id update",
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d985ffe4-abf9-11ec-9a44-232311136ad4"),
				},
				Name:   "test name",
				Detail: "test detail",
			},

			map[outdial.Field]any{
				outdial.FieldCampaignID: uuid.FromStringOrNil("d9a443b4-abf9-11ec-835b-d75f14d69cb2"),
			},

			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d985ffe4-abf9-11ec-9a44-232311136ad4"),
				},
				Name:       "test name",
				Detail:     "test detail",
				CampaignID: uuid.FromStringOrNil("d9a443b4-abf9-11ec-835b-d75f14d69cb2"),
			},
		},
		{
			"test data update",
			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0db4b65c-abfa-11ec-9e15-0b454feb2f7e"),
				},
				Name:   "test name",
				Detail: "test detail",
			},

			map[outdial.Field]any{
				outdial.FieldData: "test data string",
			},

			&outdial.Outdial{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0db4b65c-abfa-11ec-9e15-0b454feb2f7e"),
				},
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
			if err := h.OutdialUpdate(context.Background(), tt.outdial.ID, tt.fields); err != nil {
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
