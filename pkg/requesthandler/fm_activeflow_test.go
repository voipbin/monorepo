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
		sock: mockSock,
	}

	tests := []struct {
		name string

		referenceType fmactiveflow.ReferenceType
		referenceID   uuid.UUID
		flowID        uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *fmactiveflow.ActiveFlow
	}{
		{
			"type call",

			fmactiveflow.ReferenceTypeCall,
			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"4a93277e-a7b4-11ec-860a-cbb5f65b6655","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&fmactiveflow.ActiveFlow{
				ID:            uuid.FromStringOrNil("4a93277e-a7b4-11ec-860a-cbb5f65b6655"),
				ReferenceType: fmactiveflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
				FlowID:        uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
				CustomerID:    uuid.FromStringOrNil("f42b33e2-7f4d-11ec-8c86-ebf558a4306c"),
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
				Actions: []fmaction.Action{},
			},
		},
		{
			"type message",

			fmactiveflow.ReferenceTypeMessage,
			uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
			uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"a95537d8-a7b5-11ec-b63c-57c0410e1ca4","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&fmactiveflow.ActiveFlow{
				ID:            uuid.FromStringOrNil("a95537d8-a7b5-11ec-b63c-57c0410e1ca4"),
				ReferenceType: fmactiveflow.ReferenceTypeMessage,
				ReferenceID:   uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
				FlowID:        uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),
				CustomerID:    uuid.FromStringOrNil("f42b33e2-7f4d-11ec-8c86-ebf558a4306c"),
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

			res, err := reqHandler.FMV1ActvieFlowCreate(context.Background(), tt.flowID, tt.referenceType, tt.referenceID)
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
		sock: mockSock,
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
		sock: mockSock,
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

func Test_FMV1ActvieFlowExecute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	tests := []struct {
		name string

		id uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("fde4653a-a7b5-11ec-a7ae-83d2f5255ec0"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/active-flows/fde4653a-a7b5-11ec-a7ae-83d2f5255ec0/execute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
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

			if err := reqHandler.FMV1ActiveFlowExecute(context.Background(), tt.id); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}
