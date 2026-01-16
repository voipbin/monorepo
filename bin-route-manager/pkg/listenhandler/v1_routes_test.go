package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

func Test_v1RoutesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *route.Route
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/routes",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5ec129cc-4867-11ed-ac2b-fb6ace3e2e29","name":"test name","detail":"test detail","provider_id": "5eea5d92-4867-11ed-a013-bbe5c1f759ec", "priority": 1, "target": "+82"}`),
			},

			uuid.FromStringOrNil("5ec129cc-4867-11ed-ac2b-fb6ace3e2e29"),
			"test name",
			"test detail",
			uuid.FromStringOrNil("5eea5d92-4867-11ed-a013-bbe5c1f759ec"),
			1,
			"+82",

			&route.Route{
				ID: uuid.FromStringOrNil("ccb0ceec-4867-11ed-8efb-fb670e6abe45"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ccb0ceec-4867-11ed-8efb-fb670e6abe45","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockRoute.EXPECT().Create(gomock.Any(), tt.customerID, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target).Return(tt.responseRoute, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1RoutesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		responseRoutes []*route.Route

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/routes?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"e2d763cc-486a-11ed-8d55-e34391ad9311"}`),
			},

			uuid.FromStringOrNil("e2d763cc-486a-11ed-8d55-e34391ad9311"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("6af1adee-486b-11ed-abce-07169e2f9488"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6af1adee-486b-11ed-abce-07169e2f9488","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:      "/v1/routes?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"4d4b055a-486c-11ed-9e9a-8f788b18b04c"}`),
			},

			uuid.FromStringOrNil("4d4b055a-486c-11ed-9e9a-8f788b18b04c"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("4d88b648-486c-11ed-be0b-1b6f5fbb7ada"),
				},
				{
					ID: uuid.FromStringOrNil("4db63b68-486c-11ed-914b-e7864023bf96"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4d88b648-486c-11ed-be0b-1b6f5fbb7ada","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"4db63b68-486c-11ed-914b-e7864023bf96","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty response",
			&sock.Request{
				URI:      "/v1/routes?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"7a484950-486c-11ed-8224-b3c5db4f575e"}`),
			},

			uuid.FromStringOrNil("7a484950-486c-11ed-8224-b3c5db4f575e"),
			"2020-10-10T03:30:17.000000",
			10,

			[]*route.Route{},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockRoute.EXPECT().ListByCustomerID(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.responseRoutes, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1RoutesIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		routeID uuid.UUID

		responseRoute *route.Route

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/routes/15f39396-486d-11ed-b993-9fc71f6dfd8f",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("15f39396-486d-11ed-b993-9fc71f6dfd8f"),

			&route.Route{
				ID: uuid.FromStringOrNil("15f39396-486d-11ed-b993-9fc71f6dfd8f"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"15f39396-486d-11ed-b993-9fc71f6dfd8f","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockRoute.EXPECT().Get(gomock.Any(), tt.routeID).Return(tt.responseRoute, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1RoutesIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID

		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *route.Route
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/routes/a1f4bee0-486f-11ed-ae92-336bd3b7e9e0",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","provider_id":"0bf6c090-4870-11ed-bba4-6f6a8a7d8553", "priority": 1, "target": "+82"}`),
			},
			uuid.FromStringOrNil("a1f4bee0-486f-11ed-ae92-336bd3b7e9e0"),

			"update name",
			"update detail",
			uuid.FromStringOrNil("0bf6c090-4870-11ed-bba4-6f6a8a7d8553"),
			1,
			"+82",

			&route.Route{
				ID: uuid.FromStringOrNil("a1f4bee0-486f-11ed-ae92-336bd3b7e9e0"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a1f4bee0-486f-11ed-ae92-336bd3b7e9e0","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockRoute.EXPECT().Update(gomock.Any(), tt.id, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target).Return(tt.responseRoute, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_v1RoutesIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		routeID uuid.UUID

		responseRoute *route.Route
		expectRes     *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/routes/39e88472-486e-11ed-baee-8b0aad96ce8f",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("39e88472-486e-11ed-baee-8b0aad96ce8f"),

			&route.Route{
				ID: uuid.FromStringOrNil("39e88472-486e-11ed-baee-8b0aad96ce8f"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"39e88472-486e-11ed-baee-8b0aad96ce8f","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockRoute.EXPECT().Delete(gomock.Any(), tt.routeID).Return(tt.responseRoute, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
