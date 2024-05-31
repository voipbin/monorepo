package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
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
				// Data:          []byte(`{"id":"da160a64-1f63-11ef-84fb-a7d2e713626b"}`),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &resource.Resource{
				ID:            uuid.FromStringOrNil("d9321a20-1f63-11ef-bd92-57deb3f6cbbb"),
				CustomerID:    uuid.FromStringOrNil("d970d81e-1f63-11ef-8010-5300df6ebd4a"),
				AgentID:       uuid.FromStringOrNil("d99f3754-1f63-11ef-8636-2bd04a02e1e8"),
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("d9deef2a-1f63-11ef-a812-afe574b89b32"),
				Data:          []byte(`{"id":"da160a64-1f63-11ef-84fb-a7d2e713626b"}`),
				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
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

			// if reflect.DeepEqual(tt.expectRes, res) == false {
			// 	t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			// }

			tmp, err := json.Marshal(res)
			t.Errorf("tmp: %v", tmp)
		})
	}
}
