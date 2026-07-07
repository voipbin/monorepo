package conversationhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Test_GetBySelfAndPeer verifies the get-only lookup used by
// contact-case-management's §4.4 proactive-linking write path: it must
// call the DB layer's get-only ConversationGetBySelfAndPeer, and must
// NEVER fall back to creating a Conversation on a miss (that would fire
// a false conversation_created webhook for a thread that doesn't exist
// yet from the customer's perspective -- the exact defect the design
// doc's round-7 review caught in an earlier draft).
func Test_GetBySelfAndPeer(t *testing.T) {

	tests := []struct {
		name string

		self commonaddress.Address
		peer commonaddress.Address

		responseConversation *conversation.Conversation
	}{
		{
			"normal",

			commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15551110000",
			},
			commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15552220001",
			},

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1b2c3d4-0003-0003-0003-000000000001"),
					CustomerID: uuid.FromStringOrNil("f1b2c3d4-0003-0003-0003-000000000002"),
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

			mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, tt.self, tt.peer).Return(tt.responseConversation, nil)

			res, err := h.GetBySelfAndPeer(ctx, tt.self, tt.peer)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversation) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversation, res)
			}
		})
	}
}

// Test_GetBySelfAndPeer_NotFound_NeverCreates is the explicit negative test
// the design doc requires (round-7 correction, §4.4): on a miss, this
// method must return the not-found error as-is and must NOT call
// ConversationCreate or PublishWebhookEvent. gomock's strict "no
// unexpected calls" behavior (no .EXPECT() registered for those calls)
// enforces this: the test would fail if the implementation called them.
func Test_GetBySelfAndPeer_NotFound_NeverCreates(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551110000"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15552220002"}

	// No prior Conversation exists for this (self, peer) pair.
	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, self, peer).Return(nil, dbhandler.ErrNotFound)
	// Deliberately no .EXPECT() for ConversationCreate or PublishWebhookEvent:
	// gomock fails the test if either is unexpectedly invoked.

	res, err := h.GetBySelfAndPeer(ctx, self, peer)
	if err == nil {
		t.Fatalf("expected an error on miss, got nil (res=%v)", res)
	}
	if res != nil {
		t.Errorf("expected nil result on miss, got: %v", res)
	}
}
