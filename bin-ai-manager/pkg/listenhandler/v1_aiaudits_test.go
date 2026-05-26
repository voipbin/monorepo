package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/pkg/aiaudithandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_processV1AIAuditsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecords []*aiaudit.AIAudit
		responseErr     error

		expectedCustomerID uuid.UUID
		expectedAIcallID   uuid.UUID
		expectedLanguage   string
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aiaudits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","aicall_id":"a0000000-0000-0000-0000-000000000002","language":"en-US"}`),
			},

			responseRecords: []*aiaudit.AIAudit{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
						CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
					},
					AIcallID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
					Status:   aiaudit.StatusProgressing,
					Language: "en-US",
				},
			},
			responseErr: nil,

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIcallID:   uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 202,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"b0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","aicall_id":"a0000000-0000-0000-0000-000000000002","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"progressing","overall_score":null,"evaluation":null,"language":"en-US","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "not_terminated_returns_400",
			request: &sock.Request{
				URI:      "/v1/aiaudits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","aicall_id":"a0000000-0000-0000-0000-000000000002","language":"en-US"}`),
			},

			responseRecords: nil,
			responseErr:     &notTerminatedError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIcallID:   uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002"),
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIAudit := aiaudithandler.NewMockAIAuditHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				aiauditHandler: mockAIAudit,
			}

			mockAIAudit.EXPECT().Create(gomock.Any(), tt.expectedCustomerID, tt.expectedAIcallID, tt.expectedLanguage).Return(tt.responseRecords, tt.responseErr)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1AIAuditsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aiaudit.AIAudit
		responseErr    error

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aiaudits/c0000000-0000-0000-0000-000000000001",
				Method: sock.RequestMethodGet,
			},

			responseRecord: &aiaudit.AIAudit{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status:   aiaudit.StatusCompleted,
				Language: "en-US",
			},
			responseErr: nil,

			expectedID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"completed","overall_score":null,"evaluation":null,"language":"en-US","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "invalid_uuid_returns_404",
			request: &sock.Request{
				URI:    "/v1/aiaudits/not-a-uuid",
				Method: sock.RequestMethodGet,
			},

			responseRecord: nil,
			responseErr:    nil,

			expectedID:  uuid.Nil,
			expectedRes: &sock.Response{StatusCode: 404},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIAudit := aiaudithandler.NewMockAIAuditHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				aiauditHandler: mockAIAudit,
			}

			if tt.expectedID != uuid.Nil {
				mockAIAudit.EXPECT().Get(gomock.Any(), tt.expectedID).Return(tt.responseRecord, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1AIAuditsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aiaudit.AIAudit
		responseErr    error

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aiaudits/d0000000-0000-0000-0000-000000000001",
				Method: sock.RequestMethodDelete,
			},

			responseRecord: &aiaudit.AIAudit{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status:   aiaudit.StatusCompleted,
				Language: "en-US",
			},
			responseErr: nil,

			expectedID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","aicall_id":"00000000-0000-0000-0000-000000000000","ai_id":"00000000-0000-0000-0000-000000000000","prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"completed","overall_score":null,"evaluation":null,"language":"en-US","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIAudit := aiaudithandler.NewMockAIAuditHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				aiauditHandler: mockAIAudit,
			}

			mockAIAudit.EXPECT().Delete(gomock.Any(), tt.expectedID).Return(tt.responseRecord, tt.responseErr)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

// notTerminatedError simulates a "not terminated" error from Create.
type notTerminatedError struct{}

func (e *notTerminatedError) Error() string {
	return "aicall not terminated: current status progressing"
}
