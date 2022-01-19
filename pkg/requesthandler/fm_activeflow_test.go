package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestFMV1ActvieFlowCreate(t *testing.T) {
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

	tests := []struct {
		name   string
		callID uuid.UUID
		flowID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *fmactiveflow.ActiveFlow
	}{
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
			&fmactiveflow.ActiveFlow{
				CallID: uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
				FlowID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
				UserID: 0,
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
				Actions: []fmaction.Action{},
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
			&fmactiveflow.ActiveFlow{
				CallID:     uuid.FromStringOrNil("ec603bf8-82dc-11eb-afe2-d7c97817ab6f"),
				FlowID:     uuid.FromStringOrNil("f2c8b826-82dc-11eb-a8f0-e7519b0418a6"),
				UserID:     0,
				WebhookURI: "https://test.com/test_webhook",
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
				Actions: []fmaction.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMV1ActvieFlowCreate(context.Background(), tt.callID, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestFMV1ActvieFlowGetNextAction(t *testing.T) {
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

	tests := []struct {
		name string

		callID          uuid.UUID
		currentActionID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *fmaction.Action
	}{
		{
			"normal",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/447e712e-82d8-11eb-8900-7b97c080ddd8/next",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"current_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"e52c5766-57c7-11ec-836b-333ce17a1ce6","type":"answer"}`),
			},
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("e52c5766-57c7-11ec-836b-333ce17a1ce6"),
				Type: fmaction.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMV1ActvieFlowGetNextAction(context.Background(), tt.callID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestFMV1ActvieFlowUpdateForwardActionID(t *testing.T) {
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

	tests := []struct {
		name string

		callID          uuid.UUID
		forwardActionID uuid.UUID
		forwardNow      bool

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
			true,

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"forward_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","forward_now":true}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},
		},
		{
			"forward now false",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
			false,

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"forward_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","forward_now":false}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.FMV1ActvieFlowUpdateForwardActionID(context.Background(), tt.callID, tt.forwardActionID, tt.forwardNow); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}
