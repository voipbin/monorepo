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

func Test_v1DialroutesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		target     string

		responseDialroutes []*route.Route

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/dialroutes?customer_id=ad06dadc-9694-4179-920c-d0bbaf6bedc3&target=%2b82",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("ad06dadc-9694-4179-920c-d0bbaf6bedc3"),
			"+82",

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("79f2705e-b57f-4957-8ad6-6e162802b115"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"79f2705e-b57f-4957-8ad6-6e162802b115","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:      "/v1/dialroutes?customer_id=555a5772-517a-45fa-b489-c0104dc0b993&target=%2b82",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("555a5772-517a-45fa-b489-c0104dc0b993"),
			"+82",

			[]*route.Route{
				{
					ID: uuid.FromStringOrNil("ea23a015-6d11-4014-894d-2aaa96cbd851"),
				},
				{
					ID: uuid.FromStringOrNil("3bb96c6e-1526-4628-806a-dc780b43a82a"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ea23a015-6d11-4014-894d-2aaa96cbd851","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"3bb96c6e-1526-4628-806a-dc780b43a82a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_id":"00000000-0000-0000-0000-000000000000","priority":0,"target":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty response",
			&sock.Request{
				URI:      "/v1/dialroutes?customer_id=d66690be-777b-4cb4-8419-9334ceb57bd8&target=%2b82",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("d66690be-777b-4cb4-8419-9334ceb57bd8"),
			"+82",

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

			mockRoute.EXPECT().DialrouteGets(gomock.Any(), tt.customerID, tt.target.Return(tt.responseDialroutes, nil)

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
