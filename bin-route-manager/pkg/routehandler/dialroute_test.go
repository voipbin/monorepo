package routehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_DialrouteGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		target     string

		responseRoutesCustomerTarget []*route.Route
		responseRoutesCustomerAll    []*route.Route

		responseRoutesDefaultTarget []*route.Route
		responseRoutesDefaultAll    []*route.Route

		expectRes []*route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("ecfe86e4-e20f-4a06-890e-84b0e8ecfca4"),
			"+82",

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("fd05574d-a43d-4d74-a9a0-301e6e875bb9"),
					ProviderID: uuid.FromStringOrNil("9e5b4d7d-ddd3-4a10-9873-3fab21f86645"),
				},
			},
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("e1e6e844-4c84-4722-b1a7-26d43c193a26"),
					ProviderID: uuid.FromStringOrNil("f5a7c5b8-11b3-44c3-a87b-2224acecf8cf"),
				},
			},

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("ce97f95d-5498-4ae2-a9c8-ae2a2f598a93"),
					ProviderID: uuid.FromStringOrNil("44eea3bb-46b9-456e-8dbc-1c28bc01f308"),
				},
			},
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("0b71e856-f10e-40af-8a54-b05c2ae8bc81"),
					ProviderID: uuid.FromStringOrNil("0ed4e04a-9645-4002-b745-f11386af6305"),
				},
			},

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("fd05574d-a43d-4d74-a9a0-301e6e875bb9"),
					ProviderID: uuid.FromStringOrNil("9e5b4d7d-ddd3-4a10-9873-3fab21f86645"),
				},
				{
					ID:         uuid.FromStringOrNil("e1e6e844-4c84-4722-b1a7-26d43c193a26"),
					ProviderID: uuid.FromStringOrNil("f5a7c5b8-11b3-44c3-a87b-2224acecf8cf"),
				},
				{
					ID:         uuid.FromStringOrNil("ce97f95d-5498-4ae2-a9c8-ae2a2f598a93"),
					ProviderID: uuid.FromStringOrNil("44eea3bb-46b9-456e-8dbc-1c28bc01f308"),
				},
				{
					ID:         uuid.FromStringOrNil("0b71e856-f10e-40af-8a54-b05c2ae8bc81"),
					ProviderID: uuid.FromStringOrNil("0ed4e04a-9645-4002-b745-f11386af6305"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &routeHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// GetsByTarget for customer route base
			mockDB.EXPECT().RouteGetsByCustomerIDWithTarget(ctx, tt.customerID, tt.target).Return(tt.responseRoutesCustomerTarget, nil)
			mockDB.EXPECT().RouteGetsByCustomerIDWithTarget(ctx, tt.customerID, route.TargetAll).Return(tt.responseRoutesCustomerAll, nil)

			// GetsByTarget for default route base
			mockDB.EXPECT().RouteGetsByCustomerIDWithTarget(ctx, route.CustomerIDBasicRoute, tt.target).Return(tt.responseRoutesDefaultTarget, nil)
			mockDB.EXPECT().RouteGetsByCustomerIDWithTarget(ctx, route.CustomerIDBasicRoute, route.TargetAll).Return(tt.responseRoutesDefaultAll, nil)

			res, err := h.DialrouteGets(ctx, tt.customerID, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
