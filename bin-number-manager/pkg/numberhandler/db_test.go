package numberhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
)

func Test_dbCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID          uuid.UUID
		num                 string
		callFlowID          uuid.UUID
		messageFlowID       uuid.UUID
		numberName          string
		detail              string
		providerName        number.ProviderName
		providerReferenceID string
		status              number.Status
		t38Enabled          bool
		emergencyEnabled    bool

		responseUUID   uuid.UUID
		responseNumber *number.Number

		expectNumber *number.Number
	}{
		{
			name: "normal",

			customerID:          uuid.FromStringOrNil("9469dadc-1f4f-11ee-8336-df1969096eee"),
			num:                 "+821100000001",
			callFlowID:          uuid.FromStringOrNil("94a0b5c0-1f4f-11ee-aae2-4b3d5394a85a"),
			messageFlowID:       uuid.FromStringOrNil("94cd4568-1f4f-11ee-8246-4f9a649f4565"),
			numberName:          "test name",
			detail:              "test detail",
			providerName:        number.ProviderNameTelnyx,
			providerReferenceID: "94fadd84-1f4f-11ee-a287-cbe263e37e8c",
			status:              number.StatusActive,
			t38Enabled:          true,
			emergencyEnabled:    false,

			responseUUID: uuid.FromStringOrNil("95223280-1f4f-11ee-91f2-7703e1598c47"),
			responseNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("95223280-1f4f-11ee-91f2-7703e1598c47"),
				},
			},

			expectNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("95223280-1f4f-11ee-91f2-7703e1598c47"),
					CustomerID: uuid.FromStringOrNil("9469dadc-1f4f-11ee-8336-df1969096eee"),
				},
				Number:              "+821100000001",
				Type:                number.TypeNormal,
				CallFlowID:          uuid.FromStringOrNil("94a0b5c0-1f4f-11ee-aae2-4b3d5394a85a"),
				MessageFlowID:       uuid.FromStringOrNil("94cd4568-1f4f-11ee-8246-4f9a649f4565"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "94fadd84-1f4f-11ee-a287-cbe263e37e8c",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockDB.EXPECT().NumberCreate(ctx, tt.expectNumber).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, tt.responseUUID).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseNumber.CustomerID, number.EventTypeNumberCreated, tt.responseNumber)

			res, err := h.dbCreate(ctx, tt.customerID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail, number.TypeNormal, tt.providerName, tt.providerReferenceID, tt.status, tt.t38Enabled, tt.emergencyEnabled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}

func Test_dbUpdate(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[number.Field]any

		responseNumber *number.Number
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b14ed168-20b0-11ee-b635-cf0e0e6774ba"),
			fields: map[number.Field]any{
				number.FieldCallFlowID:    uuid.FromStringOrNil("b1884696-20b0-11ee-8cd8-4315da0cea2e"),
				number.FieldMessageFlowID: uuid.FromStringOrNil("b1b4c734-20b0-11ee-9097-7f486b745239"),
			},

			responseNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b14ed168-20b0-11ee-b635-cf0e0e6774ba"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberUpdate(ctx, tt.id, tt.fields).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, tt.id).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseNumber.CustomerID, number.EventTypeNumberUpdated, tt.responseNumber)
			res, err := h.dbUpdate(ctx, tt.id, tt.fields, number.EventTypeNumberUpdated)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}
