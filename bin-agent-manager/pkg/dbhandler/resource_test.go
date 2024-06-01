package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_ResourceCreate(t *testing.T) {

	tests := []struct {
		name     string
		resource *resource.Resource

		responseCurTime string
		expectRes       *resource.Resource
		expectRes2      []byte
	}{
		{
			name: "normal",
			resource: &resource.Resource{
				ID:            uuid.FromStringOrNil("d9321a20-1f63-11ef-bd92-57deb3f6cbbb"),
				CustomerID:    uuid.FromStringOrNil("d970d81e-1f63-11ef-8010-5300df6ebd4a"),
				AgentID:       uuid.FromStringOrNil("d99f3754-1f63-11ef-8636-2bd04a02e1e8"),
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("d9deef2a-1f63-11ef-a812-afe574b89b32"),
				Data: resource.Resource{
					ID: uuid.FromStringOrNil("fa2138aa-1f64-11ef-a8ea-f79130b00ec8"),
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &resource.Resource{
				ID:            uuid.FromStringOrNil("d9321a20-1f63-11ef-bd92-57deb3f6cbbb"),
				CustomerID:    uuid.FromStringOrNil("d970d81e-1f63-11ef-8010-5300df6ebd4a"),
				AgentID:       uuid.FromStringOrNil("d99f3754-1f63-11ef-8636-2bd04a02e1e8"),
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("d9deef2a-1f63-11ef-a812-afe574b89b32"),
				Data: map[string]interface{}{
					"agent_id":       "00000000-0000-0000-0000-000000000000",
					"customer_id":    "00000000-0000-0000-0000-000000000000",
					"data":           nil,
					"id":             "fa2138aa-1f64-11ef-a8ea-f79130b00ec8",
					"reference_id":   "00000000-0000-0000-0000-000000000000",
					"reference_type": "",
					"tm_create":      "",
					"tm_delete":      "",
					"tm_update":      "",
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
			expectRes2: []byte(`{"id":"d9321a20-1f63-11ef-bd92-57deb3f6cbbb","customer_id":"d970d81e-1f63-11ef-8010-5300df6ebd4a","agent_id":"d99f3754-1f63-11ef-8636-2bd04a02e1e8","reference_type":"call","reference_id":"d9deef2a-1f63-11ef-a812-afe574b89b32","data":{"agent_id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","data":null,"id":"fa2138aa-1f64-11ef-a8ea-f79130b00ec8","reference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","tm_create":"","tm_delete":"","tm_update":""},"tm_create":"2020-04-18 03:22:17.995000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
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
			mockCache.EXPECT().ResourceSet(gomock.Any(), gomock.Any())
			if err := h.ResourceCreate(ctx, tt.resource); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ResourceGet(gomock.Any(), tt.resource.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ResourceSet(gomock.Any(), gomock.Any())
			res, err := h.ResourceGet(ctx, tt.resource.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			res2, err := json.Marshal(res)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if string(res2) != string(tt.expectRes2) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes2, res2)
			}
		})
	}
}

func Test_ResourceDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent *resource.Resource

		responseCurTime string
		expectRes       *resource.Resource
	}{
		{
			name: "normal",
			agent: &resource.Resource{
				ID: uuid.FromStringOrNil("a2f503e4-2023-11ef-b9fd-c71b1004898c"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &resource.Resource{
				ID:       uuid.FromStringOrNil("a2f503e4-2023-11ef-b9fd-c71b1004898c"),
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			if err := h.ResourceCreate(ctx, tt.agent); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			if err := h.ResourceDelete(ctx, tt.agent.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ResourceGet(ctx, tt.agent.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			res, err := h.ResourceGet(ctx, tt.agent.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ResourceGets(t *testing.T) {

	tests := []struct {
		name      string
		resources []*resource.Resource

		size    uint64
		filters map[string]string

		responseCurTime string
		expectRes       []*resource.Resource
	}{
		{
			name: "normal",
			resources: []*resource.Resource{
				{
					ID:         uuid.FromStringOrNil("050af7a0-2024-11ef-8340-034f821d1ba7"),
					CustomerID: uuid.FromStringOrNil("04c0b898-2024-11ef-880d-7fcab3af32e5"),
				},
				{
					ID:         uuid.FromStringOrNil("0546ed78-2024-11ef-865e-7ba9fd7c71d4"),
					CustomerID: uuid.FromStringOrNil("04c0b898-2024-11ef-880d-7fcab3af32e5"),
				},
			},

			size: 2,
			filters: map[string]string{
				"customer_id": "04c0b898-2024-11ef-880d-7fcab3af32e5",
				"deleted":     "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*resource.Resource{
				{
					ID:         uuid.FromStringOrNil("050af7a0-2024-11ef-8340-034f821d1ba7"),
					CustomerID: uuid.FromStringOrNil("04c0b898-2024-11ef-880d-7fcab3af32e5"),
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("0546ed78-2024-11ef-865e-7ba9fd7c71d4"),
					CustomerID: uuid.FromStringOrNil("04c0b898-2024-11ef-880d-7fcab3af32e5"),
					TMCreate:   "2020-04-18 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			name:      "empty",
			resources: []*resource.Resource{},

			size: 2,
			filters: map[string]string{
				"reference_id": "864d6456-2024-11ef-a0ec-43a5d82b2024",
				"deleted":      "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes:       []*resource.Resource{},
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

			mockCache.EXPECT().ResourceSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.resources {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				if err := h.ResourceCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ResourceGets(ctx, tt.size, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ResourceSetData(t *testing.T) {

	tests := []struct {
		name      string
		resourcer *resource.Resource

		id   uuid.UUID
		data interface{}

		responseCurTime string
		expectRes       *resource.Resource
	}{
		{
			name: "normal",
			resourcer: &resource.Resource{
				ID: uuid.FromStringOrNil("3631a22c-2027-11ef-b099-f789b8cecd24"),
			},

			data: map[string]interface{}{
				"test": "test_data",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &resource.Resource{
				ID: uuid.FromStringOrNil("3631a22c-2027-11ef-b099-f789b8cecd24"),
				Data: map[string]interface{}{
					"test": "test_data",
				},
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			if err := h.ResourceCreate(ctx, tt.resourcer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			if err := h.ResourceSetData(ctx, tt.resourcer.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ResourceGet(ctx, tt.resourcer.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ResourceSet(ctx, gomock.Any())
			res, err := h.ResourceGet(ctx, tt.resourcer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
