package requesthandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestNMNumberFlowDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueCall:     "bin-manager.call-manager.request",
		queueFlow:     "bin-manager.flow-manager.request",
		queueNumber:   "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		flowID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("19d3cb88-7d72-11eb-84a7-d3b58b91c0d9"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/number_flows/19d3cb88-7d72-11eb-84a7-d3b58b91c0d9",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.NMNumberFlowDelete(tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
