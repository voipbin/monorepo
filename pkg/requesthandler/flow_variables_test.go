package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_FlowV1VariableGet(t *testing.T) {

	tests := []struct {
		name string

		variableID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *variable.Variable
	}{
		{
			"normal",

			uuid.FromStringOrNil("e25aeb10-cd06-11ec-baba-fb2f8b96ad65"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e25aeb10-cd06-11ec-baba-fb2f8b96ad65","variables":{"key 1": "value 1"}}`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/e25aeb10-cd06-11ec-baba-fb2f8b96ad65",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("e25aeb10-cd06-11ec-baba-fb2f8b96ad65"),
				Variables: map[string]string{
					"key 1": "value 1",
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

			res, err := reqHandler.FlowV1VariableGet(ctx, tt.variableID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_FlowV1VariableSetVariable(t *testing.T) {

	tests := []struct {
		name string

		variableID uuid.UUID
		variables  map[string]string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			"normal",

			uuid.FromStringOrNil("4d3c129c-cd07-11ec-bd2f-2fcee708f983"),
			map[string]string{
				"key 1": "value 1",
				"key 2": "value 2",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/4d3c129c-cd07-11ec-bd2f-2fcee708f983/variables",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"variables":{"key 1":"value 1","key 2":"value 2"}}`),
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

			if err := reqHandler.FlowV1VariableSetVariable(ctx, tt.variableID, tt.variables); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_FlowV1VariableDeleteVariable(t *testing.T) {

	tests := []struct {
		name string

		variableID uuid.UUID
		key        string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			"normal",

			uuid.FromStringOrNil("290c673e-db33-11ec-a4d9-bb00659a2a19"),
			"key1",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/290c673e-db33-11ec-a4d9-bb00659a2a19/variables/key1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
		},
		{
			"key has a space",

			uuid.FromStringOrNil("290c673e-db33-11ec-a4d9-bb00659a2a19"),
			"key 1",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/290c673e-db33-11ec-a4d9-bb00659a2a19/variables/key+1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
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

			if err := reqHandler.FlowV1VariableDeleteVariable(ctx, tt.variableID, tt.key); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
