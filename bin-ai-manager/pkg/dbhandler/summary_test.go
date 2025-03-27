package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_SummaryCreate(t *testing.T) {
	tests := []struct {
		name string

		summary *summary.Summary

		responseCurTime string
		expectRes       *summary.Summary
	}{
		{
			name: "normal",

			summary: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("69973904-0a48-11f0-8f10-037c653e7ac2"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},

				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("6a31d7ac-0a48-11f0-85af-af6f2cf78715"),

				Language: "en-US",
				Content:  "Hello",
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("69973904-0a48-11f0-8f10-037c653e7ac2"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},

				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("6a31d7ac-0a48-11f0-85af-af6f2cf78715"),

				Language: "en-US",
				Content:  "Hello",

				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			name: "empty",

			summary: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b4ec7c70-0a48-11f0-bfb1-9f0ee7583e2a"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},
			},

			responseCurTime: "2023-01-03 21:35:02.809",
			expectRes: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b4ec7c70-0a48-11f0-bfb1-9f0ee7583e2a"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
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
			mockCache.EXPECT().SummarySet(ctx, gomock.Any())
			if err := h.SummaryCreate(ctx, tt.summary); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			mockCache.EXPECT().SummaryGet(ctx, tt.summary.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().SummarySet(ctx, gomock.Any())
			res, err := h.SummaryGet(ctx, tt.summary.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			expectRes := []*summary.Summary{tt.expectRes}
			resGets, err := h.SummaryGets(ctx, 100, DefaultTimeStamp, map[string]string{"reference_id": tt.summary.ReferenceID.String()})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(expectRes, resGets) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, resGets)
			}
		})
	}
}
