package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CSV1Login(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			"test",
			"testpassword",

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"username":"test","password":"testpassword"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ed8088d8-7e41-11ec-958e-6b788edc7b1b","username":"test","name":"test user 1","detail":"test user 1 detail","permission_ids":[]}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("ed8088d8-7e41-11ec-958e-6b788edc7b1b"),
				Username:      "test",
				Name:          "test user 1",
				Detail:        "test user 1 detail",
				PermissionIDs: []uuid.UUID{},
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

			res, err := reqHandler.CSV1Login(ctx, requestTimeoutDefault, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
