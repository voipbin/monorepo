package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_DialrouteV1RouteGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		target     string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     []rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("177ca524-52b6-11ed-bc27-67e42188fe83"),
			"+82",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8297fc2c-52b7-11ed-b257-2b1bd9fe3671"}]`),
			},

			"bin-manager.route-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/dialroutes?customer_id=177ca524-52b6-11ed-bc27-67e42188fe83&target=%s", url.QueryEscape("+82")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RouteV1DialrouteGets(ctx, tt.customerID, tt.target)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
