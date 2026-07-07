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
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Test_processV1ConversationsSelfAndPeerGet verifies the get-only lookup
// endpoint used by bin-contact-manager's proactive Case-linking write
// path (contact-case-management design §4.4).
func Test_processV1ConversationsSelfAndPeerGet(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectSelf commonaddress.Address
		expectPeer commonaddress.Address

		responseConversation *conversation.Conversation

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/self_and_peer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"self":{"type":"tel","target":"+15551110000"},"peer":{"type":"tel","target":"+15552220000"}}`),
			},

			expectSelf: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"},
			expectPeer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220000"},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-0004-0004-0004-000000000001"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f1b2c3d4-0004-0004-0004-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			mockConversation.EXPECT().GetBySelfAndPeer(gomock.Any(), tt.expectSelf, tt.expectPeer).Return(tt.responseConversation, nil)

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

// Test_processV1ConversationsSelfAndPeerGet_notFound verifies a miss
// returns 404, matching this service's existing not-found convention for
// conversation lookups (Test_processV1ConversationsID_notFound).
func Test_processV1ConversationsSelfAndPeerGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)
	h := &listenHandler{
		sockHandler:         mockSock,
		conversationHandler: mockConversation,
	}

	req := &sock.Request{
		URI:      "/v1/conversations/self_and_peer",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{"self":{"type":"tel","target":"+15551110000"},"peer":{"type":"tel","target":"+15559990000"}}`),
	}

	mockConversation.EXPECT().GetBySelfAndPeer(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
	}
}
