package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RouteV1RouteCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("24d9f42d-0eb5-4276-aaf8-8df5a8342a3c"),
			"test name",
			"test detail",
			uuid.FromStringOrNil("3963772a-84ad-4a1b-a250-2b5d100f76ee"),
			1,
			"+82",

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/routes",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"24d9f42d-0eb5-4276-aaf8-8df5a8342a3c","name":"test name","detail":"test detail","provider_id":"3963772a-84ad-4a1b-a250-2b5d100f76ee","priority":1,"target":"+82"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6902a0cf-5367-4f4e-ab40-18b575f08666"}`),
			},
			&rmroute.Route{
				ID: uuid.FromStringOrNil("6902a0cf-5367-4f4e-ab40-18b575f08666"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RouteV1RouteCreate(ctx, tt.customerID, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1RouteGet(t *testing.T) {

	tests := []struct {
		name string

		routeID uuid.UUID

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("20e596b2-c7ea-4e88-bb7f-92ac5003c388"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"20e596b2-c7ea-4e88-bb7f-92ac5003c388"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/routes/20e596b2-c7ea-4e88-bb7f-92ac5003c388",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&rmroute.Route{
				ID: uuid.FromStringOrNil("20e596b2-c7ea-4e88-bb7f-92ac5003c388"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1RouteGet(ctx, tt.routeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1RouteDelete(t *testing.T) {

	tests := []struct {
		name string

		routeID uuid.UUID

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("eeda13db-aeb1-448b-bd86-cf64df8b36be"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"eeda13db-aeb1-448b-bd86-cf64df8b36be"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/routes/eeda13db-aeb1-448b-bd86-cf64df8b36be",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			&rmroute.Route{
				ID: uuid.FromStringOrNil("eeda13db-aeb1-448b-bd86-cf64df8b36be"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1RouteDelete(ctx, tt.routeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteV1RouteUpdate(t *testing.T) {

	tests := []struct {
		name string

		routeID    uuid.UUID
		routeName  string
		detail     string
		providerID uuid.UUID
		priority   int
		target     string

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("f417d043-981b-4b74-bb26-5e37771b3104"),
			"update name",
			"update detail",
			uuid.FromStringOrNil("1834094f-bebf-42b1-83d3-88b86f8d417c"),
			1,
			"+82",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f417d043-981b-4b74-bb26-5e37771b3104"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/routes/f417d043-981b-4b74-bb26-5e37771b3104",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","provider_id":"1834094f-bebf-42b1-83d3-88b86f8d417c","priority":1,"target":"+82"}`),
			},
			&rmroute.Route{
				ID: uuid.FromStringOrNil("f417d043-981b-4b74-bb26-5e37771b3104"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1RouteUpdate(ctx, tt.routeID, tt.routeName, tt.detail, tt.providerID, tt.priority, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1RouteGets_WithCustomerIDFilter(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("aee4503c-2657-41c9-8f20-5848173bcecf"),
			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"f6b8946a-7191-454d-9c16-7136071541b3"}]`),
			},

		"bin-manager.route-manager.request",
		&sock.Request{
			URI:      fmt.Sprintf("/v1/routes?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
			Method:   sock.RequestMethodGet,
			DataType: ContentTypeJSON,
			Data:     []byte(`{"customer_id":"aee4503c-2657-41c9-8f20-5848173bcecf"}`),
		},
		[]rmroute.Route{
			{
				ID: uuid.FromStringOrNil("f6b8946a-7191-454d-9c16-7136071541b3"),
			},
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
		filters := map[rmroute.Field]any{
			rmroute.FieldCustomerID: tt.customerID,
		}

		res, err := reqHandler.RouteV1RouteGets(ctx, tt.pageToken, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteV1RouteGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmroute.Route
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c5c30b12-682e-11ee-9727-578ef127932b"}]`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/routes?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
			DataType: ContentTypeJSON,
			Data:     []byte(`{}`),
			},
			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("c5c30b12-682e-11ee-9727-578ef127932b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
		filters := map[rmroute.Field]any{}

		res, err := reqHandler.RouteV1RouteGets(ctx, tt.pageToken, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
