package listenhandler

import (
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

func Test_processV1ConversationsCreateAndExecuteFlowPost(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectCall       bool
		expectCustomerID uuid.UUID
		expectFlowID     uuid.UUID
		expectType       conversation.Type
		expectDialogID   string
		expectSelf       commonaddress.Address
		expectPeer       commonaddress.Address

		responseConversation *conversation.Conversation
		responseErr          error

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/create_and_execute_flow",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"f2b2c3d4-0001-0001-0001-000000000001","flow_id":"f2b2c3d4-0001-0001-0001-000000000002","type":"webchat","dialog_id":"dialog-1","self":{"type":"webchat","target":"self-target"},"peer":{"type":"webchat","target":"peer-target"}}`),
			},

			expectCall:       true,
			expectCustomerID: uuid.FromStringOrNil("f2b2c3d4-0001-0001-0001-000000000001"),
			expectFlowID:     uuid.FromStringOrNil("f2b2c3d4-0001-0001-0001-000000000002"),
			expectType:       conversation.TypeWebchat,
			expectDialogID:   "dialog-1",
			expectSelf:       commonaddress.Address{Type: "webchat", Target: "self-target"},
			expectPeer:       commonaddress.Address{Type: "webchat", Target: "peer-target"},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f2b2c3d4-0001-0001-0001-000000000003"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f2b2c3d4-0001-0001-0001-000000000003","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "malformed json returns 400 without calling the handler",

			request: &sock.Request{
				URI:      "/v1/conversations/create_and_execute_flow",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "handler error returns 500",

			request: &sock.Request{
				URI:      "/v1/conversations/create_and_execute_flow",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"f2b2c3d4-0001-0001-0001-000000000011","flow_id":"f2b2c3d4-0001-0001-0001-000000000012","type":"webchat"}`),
			},

			expectCall:       true,
			expectCustomerID: uuid.FromStringOrNil("f2b2c3d4-0001-0001-0001-000000000011"),
			expectFlowID:     uuid.FromStringOrNil("f2b2c3d4-0001-0001-0001-000000000012"),
			expectType:       conversation.TypeWebchat,
			responseErr:      fmt.Errorf("flow failed"),

			expectRes: &sock.Response{StatusCode: 500},
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

			if tt.expectCall {
				mockConversation.EXPECT().
					CreateAndExecuteFlow(gomock.Any(), tt.expectCustomerID, tt.expectFlowID, tt.expectType, tt.expectDialogID, tt.expectSelf, tt.expectPeer).
					Return(tt.responseConversation, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
