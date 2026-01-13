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

		responseVariable *variable.Variable

		expectedVariableID uuid.UUID
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/variables/0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("01677a56-0c2d-11eb-96cb-eb2cd309ca81"),
				Variables: map[string]string{
					"key1": "val1",
				},
			},

			expectedVariableID: uuid.FromStringOrNil("0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68"),
			expectedRes: &sock.Response{
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

			mockVariableHandler.EXPECT().Get(gomock.Any(), tt.expectedVariableID.Return(tt.responseVariable, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1VariablesIDVariablesPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectedVariableID uuid.UUID
		expectedVariables  map[string]string
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/variables/f842de3c-ccfd-11ec-bfcb-670259cb01f7/variables",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"variables":{"key1": "value1", "key2": "value2"}}`),
			},

			expectedVariableID: uuid.FromStringOrNil("f842de3c-ccfd-11ec-bfcb-670259cb01f7"),
			expectedVariables: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectedRes: &sock.Response{
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

			mockVariableHandler.EXPECT().SetVariable(gomock.Any(), tt.expectedVariableID, tt.expectedVariables.Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1VariablesIDVariablesKeyDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectedVariableID uuid.UUID
		expectedKey        string

		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			expectedVariableID: uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			expectedKey:        "key1",
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			name: "key has a space",
			request: &sock.Request{
				URI:      "/v1/variables/52905588-db2f-11ec-9813-73dc3a5d302d/variables/key+1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			expectedVariableID: uuid.FromStringOrNil("52905588-db2f-11ec-9813-73dc3a5d302d"),
			expectedKey:        "key 1",
			expectedRes: &sock.Response{
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

			mockVariableHandler.EXPECT().DeleteVariable(gomock.Any(), tt.expectedVariableID, tt.expectedKey.Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_v1VariablesIDSubstitutePost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseData string

		expectedID   uuid.UUID
		expectedData string
		expectedRes  *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/variables/d13811e2-f791-11ef-8f37-33b45eb5dfae/substitute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"data":"test data string"}`),
			},

			responseData: "test response string",

			expectedID:   uuid.FromStringOrNil("d13811e2-f791-11ef-8f37-33b45eb5dfae"),
			expectedData: `test data string`,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"data":"test response string"}`),
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

			mockVariableHandler.EXPECT().Substitute(gomock.Any(), tt.expectedID, tt.expectedData.Return(tt.responseData, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
