package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

// Test_processV1ConversationsIDMetadataPut verifies the dedicated
// whole-struct-replace metadata route used by bin-contact-manager's
// Case-linking write paths (contact-case-management design
// §4.3/§4.4/§4.5). Must call conversationHandler.UpdateMetadata, NOT the
// general Update.
func Test_processV1ConversationsIDMetadataPut(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-000a-000a-000a-000000000001")

	tests := []struct {
		name string

		request *sock.Request

		expectID       uuid.UUID
		expectMetadata conversation.Metadata

		responseConversation *conversation.Conversation

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/conversations/f1b2c3d4-000a-000a-000a-000000000002/metadata",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"metadata":{"contact_case_id":"f1b2c3d4-000a-000a-000a-000000000001"}}`),
			},

			expectID:       uuid.FromStringOrNil("f1b2c3d4-000a-000a-000a-000000000002"),
			expectMetadata: conversation.Metadata{ContactCaseID: &caseID},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-000a-000a-000a-000000000002"),
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f1b2c3d4-000a-000a-000a-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{},"tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				UpdateMetadata(gomock.Any(), tt.expectID, tt.expectMetadata).
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
