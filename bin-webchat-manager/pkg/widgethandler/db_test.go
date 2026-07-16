package widgethandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseWidget *widget.Widget
		expectRes      *widget.Widget
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			responseWidget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &widgetHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().WidgetGet(ctx, tt.id).Return(tt.responseWidget, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_List(t *testing.T) {
	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[widget.Field]any

		responseWidgets []*widget.Widget
		expectRes       []*widget.Widget
	}{
		{
			name:    "normal",
			size:    10,
			token:   "",
			filters: map[widget.Field]any{},
			responseWidgets: []*widget.Widget{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
					},
				},
			},
			expectRes: []*widget.Widget{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &widgetHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().WidgetList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseWidgets, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		id                 uuid.UUID
		widgetName         string
		welcomeMessage     string
		flowID             uuid.UUID
		sessionIdleTimeout int
		themeConfig        *widget.ThemeConfig

		responseWidget *widget.Widget
		expectRes      *widget.Widget
	}{
		{
			name:               "normal",
			id:                 uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			widgetName:         "updated widget",
			welcomeMessage:     "Hi there!",
			flowID:             uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30"),
			sessionIdleTimeout: 3600,
			themeConfig:        &widget.ThemeConfig{PrimaryColor: "#ff0000"},

			responseWidget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &widgetHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().WidgetUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().WidgetGet(ctx, tt.id).Return(tt.responseWidget, nil)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.widgetName, tt.welcomeMessage, tt.flowID, tt.sessionIdleTimeout, tt.themeConfig)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseWidget *widget.Widget
		expectRes      *widget.Widget
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			responseWidget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
				DirectID: uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
			},
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
				DirectID: uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &widgetHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  mockReq,
			}
			ctx := context.Background()

			// first Get to fetch direct_id, then WidgetDelete, then final Get
			mockDB.EXPECT().WidgetGet(ctx, tt.id).Return(tt.responseWidget, nil).Times(2)
			mockReq.EXPECT().DirectV1DirectDelete(ctx, tt.responseWidget.DirectID).Return(&dmdirect.Direct{}, nil)
			mockDB.EXPECT().WidgetDelete(ctx, tt.id).Return(nil)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DirectHashRegenerate(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseWidget *widget.Widget
		expectRes      *widget.Widget
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
			responseWidget: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
				DirectID: uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
			},
			expectRes: &widget.Widget{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				},
				DirectID: uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &widgetHandler{
				utilHandler: utilhandler.NewMockUtilHandler(mc),
				db:          mockDB,
				reqHandler:  mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().WidgetGet(ctx, tt.id).Return(tt.responseWidget, nil).Times(2)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseWidget.DirectID).Return(&dmdirect.Direct{}, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &widgetHandler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          mockDB,
		reqHandler:  requesthandler.NewMockRequestHandler(mc),
	}
	ctx := context.Background()

	mockDB.EXPECT().WidgetGet(ctx, id).Return(&widget.Widget{
		Identity: commonidentity.Identity{ID: id},
	}, nil)

	res, err := h.DirectHashRegenerate(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok, res: %v", res)
	}
}
