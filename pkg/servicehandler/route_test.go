package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_RouteGet(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID

		responseRoute *rmroute.Route
		expectRes     *rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			uuid.FromStringOrNil("19dd98af-0e61-4735-909f-e0da0873ef44"),

			&rmroute.Route{
				ID:         uuid.FromStringOrNil("19dd98af-0e61-4735-909f-e0da0873ef44"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&rmroute.WebhookMessage{
				ID:         uuid.FromStringOrNil("19dd98af-0e61-4735-909f-e0da0873ef44"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteGet(ctx, tt.id).Return(tt.responseRoute, nil)

			res, err := h.RouteGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteGets(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer

		pageToken string
		pageSize  uint64

		responseRoutes []rmroute.Route
		expectRes      []*rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},

			"2021-03-01 01:00:00.995000",
			10,

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("f65b0310-68a1-11ee-8c62-73e88f334b47"),
				},
			},
			[]*rmroute.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("f65b0310-68a1-11ee-8c62-73e88f334b47"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteGets(ctx, tt.pageToken, tt.pageSize).Return(tt.responseRoutes, nil)

			res, err := h.RouteGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteGetsByCustomerID(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		responseRoutes []rmroute.Route
		expectRes      []*rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},

			uuid.FromStringOrNil("3ebe976f-ecca-436a-a2d3-bc0c75501882"),
			"2021-03-01 01:00:00.995000",
			10,

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("99a7ea66-d257-4b5c-8be3-47ddd6373c95"),
				},
			},
			[]*rmroute.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("99a7ea66-d257-4b5c-8be3-47ddd6373c95"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize).Return(tt.responseRoutes, nil)

			res, err := h.RouteGetsByCustomerID(ctx, tt.customer, tt.customerID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteCreate(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer

		customerID uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *rmroute.Route
		expectRes     *rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},

			uuid.FromStringOrNil("cf7339a3-fb3b-44ff-aedd-2b999f89fd7b"),
			"test name",
			"test detail",
			uuid.FromStringOrNil("bfe600b7-e496-4c00-84e4-e9ae05e7b829"),
			1,
			"+82",

			&rmroute.Route{
				ID: uuid.FromStringOrNil("5bbbe36b-ec7b-480d-8bb8-28dc43328269"),
			},
			&rmroute.WebhookMessage{
				ID: uuid.FromStringOrNil("5bbbe36b-ec7b-480d-8bb8-28dc43328269"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteCreate(
				ctx,
				tt.customerID,
				tt.routeName,
				tt.detail,
				tt.providerID,
				tt.priority,
				tt.target,
			).Return(tt.responseRoute, nil)

			res, err := h.RouteCreate(
				ctx,
				tt.customer,
				tt.customerID,
				tt.routeName,
				tt.detail,
				tt.providerID,
				tt.priority,
				tt.target,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteDelete(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		routeID  uuid.UUID

		responseRoute *rmroute.Route
		expectRes     *rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},
			uuid.FromStringOrNil("15700708-0f25-4d46-b72e-1d489abc2cea"),

			&rmroute.Route{
				ID:         uuid.FromStringOrNil("15700708-0f25-4d46-b72e-1d489abc2cea"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&rmroute.WebhookMessage{
				ID:         uuid.FromStringOrNil("15700708-0f25-4d46-b72e-1d489abc2cea"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteGet(ctx, tt.routeID).Return(tt.responseRoute, nil)
			mockReq.EXPECT().RouteV1RouteDelete(ctx, tt.routeID).Return(tt.responseRoute, nil)

			res, err := h.RouteDelete(ctx, tt.customer, tt.routeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteUpdate(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer

		routeID    uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *rmroute.Route
		expectRes     *rmroute.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				PermissionIDs: []uuid.UUID{cspermission.PermissionAdmin.ID},
			},

			uuid.FromStringOrNil("88c8938c-8dd3-4fcf-887f-c0e026912a6b"),
			"update name",
			"update detail",
			uuid.FromStringOrNil("902f912c-57bb-45eb-ac68-10c16057aebb"),
			1,
			"+82",

			&rmroute.Route{
				ID:         uuid.FromStringOrNil("88c8938c-8dd3-4fcf-887f-c0e026912a6b"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&rmroute.WebhookMessage{
				ID:         uuid.FromStringOrNil("88c8938c-8dd3-4fcf-887f-c0e026912a6b"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1RouteGet(ctx, tt.routeID).Return(tt.responseRoute, nil)
			mockReq.EXPECT().RouteV1RouteUpdate(
				ctx,
				tt.routeID,
				tt.routeName,
				tt.detail,
				tt.providerID,
				tt.priority,
				tt.target,
			).Return(tt.responseRoute, nil)

			res, err := h.RouteUpdate(
				ctx,
				tt.customer,
				tt.routeID,
				tt.routeName,
				tt.detail,
				tt.providerID,
				tt.priority,
				tt.target,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
