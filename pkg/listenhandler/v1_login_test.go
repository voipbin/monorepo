package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/customerhandler"
)

func Test_ProcessV1LoginPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		rabbitSock:      mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	tests := []struct {
		name     string
		request  *rabbitmqhandler.Request
		username string
		password string

		customer  *customer.Customer
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/login",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"username":"test","password":"password"}`),
			},
			"test",
			"password",

			&customer.Customer{
				ID:            uuid.FromStringOrNil("e58a9424-7dc0-11ec-82b6-d387115f2157"),
				Username:      "test",
				PermissionIDs: []uuid.UUID{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e58a9424-7dc0-11ec-82b6-d387115f2157","username":"test"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCustomer.EXPECT().Login(gomock.Any(), tt.username, tt.password).Return(tt.customer, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
