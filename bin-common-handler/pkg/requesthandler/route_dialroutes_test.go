package requesthandler

import (
	"context"
	"reflect"
	"testing"

	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_DialrouteV1RouteList(t *testing.T) {

	tests := []struct {
		name string

		filters           map[rmroute.Field]any
		targetProviderIDs []uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmroute.Route
	}{
		{
			"normal",

			map[rmroute.Field]any{
				rmroute.FieldCustomerID: uuid.FromStringOrNil("177ca524-52b6-11ed-bc27-67e42188fe83"),
				rmroute.FieldTarget:     "+82",
			},
			nil,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8297fc2c-52b7-11ed-b257-2b1bd9fe3671"}]`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/dialroutes",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"filters":{"customer_id":"177ca524-52b6-11ed-bc27-67e42188fe83","target":"+82"}}`),
			},
			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("8297fc2c-52b7-11ed-b257-2b1bd9fe3671"),
				},
			},
		},
		{
			"with target provider ids",

			map[rmroute.Field]any{
				rmroute.FieldCustomerID: uuid.FromStringOrNil("177ca524-52b6-11ed-bc27-67e42188fe83"),
				rmroute.FieldTarget:     "+82",
			},
			[]uuid.UUID{
				uuid.FromStringOrNil("9a6d2f7e-52b8-11ed-9b1a-cb4e7dfe0001"),
				uuid.FromStringOrNil("9a6d2f7e-52b8-11ed-9b1a-cb4e7dfe0002"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8297fc2c-52b7-11ed-b257-2b1bd9fe3671"}]`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/dialroutes",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"filters":{"customer_id":"177ca524-52b6-11ed-bc27-67e42188fe83","target":"+82"},"target_provider_ids":["9a6d2f7e-52b8-11ed-9b1a-cb4e7dfe0001","9a6d2f7e-52b8-11ed-9b1a-cb4e7dfe0002"]}`),
			},
			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("8297fc2c-52b7-11ed-b257-2b1bd9fe3671"),
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

			res, err := reqHandler.RouteV1DialrouteList(ctx, tt.filters, tt.targetProviderIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
