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

func Test_DialrouteV1RouteGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		target     string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("177ca524-52b6-11ed-bc27-67e42188fe83"),
			"+82",

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
			Data:     []byte(`{"customer_id":"177ca524-52b6-11ed-bc27-67e42188fe83","target":"+82"}`),
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
		filters := map[rmroute.Field]any{
			rmroute.FieldCustomerID: tt.customerID,
			rmroute.FieldTarget: tt.target,
		}

		res, err := reqHandler.RouteV1DialrouteGets(ctx, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
