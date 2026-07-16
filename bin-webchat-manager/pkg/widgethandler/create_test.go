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

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID          uuid.UUID
		widgetName          string
		welcomeMessage      string
		flowID              uuid.UUID
		sessionIdleTimeout  int
		themeConfig         *widget.ThemeConfig

		responseUUID   uuid.UUID
		responseWidget *widget.Widget

		expectRes *widget.Widget
	}{
		{
			name: "normal",

			customerID:         uuid.FromStringOrNil("1ed812a6-7f56-11ec-82c1-8bb47b0f9d98"),
			widgetName:         "test widget",
			welcomeMessage:     "Hello!",
			flowID:             uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30"),
			sessionIdleTimeout: 1800,
			themeConfig: &widget.ThemeConfig{
				PrimaryColor: "#3366ff",
			},

			responseUUID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
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

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &widgetHandler{
				utilHandler: mockUtil,
				db:          mockDB,
				reqHandler:  mockReq,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.customerID, dmdirect.ResourceTypeWebchatWidget, tt.responseUUID).Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				},
				Hash: "test-hash",
			}, nil)
			mockDB.EXPECT().WidgetCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().WidgetGet(ctx, tt.responseUUID).Return(tt.responseWidget, nil)

			res, err := h.Create(
				ctx,
				tt.customerID,
				tt.widgetName,
				tt.welcomeMessage,
				tt.flowID,
				tt.sessionIdleTimeout,
				tt.themeConfig,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
