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

func TestFMV1ActiveflowCreate(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		referenceType fmactiveflow.ReferenceType
		referenceID   uuid.UUID
		flowID        uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *fmactiveflow.Activeflow
	}{
		{
			"type call",

			uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
			fmactiveflow.ReferenceTypeCall,
			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"aa847807-6cc4-4713-9dec-53a42840e74c","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"aa847807-6cc4-4713-9dec-53a42840e74c","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&fmactiveflow.Activeflow{
				ID:            uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
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

			uuid.FromStringOrNil("be2255b2-0e47-4db8-956a-2fb9f45417b8"),
			fmactiveflow.ReferenceTypeMessage,
			uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
			uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"be2255b2-0e47-4db8-956a-2fb9f45417b8","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"be2255b2-0e47-4db8-956a-2fb9f45417b8","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&fmactiveflow.Activeflow{
				ID:            uuid.FromStringOrNil("be2255b2-0e47-4db8-956a-2fb9f45417b8"),
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
		{
			"empty id",

			uuid.Nil,
			fmactiveflow.ReferenceTypeMessage,
			uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
			uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"00000000-0000-0000-0000-000000000000","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&fmactiveflow.Activeflow{
				ID:            uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)
			res, err := reqHandler.FMV1ActiveflowCreate(ctx, tt.id, tt.flowID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestFMV1ActiveflowGetNextAction(t *testing.T) {
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
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/next",
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

			res, err := reqHandler.FMV1ActiveflowGetNextAction(context.Background(), tt.callID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func TestFMV1ActiveflowUpdateForwardActionID(t *testing.T) {
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
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
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
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
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

			if err := reqHandler.FMV1ActiveflowUpdateForwardActionID(context.Background(), tt.callID, tt.forwardActionID, tt.forwardNow); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}

func Test_FMV1ActiveflowExecute(t *testing.T) {
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
				URI:      "/v1/activeflows/fde4653a-a7b5-11ec-a7ae-83d2f5255ec0/execute",
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

			if err := reqHandler.FMV1ActiveflowExecute(context.Background(), tt.id); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}

func Test_FMV1ActiveflowDelete(t *testing.T) {
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

		response  *rabbitmqhandler.Response
		expectRes *fmactiveflow.Activeflow
	}{
		{
			"normal",

			uuid.FromStringOrNil("2f4bd474-ade1-11ec-9aca-83684de0c293"),

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/activeflows/2f4bd474-ade1-11ec-9aca-83684de0c293",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"2f4bd474-ade1-11ec-9aca-83684de0c293"}`),
			},
			&fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("2f4bd474-ade1-11ec-9aca-83684de0c293"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMV1ActiveflowDelete(context.Background(), tt.id)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
