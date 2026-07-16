package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func Test_WidgetCreate(t *testing.T) {

	tests := []struct {
		name string

		widget *widget.Widget

		responseCurTime *time.Time
		expectRes       *widget.Widget
	}{
		{
			name: "normal",
			widget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
					CustomerID: uuid.FromStringOrNil("4fc7cef8-7f54-11ec-8e1f-6f6a91905190"),
				},
				Name:               "test widget",
				Status:              widget.StatusActive,
				DirectID:            uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				WelcomeMessage:      "Hello!",
				FlowID:              uuid.FromStringOrNil("4dfaf278-205d-11f0-8be0-d74aed2ef0bc"),
				SessionIdleTimeout:  1800,
				ThemeConfig: &widget.ThemeConfig{
					PrimaryColor: "#3366ff",
					LogoURL:      "https://example.com/logo.png",
					Position:     widget.WidgetPositionBottomRight,
				},
			},

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
					CustomerID: uuid.FromStringOrNil("4fc7cef8-7f54-11ec-8e1f-6f6a91905190"),
				},
				Name:               "test widget",
				Status:              widget.StatusActive,
				DirectID:            uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				WelcomeMessage:      "Hello!",
				FlowID:              uuid.FromStringOrNil("4dfaf278-205d-11f0-8be0-d74aed2ef0bc"),
				SessionIdleTimeout:  1800,
				ThemeConfig: &widget.ThemeConfig{
					PrimaryColor: "#3366ff",
					LogoURL:      "https://example.com/logo.png",
					Position:     widget.WidgetPositionBottomRight,
				},
				TMCreate: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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
			mockCache.EXPECT().WidgetSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().WidgetGet(gomock.Any(), tt.widget.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.WidgetCreate(ctx, tt.widget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.WidgetGet(ctx, tt.widget.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_WidgetList(t *testing.T) {
	type test struct {
		name string
		data []*widget.Widget

		size    uint64
		token   string
		filters map[widget.Field]any

		responseCurtime *time.Time
		expectRes       []*widget.Widget
	}

	tests := []test{
		{
			name: "normal",
			data: []*widget.Widget{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("3ac1a1a8-6b7c-11f0-9c1f-eb7b6dcd0e0a"),
					},
					Name:   "list test widget",
					Status: widget.StatusActive,
				},
			},

			size:    10,
			token:   "",
			filters: map[widget.Field]any{widget.FieldCustomerID: uuid.FromStringOrNil("3ac1a1a8-6b7c-11f0-9c1f-eb7b6dcd0e0a"), widget.FieldDeleted: false},

			responseCurtime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: []*widget.Widget{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("3ac1a1a8-6b7c-11f0-9c1f-eb7b6dcd0e0a"),
					},
					Name:     "list test widget",
					Status:   widget.StatusActive,
					TMCreate: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, u := range tt.data {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurtime)
				mockCache.EXPECT().WidgetSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				if err := h.WidgetCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.NewUtilHandler().TimeGetCurTime()).AnyTimes()
			res, err := h.WidgetList(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != len(tt.expectRes) {
				t.Errorf("Wrong match. expect len: %d, got len: %d, res: %v", len(tt.expectRes), len(res), res)
			}
		})
	}
}

func Test_WidgetUpdate(t *testing.T) {

	tests := []struct {
		name string

		widget *widget.Widget

		id     uuid.UUID
		fields map[widget.Field]any

		responseCurTime *time.Time
		expectRes       *widget.Widget
	}{
		{
			name: "normal",
			widget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("1baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				Name:   "before update",
				Status: widget.StatusActive,
			},

			id: uuid.FromStringOrNil("1b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
			fields: map[widget.Field]any{
				widget.FieldName:   "after update",
				widget.FieldStatus: widget.StatusInactive,
			},

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("1baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				Name:     "after update",
				Status:   widget.StatusInactive,
				TMCreate: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMUpdate: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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
			mockCache.EXPECT().WidgetSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().WidgetGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.WidgetCreate(ctx, tt.widget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.WidgetUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.WidgetGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_WidgetDelete(t *testing.T) {

	tests := []struct {
		name string

		widget *widget.Widget

		id uuid.UUID

		responseCurTime *time.Time
	}{
		{
			name: "normal",
			widget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("2baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				Name:   "delete me",
				Status: widget.StatusActive,
			},

			id: uuid.FromStringOrNil("2b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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
			mockCache.EXPECT().WidgetSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().WidgetGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.WidgetCreate(ctx, tt.widget); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.WidgetDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.WidgetGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == nil {
				t.Errorf("Wrong match. expect: non-nil TMDelete, got: nil")
			}
		})
	}
}
