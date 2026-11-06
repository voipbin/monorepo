package messagehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

// Test_db_eventsHaveNoProviderKeys pins that every event that the database layer publishes carries a copy without the
// provider keys of the targets, while the message returned to the caller and stored keeps them.
func Test_db_eventsHaveNoProviderKeys(t *testing.T) {
	newRow := func() *message.Message {
		m := storedMessage(withParts(testTargets(usDest1), 1))
		m.Targets[0].ProviderReferenceID = "4031a1af-0000-4000-8000-0000000000aa"
		m.Targets[0].ProviderParts = 2
		return m
	}

	check := func(t *testing.T, name string, run func(h *messageHandler, ctx context.Context, db *dbhandler.MockDBHandler, row *message.Message) *message.Message, eventType string) {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			db := dbhandler.NewMockDBHandler(mc)
			notify := notifyhandler.NewMockNotifyHandler(mc)
			h := &messageHandler{db: db, notifyHandler: notify}
			ctx := context.Background()
			row := newRow()

			var published *message.Message
			notify.EXPECT().PublishWebhookEvent(ctx, testCustomerID, eventType, gomock.Any()).Do(func(_ context.Context, _ uuid.UUID, _ string, body any) {
				published = body.(*message.Message)
			})

			returned := run(h, ctx, db, row)

			if published == nil {
				t.Fatalf("Nothing was published")
			}
			if published.Targets[0].ProviderReferenceID != "" || published.Targets[0].ProviderParts != 0 {
				t.Errorf("The event leaks the provider keys. got: %v", published.Targets[0])
			}
			if returned.Targets[0].ProviderReferenceID == "" || returned.Targets[0].ProviderParts != 2 {
				t.Errorf("The returned message lost the provider keys. got: %v", returned.Targets[0])
			}
			if published.Targets[0].Parts != 1 || published.Targets[0].Status != target.StatusQueued {
				t.Errorf("The public fields must stay. got: %v", published.Targets[0])
			}
		})
	}

	check(t, "dbCreate", func(h *messageHandler, ctx context.Context, db *dbhandler.MockDBHandler, row *message.Message) *message.Message {
		db.EXPECT().MessageCreate(ctx, row).Return(nil)
		db.EXPECT().MessageGet(ctx, row.ID).Return(row, nil)
		res, _ := h.dbCreate(ctx, row)
		return res
	}, message.EventTypeMessageCreated)

	check(t, "dbUpdateMedias", func(h *messageHandler, ctx context.Context, db *dbhandler.MockDBHandler, row *message.Message) *message.Message {
		db.EXPECT().MessageUpdate(ctx, row.ID, gomock.Any()).Return(nil)
		db.EXPECT().MessageGet(ctx, row.ID).Return(row, nil)
		res, _ := h.dbUpdateMedias(ctx, row.ID, []string{"a"})
		return res
	}, message.EventTypeMessageUpdated)

	check(t, "dbDelete", func(h *messageHandler, ctx context.Context, db *dbhandler.MockDBHandler, row *message.Message) *message.Message {
		db.EXPECT().MessageDelete(ctx, row.ID).Return(nil)
		db.EXPECT().MessageGet(ctx, row.ID).Return(row, nil)
		res, _ := h.dbDelete(ctx, row.ID)
		return res
	}, message.EventTypeMessageDeleted)
}
