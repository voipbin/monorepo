package messagehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
)

// Test_Create_ReattachesCaseIDAfterDBRoundTrip verifies Task 1.7's core
// mechanism: Message.CaseID is deliberately db:"-" (not a persisted
// column -- it's a point-in-time event hint, not message state), so the
// post-create MessageGet re-read from the DB always comes back with
// CaseID unset. Create must re-attach args.CaseID onto that re-read
// result before publishing the event, or the case_id hint would
// silently vanish from every conversation_message_created event.
func Test_Create_ReattachesCaseIDAfterDBRoundTrip(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-000d-000d-000d-000000000001")
	responseUUID := uuid.FromStringOrNil("f1b2c3d4-000d-000d-000d-000000000002")
	customerID := uuid.FromStringOrNil("f1b2c3d4-000d-000d-000d-000000000003")

	// Simulates the DB layer: MessageGet returns a fresh struct with no
	// CaseID set (as if freshly scanned from a row with no case_id column).
	dbReadResult := &message.Message{
		Identity: commonidentity.Identity{
			ID:         responseUUID,
			CustomerID: customerID,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &messageHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, responseUUID).Return(dbReadResult, nil)

	// The published event must carry CaseID, proving Create re-attached
	// it after the DB round-trip.
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, gomock.Any()).
		Do(func(_ context.Context, _ uuid.UUID, _ string, published notifyhandler.WebhookMessage) {
			publishedMsg, ok := published.(*message.Message)
			if !ok {
				t.Fatalf("expected *message.Message, got %T", published)
			}
			if publishedMsg.CaseID == nil || !reflect.DeepEqual(*publishedMsg.CaseID, caseID) {
				t.Errorf("expected published event to carry CaseID: %v, got: %v", caseID, publishedMsg.CaseID)
			}
		})

	res, err := h.Create(ctx, MessageCreateArgs{
		ID:         uuid.Nil,
		CustomerID: customerID,
		CaseID:     &caseID,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.CaseID == nil || !reflect.DeepEqual(*res.CaseID, caseID) {
		t.Errorf("expected returned message to carry CaseID: %v, got: %v", caseID, res.CaseID)
	}
}
