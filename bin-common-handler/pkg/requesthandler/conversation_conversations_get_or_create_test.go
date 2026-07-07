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

// Test_ConversationV1ConversationGetOrCreateBySelfAndPeer verifies the
// get-or-create RPC client used by bin-contact-manager's agent-send
// Case-linked messaging path (contact-case-management design §4.5,
// round-12 correction). Distinct client method from
// ConversationV1ConversationGetBySelfAndPeer -- must hit the
// get_or_create_by_self_and_peer route, not self_and_peer.
func Test_ConversationV1ConversationGetOrCreateBySelfAndPeer(t *testing.T) {
	tests := []struct {
		name string

		customerID       uuid.UUID
		conversationType cvconversation.Type
		dialogID         string
		self             address.Address
		peer             address.Address

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			customerID:       uuid.FromStringOrNil("f1b2c3d4-0007-0007-0007-000000000001"),
			conversationType: cvconversation.TypeMessage,
			dialogID:         "",
			self:             address.Address{Type: address.TypeTel, Target: "+15551110000"},
			peer:             address.Address{Type: address.TypeTel, Target: "+15552220000"},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/get_or_create_by_self_and_peer",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"f1b2c3d4-0007-0007-0007-000000000001","type":"message","self":{"type":"tel","target":"+15551110000"},"peer":{"type":"tel","target":"+15552220000"}}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"f1b2c3d4-0007-0007-0007-000000000002"}`),
			},
			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-0007-0007-0007-000000000002"),
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

			res, err := reqHandler.ConversationV1ConversationGetOrCreateBySelfAndPeer(
				ctx, tt.customerID, tt.conversationType, tt.dialogID, tt.self, tt.peer,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
