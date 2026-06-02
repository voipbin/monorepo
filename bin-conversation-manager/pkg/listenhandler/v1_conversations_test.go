package listenhandler

import (
	stderrors "errors"
	"fmt"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1ConversationsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectPageSize  uint64
		expectPageToken string
		expectFields    map[conversation.Field]any

		responseConversations []*conversation.Conversation

		response *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations?page_size=10&page_token=2021-03-01T03:30:17.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01T03:30:17.000000Z",
			expectFields: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("645891fe-e863-11ec-b291-9f454e92f1bb"),
						CustomerID: uuid.FromStringOrNil("64a3cbd8-e863-11ec-85de-1bcd09d3872e"),
					},
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"645891fe-e863-11ec-b291-9f454e92f1bb","customer_id":"64a3cbd8-e863-11ec-85de-1bcd09d3872e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "2 results",

			request: &sock.Request{
				URI:      "/v1/conversations?page_size=10&page_token=2021-03-01T03:30:17.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01T03:30:17.000000Z",
			expectFields: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b7ac843c-e863-11ec-9652-0ff162b38a15"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c45aec8c-e863-11ec-9bae-4fcfe883444a"),
						CustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
					},
				},
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"b7ac843c-e863-11ec-9652-0ff162b38a15","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"c45aec8c-e863-11ec-9bae-4fcfe883444a","customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			// owner_id passes through ConvertStringMapToFieldMap as a parsed
			// uuid.UUID under FieldOwnerID, so non-admin agent callers can list
			// "my conversations" via the owner_id filter (design §5.5).
			name: "owner_id filter pass-through",

			request: &sock.Request{
				URI:      "/v1/conversations?page_size=10&page_token=2021-03-01T03:30:17.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b77be746-e863-11ec-97b0-bb06bbb7db0e","owner_id":"eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b","deleted":false}`),
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-01T03:30:17.000000Z",
			expectFields: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("b77be746-e863-11ec-97b0-bb06bbb7db0e"),
				conversation.FieldOwnerID:    uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				utilHandler:         mockUtil,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().
				List(gomock.Any(), tt.expectPageToken, tt.expectPageSize, tt.expectFields).
				Return(tt.responseConversations, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1ConversationsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConversation *conversation.Conversation

		expectedCustomerID uuid.UUID
		expectedName       string
		expectedDetail     string
		expectedType       conversation.Type
		expectedDialogID   string
		expectedSelf       commonaddress.Address
		expectedPeer       commonaddress.Address
		expectedRes        *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"456609ea-fecc-11ed-a717-5f6984c51794","name":"test name","detail":"test detail","type":"line","dialog_id":"b5404340-1acf-11f0-941a-633dfb3b6be3","self":{"type":"line","target":"b589c6f0-1acf-11f0-b1ad-a32b39bc73a2"},"peer":{"type":"line","target":"b5b28d6a-1acf-11f0-86ed-cb4575eb8b11"}}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5dc1f5e-1acf-11f0-8f6f-7f18c79eb9a2"),
				},
			},

			expectedCustomerID: uuid.FromStringOrNil("456609ea-fecc-11ed-a717-5f6984c51794"),
			expectedName:       "test name",
			expectedDetail:     "test detail",
			expectedType:       conversation.TypeLine,
			expectedDialogID:   "b5404340-1acf-11f0-941a-633dfb3b6be3",
			expectedSelf: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "b589c6f0-1acf-11f0-b1ad-a32b39bc73a2",
			},
			expectedPeer: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "b5b28d6a-1acf-11f0-86ed-cb4575eb8b11",
			},

			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b5dc1f5e-1acf-11f0-8f6f-7f18c79eb9a2","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				accountHandler:      mockAccount,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Create(
				gomock.Any(),
				tt.expectedCustomerID,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedType,
				tt.expectedDialogID,
				tt.expectedSelf,
				tt.expectedPeer,
			).Return(tt.responseConversation, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1ConversationsIDGet(t *testing.T) {

	tests := []struct {
		name string

		expectID uuid.UUID

		resultData *conversation.Conversation

		responseConversation *sock.Request
		response             *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),
				},
			},

			&sock.Request{
				URI:    "/v1/conversations/73071e00-a29a-11ec-a43a-079fe08ce740",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"73071e00-a29a-11ec-a43a-079fe08ce740","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.responseConversation)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1ConversationsIDPut(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseConversation *conversation.Conversation

		expectedConversationID uuid.UUID
		expectedFields         map[conversation.Field]any
		expectRes              *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name", "detail":"test detail", "account_id": "a3f340b4-21ec-11f0-9b2a-f70f3bf0b3be"}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
				},
			},

			expectedConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectedFields: map[conversation.Field]any{
				conversation.FieldName:      "test name",
				conversation.FieldDetail:    "test detail",
				conversation.FieldAccountID: uuid.FromStringOrNil("a3f340b4-21ec-11f0-9b2a-f70f3bf0b3be"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d8ab6ae-0074-11ee-80d0-df60c15605d7","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			// Empty-string values must round-trip from the JSON body through
			// GetFilteredItems and ConvertStringMapToFieldMap into the field
			// map passed to Update unchanged. The conversion pipeline must
			// not collapse "" into nil/missing.
			name: "empty-string name preserved",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":""}`),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
				},
			},

			expectedConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectedFields: map[conversation.Field]any{
				conversation.FieldName: "",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8d8ab6ae-0074-11ee-80d0-df60c15605d7","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Update(gomock.Any(), tt.expectedConversationID, tt.expectedFields).Return(tt.responseConversation, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1ConversationsID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "GET non-existent conversation returns 404",
			request: &sock.Request{
				URI:    "/v1/conversations/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Get(gomock.Any(), tt.id).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1ConversationsID_notFoundTyped verifies that the typed
// *cerrors.VoipbinError (NotFound) returned by the real conversationHandler.Get
// is correctly translated to HTTP 404 via the errors.As → cerrors.ToResponse branch
// in errorResponse.
func Test_processV1ConversationsID_notFoundTyped(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "GET non-existent conversation returns 404 via typed cerrors.NotFound",
			request: &sock.Request{
				URI:    "/v1/conversations/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Get(gomock.Any(), tt.id).Return(nil, cerrors.NotFound(
				commonoutline.ServiceNameConversationManager,
				"CONVERSATION_NOT_FOUND",
				"The conversation was not found.",
			))

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1ConversationsIDPut_ErrorPassthrough verifies that errors
// returned from conversationHandler.Update flow through errorResponse so the
// api-manager edge sees the right HTTP status (design §5.4):
//   - typed *cerrors.VoipbinError (InvalidArgument) → 400 with the typed envelope
//   - any other error → 500 simple response
func Test_processV1ConversationsIDPut_ErrorPassthrough(t *testing.T) {
	tests := []struct {
		name string

		request   *sock.Request
		updateErr error

		expectedConversationID uuid.UUID
		expectedFields         map[conversation.Field]any

		expectStatusCode int
		expectDataType   string
		expectIsTyped    bool
	}{
		{
			name: "typed cerrors.InvalidArgument surfaces as 400",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"owner_id":"f1233333-ab21-11ee-80d0-aabbccddeeff"}`),
			},
			updateErr: cerrors.InvalidArgument(
				commonoutline.ServiceNameConversationManager,
				"AGENT_NOT_FOUND",
				"agent not found. owner_id: f1233333-ab21-11ee-80d0-aabbccddeeff",
			),

			expectedConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectedFields: map[conversation.Field]any{
				conversation.FieldOwnerID: uuid.FromStringOrNil("f1233333-ab21-11ee-80d0-aabbccddeeff"),
			},

			expectStatusCode: 400,
			expectDataType:   string(cerrors.DataTypeVoipbinError),
			expectIsTyped:    true,
		},
		{
			name: "generic error surfaces as 500",

			request: &sock.Request{
				URI:      "/v1/conversations/8d8ab6ae-0074-11ee-80d0-df60c15605d7",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name"}`),
			},
			updateErr: fmt.Errorf("database write failed"),

			expectedConversationID: uuid.FromStringOrNil("8d8ab6ae-0074-11ee-80d0-df60c15605d7"),
			expectedFields: map[conversation.Field]any{
				conversation.FieldName: "test name",
			},

			expectStatusCode: 500,
			expectDataType:   "",
			expectIsTyped:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().
				Update(gomock.Any(), tt.expectedConversationID, tt.expectedFields).
				Return(nil, tt.updateErr)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil err from processRequest, got: %v", err)
			}
			if res == nil {
				t.Fatalf("expected non-nil response")
			}
			if res.StatusCode != tt.expectStatusCode {
				t.Errorf("wrong status code. expect: %d, got: %d", tt.expectStatusCode, res.StatusCode)
			}
			if string(res.DataType) != tt.expectDataType {
				t.Errorf("wrong data type. expect: %q, got: %q", tt.expectDataType, res.DataType)
			}

			if tt.expectIsTyped {
				ve := cerrors.FromResponse(res)
				if ve == nil {
					t.Fatalf("expected typed *VoipbinError in response body, got none. res: %+v", res)
				}
				if ve.Status != cerrors.StatusInvalidArgument {
					t.Errorf("expected Status=InvalidArgument, got: %v", ve.Status)
				}
			} else {
				// Legacy simpleResponse — no typed envelope, no body.
				if cerrors.FromResponse(res) != nil {
					t.Errorf("did not expect typed envelope for non-typed error path")
				}
			}

			// Sanity: the typed envelope check above already exercises errors.As semantics
			// indirectly; this asserts our fixture really is a typed error chain when expected.
			if tt.expectIsTyped {
				var ve *cerrors.VoipbinError
				if !stderrors.As(tt.updateErr, &ve) {
					t.Fatalf("test fixture should be a typed VoipbinError, got: %T", tt.updateErr)
				}
			}
		})
	}
}
