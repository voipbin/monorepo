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

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseProvider *route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("a5ca3dc4-4658-11ed-8ecc-e35285c03d5c"),

			&route.Route{
				ID: uuid.FromStringOrNil("a5ca3dc4-4658-11ed-8ecc-e35285c03d5c"),
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

			mockDB.EXPECT().RouteGet(ctx, tt.id).Return(tt.responseProvider, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseProvider *route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("502369f4-4662-11ed-8f0a-aff10c31ed97"),
			"test name",
			"test detail",
			uuid.FromStringOrNil("505e0a96-4662-11ed-90ea-6b19829a782d"),
			1,
			"+82",

			&route.Route{
				ID: uuid.FromStringOrNil("5080cce8-4662-11ed-afd9-23fe95b1decc"),
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

			mockDB.EXPECT().RouteCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().RouteGet(ctx, gomock.Any()).Return(tt.responseProvider, nil)
			mockNotify.EXPECT().PublishEvent(ctx, route.EventTypeRouteCreated, tt.responseProvider)

			res, err := h.Create(ctx, tt.customerID, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		token      string
		limit      uint64

		responseRoutes []*route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("b4ff0a22-4662-11ed-bba2-dfe5060382ff"),
			"2020-04-18 03:22:17.995000",
			10,

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("bc61a324-4662-11ed-ad84-a750c5ec0ee4"),
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

			filters := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
			}
			mockDB.EXPECT().RouteGets(ctx, tt.token, tt.limit, filters).Return(tt.responseRoutes, nil)

			res, err := h.GetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoutes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoutes, res)
			}
		})
	}
}

func Test_GetsByCustomerID_customer_id_is_nil(t *testing.T) {

	tests := []struct {
		name string

		token string
		limit uint64

		responseRoutes []*route.Route
	}{
		{
			"normal",

			"2020-04-18 03:22:17.995000",
			10,

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("e1c2857c-680b-11ee-a7bb-d7acc274bf0f"),
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

			filters := map[route.Field]any{}
			mockDB.EXPECT().RouteGets(ctx, tt.token, tt.limit, filters).Return(tt.responseRoutes, nil)

			res, err := h.GetsByCustomerID(ctx, uuid.Nil, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoutes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoutes, res)
			}
		})
	}
}

func Test_GetsByTarget(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		target     string

		responseRoutesCustomer []*route.Route
		responseRoutesAll      []*route.Route

		expectRes []*route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("dea3392a-4662-11ed-a613-efc9ea475e11"),
			"+82",

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("e7624952-4662-11ed-9537-f37db6525b59"),
					ProviderID: uuid.FromStringOrNil("355d274e-4663-11ed-b877-4b5e771328dc"),
				},
			},
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("10125400-4663-11ed-97ab-b75f92cddce4"),
					ProviderID: uuid.FromStringOrNil("3581b6e0-4663-11ed-beb1-875b28f9db42"),
				},
			},

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("e7624952-4662-11ed-9537-f37db6525b59"),
					ProviderID: uuid.FromStringOrNil("355d274e-4663-11ed-b877-4b5e771328dc"),
				},
				{
					ID:         uuid.FromStringOrNil("10125400-4663-11ed-97ab-b75f92cddce4"),
					ProviderID: uuid.FromStringOrNil("3581b6e0-4663-11ed-beb1-875b28f9db42"),
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

			filtersTarget := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
				route.FieldTarget:     tt.target,
			}
			filtersAll := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
				route.FieldTarget:     route.TargetAll,
			}
			mockDB.EXPECT().RouteGets(ctx, "", uint64(1000), filtersTarget).Return(tt.responseRoutesCustomer, nil)
			mockDB.EXPECT().RouteGets(ctx, "", uint64(1000), filtersAll).Return(tt.responseRoutesAll, nil)

			res, err := h.GetsByTarget(ctx, tt.customerID, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseRoute *route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("52c2a598-4663-11ed-b6a2-93dcd490c49c"),

			&route.Route{
				ID: uuid.FromStringOrNil("52c2a598-4663-11ed-b6a2-93dcd490c49c"),
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

			mockDB.EXPECT().RouteDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().RouteGet(ctx, tt.id).Return(tt.responseRoute, nil)
			mockNotify.EXPECT().PublishEvent(ctx, route.EventTypeRouteDeleted, tt.responseRoute)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoute) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoute, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("76eec186-4663-11ed-b7b4-57471964d4f5"),
			"update name",
			"update detail",
			uuid.FromStringOrNil("771ba87c-4663-11ed-bc0e-2ba6cb69d485"),
			1,
			"+82",

			&route.Route{
				ID: uuid.FromStringOrNil("76eec186-4663-11ed-b7b4-57471964d4f5"),
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

			fields := map[route.Field]any{
				route.FieldName:       tt.routeName,
				route.FieldDetail:     tt.detail,
				route.FieldProviderID: tt.providerID,
				route.FieldPriority:   tt.priority,
				route.FieldTarget:     tt.target,
			}
			mockDB.EXPECT().RouteUpdate(ctx, tt.id, fields).Return(nil)
			mockDB.EXPECT().RouteGet(ctx, tt.id).Return(tt.responseRoute, nil)
			mockNotify.EXPECT().PublishEvent(ctx, route.EventTypeRouteUpdated, tt.responseRoute)

			res, err := h.Update(ctx, tt.id, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoute) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoute, res)
			}
		})
	}
}
