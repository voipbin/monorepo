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
