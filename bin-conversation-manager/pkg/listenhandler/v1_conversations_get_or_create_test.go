package listenhandler

import (
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

// Test_processV1ConversationsGetOrCreateBySelfAndPeerPost verifies the
// distinct get-or-create endpoint used by bin-contact-manager's
// agent-send Case-linked messaging path (contact-case-management design
// §4.5, round-12 correction). Must call GetOrCreateBySelfAndPeer, NOT
// GetBySelfAndPeer.
func Test_processV1ConversationsGetOrCreateBySelfAndPeerPost(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectCustomerID uuid.UUID
		expectType       conversation.Type
		expectDialogID   string
		expectSelf       commonaddress.Address
		expectPeer       commonaddress.Address

		responseConversation *conversation.Conversation

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/get_or_create_by_self_and_peer",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"f1b2c3d4-0006-0006-0006-000000000001","type":"message","dialog_id":"","self":{"type":"tel","target":"+15551110000"},"peer":{"type":"tel","target":"+15552220000"}}`),
			},

			expectCustomerID: uuid.FromStringOrNil("f1b2c3d4-0006-0006-0006-000000000001"),
			expectType:       conversation.TypeMessage,
			expectDialogID:   "",
			expectSelf:       commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"},
			expectPeer:       commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-0006-0006-0006-000000000002"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f1b2c3d4-0006-0006-0006-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			mockConversation.EXPECT().
				GetOrCreateBySelfAndPeer(gomock.Any(), tt.expectCustomerID, tt.expectType, tt.expectDialogID, tt.expectSelf, tt.expectPeer).
				Return(tt.responseConversation, nil)

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
