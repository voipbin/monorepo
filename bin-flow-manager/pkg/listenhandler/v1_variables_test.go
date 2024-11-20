package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_v1VariablesIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		variableID uuid.UUID

		responseVariable *variable.Variable

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/variables/0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68",
				Method:   sock.RequestMethodGet,
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
			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
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
		request *sock.Request

		variableID uuid.UUID
		variables  map[string]string

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/variables/f842de3c-ccfd-11ec-bfcb-670259cb01f7/variables",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"variables":{"key1": "value1", "key2": "value2"}}`),
			},

			uuid.FromStringOrNil("f842de3c-ccfd-11ec-bfcb-670259cb01f7"),
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
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
		request *sock.Request

		variableID uuid.UUID
		key        string

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			"key1",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"key has a space",
			&sock.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key+1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			"key 1",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
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
