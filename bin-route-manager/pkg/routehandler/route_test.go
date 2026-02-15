package routehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

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

func Test_ListByCustomerID(t *testing.T) {

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
			"2020-04-18T03:22:17.995000Z",
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
			mockDB.EXPECT().RouteList(ctx, tt.token, tt.limit, filters).Return(tt.responseRoutes, nil)

			res, err := h.ListByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoutes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoutes, res)
			}
		})
	}
}

func Test_ListByCustomerID_customer_id_is_nil(t *testing.T) {

	tests := []struct {
		name string

		token string
		limit uint64

		responseRoutes []*route.Route
	}{
		{
			"normal",

			"2020-04-18T03:22:17.995000Z",
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
			mockDB.EXPECT().RouteList(ctx, tt.token, tt.limit, filters).Return(tt.responseRoutes, nil)

			res, err := h.ListByCustomerID(ctx, uuid.Nil, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRoutes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseRoutes, res)
			}
		})
	}
}

func Test_ListByTarget(t *testing.T) {

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
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersTarget).Return(tt.responseRoutesCustomer, nil)
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersAll).Return(tt.responseRoutesAll, nil)

			res, err := h.ListByTarget(ctx, tt.customerID, tt.target)
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

func Test_Get_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("a5ca3dc4-4658-11ed-8ecc-e35285c03d5c")

	mockDB.EXPECT().RouteGet(ctx, id).Return(nil, fmt.Errorf("database error"))

	res, err := h.Get(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Create_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("502369f4-4662-11ed-8f0a-aff10c31ed97")
	providerID := uuid.FromStringOrNil("505e0a96-4662-11ed-90ea-6b19829a782d")

	mockDB.EXPECT().RouteCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	res, err := h.Create(ctx, customerID, "name", "detail", providerID, 1, "+82")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_ListByCustomerID_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("b4ff0a22-4662-11ed-bba2-dfe5060382ff")
	filters := map[route.Field]any{
		route.FieldCustomerID: customerID,
	}

	mockDB.EXPECT().RouteList(ctx, "", uint64(10), filters).Return(nil, fmt.Errorf("database error"))

	res, err := h.ListByCustomerID(ctx, customerID, "", 10)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_ListByTarget_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("dea3392a-4662-11ed-a613-efc9ea475e11")

	filtersTarget := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}

	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersTarget).Return(nil, fmt.Errorf("database error"))

	res, err := h.ListByTarget(ctx, customerID, "+82")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Delete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("52c2a598-4663-11ed-b6a2-93dcd490c49c")

	mockDB.EXPECT().RouteDelete(ctx, id).Return(fmt.Errorf("database error"))

	res, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Update_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("76eec186-4663-11ed-b7b4-57471964d4f5")
	providerID := uuid.FromStringOrNil("771ba87c-4663-11ed-bc0e-2ba6cb69d485")

	mockDB.EXPECT().RouteUpdate(ctx, id, gomock.Any()).Return(fmt.Errorf("database error"))

	res, err := h.Update(ctx, id, "name", "detail", providerID, 1, "+82")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Create_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("502369f4-4662-11ed-8f0a-aff10c31ed97")
	providerID := uuid.FromStringOrNil("505e0a96-4662-11ed-90ea-6b19829a782d")

	mockDB.EXPECT().RouteCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().RouteGet(ctx, gomock.Any()).Return(nil, fmt.Errorf("get error"))

	res, err := h.Create(ctx, customerID, "name", "detail", providerID, 1, "+82")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Delete_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("52c2a598-4663-11ed-b6a2-93dcd490c49c")

	mockDB.EXPECT().RouteDelete(ctx, id).Return(nil)
	mockDB.EXPECT().RouteGet(ctx, id).Return(nil, fmt.Errorf("get error"))

	res, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Update_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("76eec186-4663-11ed-b7b4-57471964d4f5")
	providerID := uuid.FromStringOrNil("771ba87c-4663-11ed-bc0e-2ba6cb69d485")

	mockDB.EXPECT().RouteUpdate(ctx, id, gomock.Any()).Return(nil)
	mockDB.EXPECT().RouteGet(ctx, id).Return(nil, fmt.Errorf("get error"))

	res, err := h.Update(ctx, id, "name", "detail", providerID, 1, "+82")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_ListByTarget_AllRoutesError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("dea3392a-4662-11ed-a613-efc9ea475e11")

	filtersTarget := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}
	filtersAll := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}

	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersTarget).Return([]*route.Route{}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersAll).Return(nil, fmt.Errorf("database error"))

	res, err := h.ListByTarget(ctx, customerID, "+82")
	if err != nil {
		t.Errorf("Expected no error (error is ignored), got %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Expected empty result, got %v items", len(res))
	}
}

func Test_NewRouteHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewRouteHandler(mockDB, mockReq, mockNotify)
	if h == nil {
		t.Errorf("Expected handler to be created, got nil")
	}
}
