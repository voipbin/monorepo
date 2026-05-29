package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/aipromptproposalhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_processV1AIPromptProposalsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aipromptproposal.AIPromptProposal
		responseErr    error

		expectedCustomerID uuid.UUID
		expectedAIID       uuid.UUID
		expectedAuditIDs   []uuid.UUID
		expectedLanguage   string
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"a0000000-0000-0000-0000-000000000003","audit_ids":["a0000000-0000-0000-0000-000000000004"],"language":"en-US"}`),
			},

			responseRecord: &aipromptproposal.AIPromptProposal{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				AIID:                 uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
				AuditIDs:             []uuid.UUID{uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004")},
				BasisPromptHistoryID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000005"),
				Status:               aipromptproposal.StatusProgressing,
			},
			responseErr: nil,

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			expectedAuditIDs:   []uuid.UUID{uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004")},
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 202,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"a0000000-0000-0000-0000-000000000003","audit_ids":["a0000000-0000-0000-0000-000000000004"],"basis_prompt_history_id":"a0000000-0000-0000-0000-000000000005","status":"progressing","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "rate_limit_exceeded_returns_429",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"a0000000-0000-0000-0000-000000000003","audit_ids":["a0000000-0000-0000-0000-000000000004"],"language":"en-US"}`),
			},

			responseRecord: nil,
			responseErr:    &proposalRateLimitError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			expectedAuditIDs:   []uuid.UUID{uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004")},
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 429,
			},
		},
		{
			name: "invalid_audit_set_returns_400",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"a0000000-0000-0000-0000-000000000003","audit_ids":["a0000000-0000-0000-0000-000000000004"],"language":"en-US"}`),
			},

			responseRecord: nil,
			responseErr:    &invalidAuditSetError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			expectedAuditIDs:   []uuid.UUID{uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004")},
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 400,
			},
		},
		{
			name: "ai_not_found_returns_404",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"a0000000-0000-0000-0000-000000000003","audit_ids":["a0000000-0000-0000-0000-000000000004"],"language":"en-US"}`),
			},

			responseRecord: nil,
			responseErr:    &aiNotFoundError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedAIID:       uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003"),
			expectedAuditIDs:   []uuid.UUID{uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000004")},
			expectedLanguage:   "en-US",
			expectedRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().Create(gomock.Any(), tt.expectedCustomerID, tt.expectedAIID, tt.expectedAuditIDs, tt.expectedLanguage).Return(tt.responseRecord, tt.responseErr)
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

func Test_processV1AIPromptProposalsGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecords []*aipromptproposal.AIPromptProposal
		responseErr     error

		expectedPageSize uint64
		expectedToken    string
		expectedFilters  map[aipromptproposal.Field]any
		expectedRes      *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals?page_size=10&page_token=",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001","deleted":false}`),
			},

			responseRecords: []*aipromptproposal.AIPromptProposal{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
						CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
					},
					Status: aipromptproposal.StatusCompleted,
				},
			},
			responseErr: nil,

			expectedPageSize: 10,
			expectedToken:    "",
			expectedFilters: map[aipromptproposal.Field]any{
				aipromptproposal.FieldCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				aipromptproposal.FieldDeleted:    false,
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"completed","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().List(gomock.Any(), tt.expectedPageSize, tt.expectedToken, tt.expectedFilters).Return(tt.responseRecords, tt.responseErr)
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

func Test_processV1AIPromptProposalsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aipromptproposal.AIPromptProposal
		responseErr    error

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aipromptproposals/c0000000-0000-0000-0000-000000000001",
				Method: sock.RequestMethodGet,
			},

			responseRecord: &aipromptproposal.AIPromptProposal{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status: aipromptproposal.StatusCompleted,
			},
			responseErr: nil,

			expectedID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"completed","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().Get(gomock.Any(), tt.expectedID).Return(tt.responseRecord, tt.responseErr)
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

func Test_processV1AIPromptProposalsIDAcceptPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aipromptproposal.AIPromptProposal
		responseErr    error

		expectedCustomerID uuid.UUID
		expectedID         uuid.UUID
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals/c0000000-0000-0000-0000-000000000001/accept",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001"}`),
			},

			responseRecord: &aipromptproposal.AIPromptProposal{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status: aipromptproposal.StatusAccepted,
			},
			responseErr: nil,

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"accepted","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "prompt_version_drifted_returns_409",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals/c0000000-0000-0000-0000-000000000001/accept",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001"}`),
			},

			responseRecord: nil,
			responseErr:    &promptVersionDriftedError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 409,
			},
		},
		{
			name: "proposal_not_found_returns_404",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals/c0000000-0000-0000-0000-000000000001/accept",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001"}`),
			},

			responseRecord: nil,
			responseErr:    &proposalNotFoundError{},

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().Accept(gomock.Any(), tt.expectedCustomerID, tt.expectedID).Return(tt.responseRecord, tt.responseErr)
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

func Test_processV1AIPromptProposalsIDRejectPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aipromptproposal.AIPromptProposal
		responseErr    error

		expectedCustomerID uuid.UUID
		expectedID         uuid.UUID
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/aipromptproposals/c0000000-0000-0000-0000-000000000001/reject",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a0000000-0000-0000-0000-000000000001"}`),
			},

			responseRecord: &aipromptproposal.AIPromptProposal{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status: aipromptproposal.StatusRejected,
			},
			responseErr: nil,

			expectedCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			expectedID:         uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"rejected","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().Reject(gomock.Any(), tt.expectedCustomerID, tt.expectedID).Return(tt.responseRecord, tt.responseErr)
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

func Test_processV1AIPromptProposalsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseRecord *aipromptproposal.AIPromptProposal
		responseErr    error

		expectedID  uuid.UUID
		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/aipromptproposals/d0000000-0000-0000-0000-000000000001",
				Method: sock.RequestMethodDelete,
			},

			responseRecord: &aipromptproposal.AIPromptProposal{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				},
				Status: aipromptproposal.StatusRejected,
			},
			responseErr: nil,

			expectedID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0000000-0000-0000-0000-000000000001","customer_id":"a0000000-0000-0000-0000-000000000001","ai_id":"00000000-0000-0000-0000-000000000000","basis_prompt_history_id":"00000000-0000-0000-0000-000000000000","status":"rejected","applied_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockProposal := aipromptproposalhandler.NewMockAIPromptProposalHandler(mc)

			h := &listenHandler{
				sockHandler:             mockSock,
				aipromptproposalHandler: mockProposal,
			}

			mockProposal.EXPECT().Delete(gomock.Any(), tt.expectedID).Return(tt.responseRecord, tt.responseErr)
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

// proposalRateLimitError simulates a "rate limit exceeded" error from Create.
type proposalRateLimitError struct{}

func (e *proposalRateLimitError) Error() string {
	return "rate limit exceeded: customer already has 3 proposals in progress"
}

// invalidAuditSetError simulates an "invalid audit set" error from Create.
type invalidAuditSetError struct{}

func (e *invalidAuditSetError) Error() string {
	return "invalid audit set: empty audit list"
}

// aiNotFoundError simulates an "ai not found" error from Create.
type aiNotFoundError struct{}

func (e *aiNotFoundError) Error() string {
	return "ai not found"
}

// promptVersionDriftedError simulates a "prompt version drifted" error from Accept.
type promptVersionDriftedError struct{}

func (e *promptVersionDriftedError) Error() string {
	return "prompt version drifted"
}

// proposalNotFoundError simulates a "proposal not found" error from Accept/Reject.
type proposalNotFoundError struct{}

func (e *proposalNotFoundError) Error() string {
	return "proposal not found"
}
