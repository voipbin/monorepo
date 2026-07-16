package sessionhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		widgetID   uuid.UUID

		responseUUID    uuid.UUID
		responseSession *session.Session

		expectRes *session.Session
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
			widgetID:   uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),

			responseUUID: uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
			responseSession: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				WidgetID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				Status:   session.StatusActive,
			},

			expectRes: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
				},
				WidgetID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f"),
				Status:   session.StatusActive,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &sessionHandler{
				utilHandler: mockUtil,
				db:          mockDB,
				reqHandler:  requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().SessionGet(ctx, tt.responseUUID).Return(tt.responseSession, nil)

			res, err := h.Create(ctx, tt.customerID, tt.widgetID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
