package conversationhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Test_UpdateMetadata verifies the whole-struct-replace metadata update
// used by bin-contact-manager's Case-linking write paths
// (contact-case-management design §4.3/§4.4/§4.5). Mirrors
// bin-customer-manager's UpdateMetadata shape (models/customer, db.go).
//
// Critical invariant this test enforces: NO PublishWebhookEvent call.
// Metadata.ContactCaseID is purely internal case-linking plumbing and
// must never leak to a customer webhook -- gomock's strict
// no-unexpected-call behavior (no .EXPECT() registered for
// PublishWebhookEvent) enforces this.
func Test_UpdateMetadata(t *testing.T) {
	tests := []struct {
		name string

		id       uuid.UUID
		metadata conversation.Metadata

		responseConversation *conversation.Conversation
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("f1b2c3d4-0008-0008-0008-000000000001"),
			metadata: conversation.Metadata{
				ContactCaseID: func() *uuid.UUID {
					v := uuid.FromStringOrNil("f1b2c3d4-0008-0008-0008-000000000002")
					return &v
				}(),
			},

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1b2c3d4-0008-0008-0008-000000000001"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			expectFields := map[conversation.Field]any{
				conversation.FieldMetadata: tt.metadata,
			}
			mockDB.EXPECT().ConversationUpdate(ctx, tt.id, expectFields).Return(nil)
			mockDB.EXPECT().ConversationGet(ctx, tt.id).Return(tt.responseConversation, nil)
			// Deliberately no .EXPECT() for PublishWebhookEvent or
			// PublishEvent: gomock fails the test if either is
			// unexpectedly invoked.

			res, err := h.UpdateMetadata(ctx, tt.id, tt.metadata)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversation) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversation, res)
			}
		})
	}
}
