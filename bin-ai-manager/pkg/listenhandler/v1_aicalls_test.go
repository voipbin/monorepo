package listenhandler

import (
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
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
				Data:       []byte(`[{"id":"64b555fe-a773-11ed-9dc7-2fccabe21218","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
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

		expectedAssistanceType aicall.AssistanceType
		expectedAssistanceID   uuid.UUID
		expectedActiveflowID   uuid.UUID
		expectedReferenceType  aicall.ReferenceType
		expectedReferenceID    uuid.UUID
		expectedRes            *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aicalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"assistance_type": "ai", "assistance_id": "f9e5ec32-ef4d-11ef-80de-8bc376898e49", "activeflow_id": "969e3754-0cc3-11f0-80b3-7760a1de452c", "reference_type": "call", "reference_id":"fa2471be-ef4d-11ef-80b1-5bee84085737"}`),
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
				},
			},

			expectedAssistanceType: aicall.AssistanceTypeAI,
			expectedAssistanceID:   uuid.FromStringOrNil("f9e5ec32-ef4d-11ef-80de-8bc376898e49"),
			expectedActiveflowID:   uuid.FromStringOrNil("969e3754-0cc3-11f0-80b3-7760a1de452c"),
			expectedReferenceType:  aicall.ReferenceTypeCall,
			expectedReferenceID:    uuid.FromStringOrNil("fa2471be-ef4d-11ef-80b1-5bee84085737"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			mockAIcall.EXPECT().Start(gomock.Any(), tt.expectedAssistanceType, tt.expectedAssistanceID, tt.expectedActiveflowID, tt.expectedReferenceType, tt.expectedReferenceID).Return(tt.responseAIcall, nil)
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
				Data:       []byte(`{"id":"d9d804d8-ef03-4a23-906c-c192029b19fc","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				Data:       []byte(`{"id":"3e349bb8-7b31-4533-8e2b-6654ebc84e3e","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				Data:       []byte(`{"id":"24a00d20-9199-11f0-b036-f7aebbe6e8f8","customer_id":"00000000-0000-0000-0000-000000000000","assistance_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","pipecatcall_id":"00000000-0000-0000-0000-000000000000","current_member_id":"00000000-0000-0000-0000-000000000000","listen_call_id":"00000000-0000-0000-0000-000000000000","tm_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

		expectedID            uuid.UUID
		expectedToolID        string
		expectedToolType      message.ToolType
		expectedToolFunction  message.FunctionCall
		expectedPipecatcallID uuid.UUID

		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aicalls/a02f9d60-bbb6-11f0-81e6-7fbbd900fc6b/tool_execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"tool-1234","type":"function","function":{"name":"connect","arguments":"{\"source\":{\"target\":\"+1234567890\"}}"},"pipecatcall_id":"2c1d4e6f-8a9b-4c0d-9e1f-3a5b7c9d1e2f"}`),
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
			expectedPipecatcallID: uuid.FromStringOrNil("2c1d4e6f-8a9b-4c0d-9e1f-3a5b7c9d1e2f"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"message":"","result":"success"}`),
			},
		},
		{
			// Pins the rolling-deploy compatibility promise in
			// V1DataAIcallsIDToolExecutePost.PipecatcallID's own doc comment: an
			// old bin-pipecat-manager sends no such field, and it must unmarshal
			// to uuid.Nil rather than fail the request.
			name: "no pipecatcall_id in the body unmarshals to uuid.Nil",
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
			expectedPipecatcallID: uuid.Nil,
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

			mockAIcall.EXPECT().ToolHandle(gomock.Any(), tt.expectedID, tt.expectedToolID, tt.expectedToolType, tt.expectedToolFunction, tt.expectedPipecatcallID).Return(tt.responseToolHandle, nil)
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

func Test_processV1AIcallsIDParticipantsGet(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	now := time.Now().UTC()

	tests := []struct {
		name      string
		request   *sock.Request
		mockSetup func(mockParticipant *participanthandler.MockParticipantHandler)
		expectRes *sock.Response
	}{
		{
			name: "returns participants list",
			request: &sock.Request{
				URI:    "/v1/aicalls/11111111-1111-1111-1111-111111111111/participants?page_size=10&page_token=2026-01-01T00:00:00Z",
				Method: sock.RequestMethodGet,
			},
			mockSetup: func(mockParticipant *participanthandler.MockParticipantHandler) {
				mockParticipant.EXPECT().ListByAIcallID(gomock.Any(), aicallID, uint64(10), "2026-01-01T00:00:00Z").Return([]*participant.Participant{
					{AIID: aiID, AIcallID: aicallID, TMCreate: &now},
				}, nil).Times(1)
			},
			expectRes: &sock.Response{StatusCode: 200, DataType: "application/json"},
		},
		{
			name: "returns 404 on invalid UUID (no route match)",
			request: &sock.Request{
				URI:    "/v1/aicalls/not-a-uuid/participants?page_size=10",
				Method: sock.RequestMethodGet,
			},
			mockSetup: func(mockParticipant *participanthandler.MockParticipantHandler) {},
			expectRes: &sock.Response{StatusCode: 404},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)
			tt.mockSetup(mockParticipant)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.StatusCode != tt.expectRes.StatusCode {
				t.Fatalf("expected status %d, got %d", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1AIcallsIDToolExecutePost_errorMapping(t *testing.T) {
	tests := []struct {
		name         string
		request      *sock.Request
		handlerErr   error
		expectStatus int
	}{
		{
			name: "dbhandler.ErrNotFound maps to 404",
			request: &sock.Request{
				URI:      "/v1/aicalls/a02f9d60-bbb6-11f0-81e6-7fbbd900fc6b/tool_execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"tool-1","type":"function","function":{"name":"connect","arguments":"{}"}}`),
			},
			handlerErr:   dbhandler.ErrNotFound,
			expectStatus: 404,
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

			mockAIcall.EXPECT().ToolHandle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.handlerErr)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

// Test_processV1AIcallsIDGet_SkipCache pins that a GET carrying a query string
// still routes and still parses its id.
//
// Two independent things would break it. (1) regV1AIcallsID is anchored with
// "$", so a query string makes it match nothing and the request silently falls
// through to the router's default -- hence the separate query-tolerant pattern.
// (2) The handler used to split the URI on "/" and take element 3, which for a
// query-bearing URI is "<uuid>?skip_cache=true" and parses to uuid.Nil -- hence
// url.Parse.
func Test_processV1AIcallsIDGet_SkipCache(t *testing.T) {
	aicallID := uuid.FromStringOrNil("3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c")

	tests := []struct {
		name string

		request *sock.Request

		expectSkipCache bool
	}{
		{
			name: "no query string reads through the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: false,
		},
		{
			name: "skip_cache=true bypasses the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c?skip_cache=true",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: true,
		},
		{
			name: "skip_cache=false reads through the cache",
			request: &sock.Request{
				URI:    "/v1/aicalls/3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c?skip_cache=false",
				Method: sock.RequestMethodGet,
			},
			expectSkipCache: false,
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

			if tt.expectSkipCache {
				mockAIcall.EXPECT().GetSkipCache(gomock.Any(), aicallID).Return(&aicall.AIcall{}, nil)
			} else {
				mockAIcall.EXPECT().Get(gomock.Any(), aicallID).Return(&aicall.AIcall{}, nil)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest returned an unexpected error. err: %v", err)
			}
			if res.StatusCode != 200 {
				t.Errorf("status mismatch. expected: 200, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1AIcallsIDListenPost pins the new route end to end at the
// transport layer: the regex, the dispatcher case, the id parse, and the one
// business-handler call.
//
// IT ROUTES THROUGH processRequest, NOT processV1AIcallsIDListenPost DIRECTLY.
// That is the same deliberate choice Test_processV1AIcallsIDGet_SkipCache made,
// for the same reason: the known regex-anchoring behaviour of this dispatcher
// (regV1AIcallsID ends in "$", which is why a separate query-tolerant pattern
// had to be added) lives in the dispatcher, so a test that calls the handler
// function directly would pass while production never routes the request at all.
func Test_processV1AIcallsIDListenPost(t *testing.T) {
	aicallID := uuid.FromStringOrNil("6a1f0c22-9b3d-4e7f-8a5b-1c2d3e4f5a6b")

	t.Run("valid id routes and calls ProcessListen exactly once", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

		h := &listenHandler{sockHandler: mockSock, aicallHandler: mockAIcall}

		mockAIcall.EXPECT().ProcessListen(gomock.Any(), aicallID).
			Return(&aicall.AIcall{Identity: identity.Identity{ID: aicallID}}, nil).Times(1)

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/aicalls/6a1f0c22-9b3d-4e7f-8a5b-1c2d3e4f5a6b/listen",
			Method: sock.RequestMethodPost,
		})
		if err != nil {
			t.Fatalf("processRequest returned an unexpected error. err: %v", err)
		}
		if res.StatusCode != 200 {
			t.Errorf("status mismatch. expected: 200, got: %d", res.StatusCode)
		}
	})

	t.Run("unknown id surfaces as 404, not a generic 500", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

		h := &listenHandler{sockHandler: mockSock, aicallHandler: mockAIcall}

		mockAIcall.EXPECT().ProcessListen(gomock.Any(), aicallID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/aicalls/6a1f0c22-9b3d-4e7f-8a5b-1c2d3e4f5a6b/listen",
			Method: sock.RequestMethodPost,
		})
		if err != nil {
			t.Fatalf("processRequest returned an unexpected error. err: %v", err)
		}
		if res.StatusCode != 404 {
			t.Errorf("status mismatch. expected: 404, got: %d", res.StatusCode)
		}
	})

	t.Run("unparseable id never reaches ProcessListen", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

		h := &listenHandler{sockHandler: mockSock, aicallHandler: mockAIcall}

		// The invariant that matters, whichever layer enforces it: no
		// business-handler call is made on a malformed id.
		mockAIcall.EXPECT().ProcessListen(gomock.Any(), gomock.Any()).Times(0)

		if _, err := h.processRequest(&sock.Request{
			URI:    "/v1/aicalls/not-a-uuid/listen",
			Method: sock.RequestMethodPost,
		}); err != nil {
			t.Fatalf("processRequest returned an unexpected error. err: %v", err)
		}
	})

	t.Run("GET on the same path does not dispatch to the listen handler", func(t *testing.T) {
		// Pins the `&& m.Method == sock.RequestMethodPost` half of the switch
		// case.
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

		h := &listenHandler{sockHandler: mockSock, aicallHandler: mockAIcall}

		mockAIcall.EXPECT().ProcessListen(gomock.Any(), gomock.Any()).Times(0)

		if _, err := h.processRequest(&sock.Request{
			URI:    "/v1/aicalls/6a1f0c22-9b3d-4e7f-8a5b-1c2d3e4f5a6b/listen",
			Method: sock.RequestMethodGet,
		}); err != nil {
			t.Fatalf("processRequest returned an unexpected error. err: %v", err)
		}
	})
}
