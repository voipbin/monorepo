package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// Test_ConversationV1ConversationUpdateMetadata verifies the
// whole-struct-replace metadata RPC client used by
// bin-contact-manager's Case-linking write paths
// (contact-case-management design §4.3/§4.4/§4.5).
func Test_ConversationV1ConversationUpdateMetadata(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-000b-000b-000b-000000000001")

	tests := []struct {
		name string

		conversationID uuid.UUID
		metadata       cvconversation.Metadata

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			conversationID: uuid.FromStringOrNil("f1b2c3d4-000b-000b-000b-000000000002"),
			metadata:       cvconversation.Metadata{ContactCaseID: &caseID},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/f1b2c3d4-000b-000b-000b-000000000002/metadata",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"metadata":{"contact_case_id":"f1b2c3d4-000b-000b-000b-000000000001"}}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"f1b2c3d4-000b-000b-000b-000000000002"}`),
			},
			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-000b-000b-000b-000000000002"),
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

			res, err := reqHandler.ConversationV1ConversationUpdateMetadata(ctx, tt.conversationID, tt.metadata)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
