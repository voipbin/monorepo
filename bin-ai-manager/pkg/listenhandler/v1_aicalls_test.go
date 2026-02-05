package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
)

func Test_processV1AIcallsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcalls []*aicall.AIcall

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[aicall.Field]any
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aicalls?page_size=10&page_token=2020-05-03T21:35:02.809Z&filter_customer_id=645e65c8-a773-11ed-b5ae-df76e94347ad&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseAIcalls: []*aicall.AIcall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("64b555fe-a773-11ed-9dc7-2fccabe21218"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-05-03T21:35:02.809Z",
			expectFilters: map[aicall.Field]any{
				aicall.FieldDeleted:    false,
				aicall.FieldCustomerID: uuid.FromStringOrNil("645e65c8-a773-11ed-b5ae-df76e94347ad"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"64b555fe-a773-11ed-9dc7-2fccabe21218","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(tt.responseAIcalls, nil)
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

func Test_processV1AIcallsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectedAIID          uuid.UUID
		expectedActiveflowID  uuid.UUID
		expectedReferenceType aicall.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedGender        aicall.Gender
		expectedLanguage      string
		expectedRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aicalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"ai_id": "f9e5ec32-ef4d-11ef-80de-8bc376898e49", "activeflow_id": "969e3754-0cc3-11f0-80b3-7760a1de452c", "reference_type": "call", "reference_id":"fa2471be-ef4d-11ef-80b1-5bee84085737","gender":"female","language":"en-US"}`),
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
				},
			},

			expectedAIID:          uuid.FromStringOrNil("f9e5ec32-ef4d-11ef-80de-8bc376898e49"),
			expectedActiveflowID:  uuid.FromStringOrNil("969e3754-0cc3-11f0-80b3-7760a1de452c"),
			expectedReferenceType: aicall.ReferenceTypeCall,
			expectedReferenceID:   uuid.FromStringOrNil("fa2471be-ef4d-11ef-80b1-5bee84085737"),
			expectedGender:        aicall.GenderFemale,
			expectedLanguage:      "en-US",
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().Start(gomock.Any(), tt.expectedAIID, tt.expectedActiveflowID, tt.expectedReferenceType, tt.expectedReferenceID, tt.expectedGender, tt.expectedLanguage).Return(tt.responseAIcall, nil)
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

func Test_processV1AIcallsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aicalls/d9d804d8-ef03-4a23-906c-c192029b19fc",
				Method: sock.RequestMethodDelete,
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),
				},
			},

			expectedID: uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d9d804d8-ef03-4a23-906c-c192029b19fc","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().Delete(gomock.Any(), tt.expectedID).Return(tt.responseAIcall, nil)
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

func Test_processV1AIcallsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aicalls/3e349bb8-7b31-4533-8e2b-6654ebc84e3e",
				Method: sock.RequestMethodGet,
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),
				},
			},

			expectedID: uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3e349bb8-7b31-4533-8e2b-6654ebc84e3e","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().Get(gomock.Any(), tt.expectedID).Return(tt.responseAIcall, nil)
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

func Test_processV1AIcallsIDTerminatePost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aicalls/24a00d20-9199-11f0-b036-f7aebbe6e8f8/terminate",
				Method: sock.RequestMethodPost,
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("24a00d20-9199-11f0-b036-f7aebbe6e8f8"),
				},
			},

			expectedID: uuid.FromStringOrNil("24a00d20-9199-11f0-b036-f7aebbe6e8f8"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"24a00d20-9199-11f0-b036-f7aebbe6e8f8","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().ProcessTerminate(gomock.Any(), tt.expectedID).Return(tt.responseAIcall, nil)
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

func Test_processV1AIcallsIDToolExecutePost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseToolHandle map[string]any

		expectedID           uuid.UUID
		expectedToolID       string
		expectedToolType     message.ToolType
		expectedToolFunction message.FunctionCall

		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aicalls/a02f9d60-bbb6-11f0-81e6-7fbbd900fc6b/tool_execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"tool-1234","type":"function","function":{"name":"connect","arguments":"{\"source\":{\"target\":\"+1234567890\"}}"}}`),
			},

			responseToolHandle: map[string]any{
				"result":  "success",
				"message": "",
			},

			expectedID:       uuid.FromStringOrNil("a02f9d60-bbb6-11f0-81e6-7fbbd900fc6b"),
			expectedToolID:   "tool-1234",
			expectedToolType: message.ToolTypeFunction,
			expectedToolFunction: message.FunctionCall{
				Name:      "connect",
				Arguments: `{"source":{"target":"+1234567890"}}`,
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"message":"","result":"success"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().ToolHandle(gomock.Any(), tt.expectedID, tt.expectedToolID, tt.expectedToolType, tt.expectedToolFunction).Return(tt.responseToolHandle, nil)
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
