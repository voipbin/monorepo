package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_v1VariablesIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		variableID uuid.UUID

		responseVariable *variable.Variable

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			uuid.FromStringOrNil("0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68"),

			&variable.Variable{
				ID: uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
				Variables: map[string]string{
					"key1": "val1",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"01677a56-0c2d-11eb-96cb-eb2cd309ca81","variables":{"key1":"val1"}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				variableHandler: mockVariableHandler,
			}

			mockVariableHandler.EXPECT().Get(gomock.Any(), tt.variableID).Return(tt.responseVariable, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1VariablesIDVariablesPost(t *testing.T) {
	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		variableID uuid.UUID
		variables  map[string]string

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/f842de3c-ccfd-11ec-bfcb-670259cb01f7/variables",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"variables":{"key1": "value1", "key2": "value2"}}`),
			},

			uuid.FromStringOrNil("f842de3c-ccfd-11ec-bfcb-670259cb01f7"),
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				variableHandler: mockVariableHandler,
			}

			mockVariableHandler.EXPECT().SetVariable(gomock.Any(), tt.variableID, tt.variables).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1VariablesIDVariablesKeyDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		variableID uuid.UUID
		key        string

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			"key1",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"key has a space",
			&rabbitmqhandler.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key+1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			"key 1",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				variableHandler: mockVariableHandler,
			}

			mockVariableHandler.EXPECT().DeleteVariable(gomock.Any(), tt.variableID, tt.key).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
