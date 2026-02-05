package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_SummaryCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string

		summary *summary.Summary

		responseCurTime *time.Time
		expectRes       *summary.Summary
	}{
		{
			name: "normal",

			summary: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("69973904-0a48-11f0-8f10-037c653e7ac2"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},

				ActiveflowID:  uuid.FromStringOrNil("73c7019a-0ba4-11f0-aee5-9b7073db9f34"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("6a31d7ac-0a48-11f0-85af-af6f2cf78715"),

				Status:   summary.StatusProgressing,
				Language: "en-US",
				Content:  "Hello",
			},

			responseCurTime: curTime,
			expectRes: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("69973904-0a48-11f0-8f10-037c653e7ac2"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},

				ActiveflowID:  uuid.FromStringOrNil("73c7019a-0ba4-11f0-aee5-9b7073db9f34"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("6a31d7ac-0a48-11f0-85af-af6f2cf78715"),

				Status:   summary.StatusProgressing,
				Language: "en-US",
				Content:  "Hello",

				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
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

			responseCurTime: curTime,
			expectRes: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b4ec7c70-0a48-11f0-bfb1-9f0ee7583e2a"),
					CustomerID: uuid.FromStringOrNil("6a04b59c-0a48-11f0-a206-d723dd7442a6"),
				},
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime.Format(utilhandler.ISO8601Layout))
			resGets, err := h.SummaryList(ctx, 100, "", map[summary.Field]any{summary.FieldReferenceID: tt.summary.ReferenceID})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(expectRes, resGets) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, resGets)
			}
		})
	}
}

func Test_SummaryUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		summary *summary.Summary

		id     uuid.UUID
		fields map[summary.Field]any

		responseCurTime *time.Time
		expectRes       *summary.Summary
	}{
		{
			name: "normal",

			summary: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("0951c95e-0bd5-11f0-a747-c75eccfcb319"),
					CustomerID: uuid.FromStringOrNil("098d4466-0bd5-11f0-9456-8fd0aa51f485"),
				},

				ActiveflowID:  uuid.FromStringOrNil("09b1b5b2-0bd5-11f0-a4cd-139990020880"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("09dc263a-0bd5-11f0-82fa-6b4b3357de0f"),

				Status:   summary.StatusProgressing,
				Language: "en-US",
			},

			id: uuid.FromStringOrNil("0951c95e-0bd5-11f0-a747-c75eccfcb319"),
			fields: map[summary.Field]any{
				summary.FieldStatus:  summary.StatusDone,
				summary.FieldContent: "test content",
			},

			responseCurTime: curTime,
			expectRes: &summary.Summary{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("0951c95e-0bd5-11f0-a747-c75eccfcb319"),
					CustomerID: uuid.FromStringOrNil("098d4466-0bd5-11f0-9456-8fd0aa51f485"),
				},

				ActiveflowID:  uuid.FromStringOrNil("09b1b5b2-0bd5-11f0-a4cd-139990020880"),
				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("09dc263a-0bd5-11f0-82fa-6b4b3357de0f"),

				Status:   summary.StatusDone,
				Language: "en-US",
				Content:  "test content",

				TMCreate: curTime,
				TMUpdate: curTime,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().SummarySet(ctx, gomock.Any())
			if err := h.SummaryCreate(ctx, tt.summary); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().SummarySet(ctx, gomock.Any())
			if errUpdate := h.SummaryUpdate(ctx, tt.id, tt.fields); errUpdate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errUpdate)
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
		})
	}
}
