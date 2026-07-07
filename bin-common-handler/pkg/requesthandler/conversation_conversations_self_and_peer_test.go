package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// Test_ConversationV1ConversationGetBySelfAndPeer verifies the get-only
// (never-create) RPC client used by bin-contact-manager's proactive
// Case-linking write path (contact-case-management design §4.4).
func Test_ConversationV1ConversationGetBySelfAndPeer(t *testing.T) {
	tests := []struct {
		name string

		self address.Address
		peer address.Address

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			self: address.Address{Type: address.TypeTel, Target: "+15551110000"},
			peer: address.Address{Type: address.TypeTel, Target: "+15552220000"},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/self_and_peer",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"self":{"type":"tel","target":"+15551110000"},"peer":{"type":"tel","target":"+15552220000"}}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"f1b2c3d4-0005-0005-0005-000000000001"}`),
			},
			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-0005-0005-0005-000000000001"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1ConversationGetBySelfAndPeer(ctx, tt.self, tt.peer)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
