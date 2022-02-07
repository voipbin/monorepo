package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
)

func TestProcessV1NumberFlowsDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name   string
		flowID uuid.UUID

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 number",
			uuid.FromStringOrNil("c5932f4c-7d25-11eb-861b-637cf710e77d"),
			&rabbitmqhandler.Request{
				URI:    "/v1/number_flows/c5932f4c-7d25-11eb-861b-637cf710e77d",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().RemoveNumbersFlowID(gomock.Any(), tt.flowID)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}
