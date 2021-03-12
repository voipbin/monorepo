package requesthandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestFlowActvieFlowPost(t *testing.T) {

	type test struct {
		name   string
		callID uuid.UUID
		flowID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *activeflow.ActiveFlow
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"447e712e-82d8-11eb-8900-7b97c080ddd8","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"call_id":"447e712e-82d8-11eb-8900-7b97c080ddd8","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","user_id":0,"webhook_uri":"","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&activeflow.ActiveFlow{
				CallID: uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
				FlowID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
				UserID: 0,
				CurrentAction: action.Action{
					ID: action.IDBegin,
				},
				Actions: []action.Action{},
			},
		},
		{
			"webhook",
			uuid.FromStringOrNil("ec603bf8-82dc-11eb-afe2-d7c97817ab6f"),
			uuid.FromStringOrNil("f2c8b826-82dc-11eb-a8f0-e7519b0418a6"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"ec603bf8-82dc-11eb-afe2-d7c97817ab6f","flow_id":"f2c8b826-82dc-11eb-a8f0-e7519b0418a6"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"call_id":"ec603bf8-82dc-11eb-afe2-d7c97817ab6f","flow_id":"f2c8b826-82dc-11eb-a8f0-e7519b0418a6","user_id":0,"webhook_uri":"https://test.com/test_webhook","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&activeflow.ActiveFlow{
				CallID:     uuid.FromStringOrNil("ec603bf8-82dc-11eb-afe2-d7c97817ab6f"),
				FlowID:     uuid.FromStringOrNil("f2c8b826-82dc-11eb-a8f0-e7519b0418a6"),
				UserID:     0,
				WebhookURI: "https://test.com/test_webhook",
				CurrentAction: action.Action{
					ID: action.IDBegin,
				},
				Actions: []action.Action{},
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowActvieFlowPost(tt.callID, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
