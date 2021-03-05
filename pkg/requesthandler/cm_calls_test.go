package requesthandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/requesthandler/models/cmcall"
)

func TestCMCallAddChainedCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueCall:     "bin-manager.call-manager.request",
		queueFlow:     "bin-manager.flow-manager.request",
	}

	type test struct {
		name string

		callID        uuid.UUID
		chainedCallID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		// expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("887a7600-25c9-11eb-ab60-338d7ef0ba0f"),
			uuid.FromStringOrNil("8d48ded8-25c9-11eb-a8da-a7bcaada697c"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/887a7600-25c9-11eb-ab60-338d7ef0ba0f/chained-call-ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"chained_call_id":"8d48ded8-25c9-11eb-a8da-a7bcaada697c"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CMCallAddChainedCall(tt.callID, tt.chainedCallID); err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}

func TestCMCallHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueCall:     "bin-manager.call-manager.request",
		queueFlow:     "bin-manager.flow-manager.request",
	}

	type test struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa0ddb32-25cd-11eb-a604-8b239b305055","user_id":1,"asterisk_id":"","channel_id":"","flow_id":"59518eae-ed66-11ea-85ef-b77bdbc74ccc","conf_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"source":{"type":"","target":"","name":""},"destination":{"type":"","target":"","name":""},"status":"","data":null,"action":{"id":"00000000-0000-0000-0000-000000000000","type":"","tm_execute":""},"direction":"","hangup_by":"","hangup_reason":"","tm_create":"","tm_update":"","tm_progressing":"","tm_ringing":"","tm_hangup":""}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/calls/fa0ddb32-25cd-11eb-a604-8b239b305055",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},
			&cmcall.Call{
				ID:          uuid.FromStringOrNil("fa0ddb32-25cd-11eb-a604-8b239b305055"),
				UserID:      1,
				FlowID:      uuid.FromStringOrNil("59518eae-ed66-11ea-85ef-b77bdbc74ccc"),
				Source:      cmcall.Address{},
				Destination: cmcall.Address{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMCallHangup(tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
