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
	"monorepo/bin-ai-manager/pkg/aicallhandler"
)

func Test_processV1AIcallsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcalls []*aicall.AIcall

		expectCustomerID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectFilters    map[string]string
		expectRes        *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/aicalls?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=645e65c8-a773-11ed-b5ae-df76e94347ad&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			[]*aicall.AIcall{
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

			uuid.FromStringOrNil("645e65c8-a773-11ed-b5ae-df76e94347ad"),
			10,
			"2020-05-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"64b555fe-a773-11ed-9dc7-2fccabe21218","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"},{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}]`),
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

			mockAIcall.EXPECT().Gets(gomock.Any(), tt.expectCustomerID, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseAIcalls, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIcallsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectAIID          uuid.UUID
		expectActiveflowID  uuid.UUID
		expectReferenceType aicall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        aicall.Gender
		expectLanguage      string

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aicalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"ai_id": "f9e5ec32-ef4d-11ef-80de-8bc376898e49", "reference_type": "call", "reference_id":"fa2471be-ef4d-11ef-80b1-5bee84085737","gender":"female","language":"en-US"}`),
			},

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6792a0d8-a773-11ed-b28c-c79bf61e95b2"),
				},
			},

			expectAIID:          uuid.FromStringOrNil("f9e5ec32-ef4d-11ef-80de-8bc376898e49"),
			expectActiveflowID:  uuid.Nil,
			expectReferenceType: aicall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("fa2471be-ef4d-11ef-80b1-5bee84085737"),
			expectGender:        aicall.GenderFemale,
			expectLanguage:      "en-US",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6792a0d8-a773-11ed-b28c-c79bf61e95b2","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
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

			mockAIcall.EXPECT().Start(gomock.Any(), tt.expectAIID, tt.expectActiveflowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectGender, tt.expectLanguage).Return(tt.responseAIcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIcallsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/aicalls/d9d804d8-ef03-4a23-906c-c192029b19fc",
				Method: sock.RequestMethodDelete,
			},

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),
				},
			},

			uuid.FromStringOrNil("d9d804d8-ef03-4a23-906c-c192029b19fc"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d9d804d8-ef03-4a23-906c-c192029b19fc","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
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

			mockAIcall.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseAIcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIcallsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIcall *aicall.AIcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/aicalls/3e349bb8-7b31-4533-8e2b-6654ebc84e3e",
				Method: sock.RequestMethodGet,
			},

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),
				},
			},

			uuid.FromStringOrNil("3e349bb8-7b31-4533-8e2b-6654ebc84e3e"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3e349bb8-7b31-4533-8e2b-6654ebc84e3e","customer_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`),
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

			mockAIcall.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseAIcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
