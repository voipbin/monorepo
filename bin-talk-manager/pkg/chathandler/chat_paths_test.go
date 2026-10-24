package chathandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
)

type chatMocks struct {
	db          *dbhandler.MockDBHandler
	notify      *notifyhandler.MockNotifyHandler
	participant *participanthandler.MockParticipantHandler
	util        *utilhandler.MockUtilHandler
}

func newChatHandlerWithMocks(t *testing.T) (*chatHandler, *chatMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	m := &chatMocks{
		db:          dbhandler.NewMockDBHandler(mc),
		notify:      notifyhandler.NewMockNotifyHandler(mc),
		participant: participanthandler.NewMockParticipantHandler(mc),
		util:        utilhandler.NewMockUtilHandler(mc),
	}

	return New(m.db, m.participant, m.notify, m.util).(*chatHandler), m
}

func Test_New(t *testing.T) {
	// a field that New() leaves unset would panic on first use, so each collaborator is exercised once
	h, m := newChatHandlerWithMocks(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("a1b2c3d4-0001-11ef-8e95-a3bc92af9795")

	m.db.EXPECT().ChatGet(ctx, id).Return(&chat.Chat{}, nil)
	if _, err := h.ChatGet(ctx, id); err != nil {
		t.Fatalf("ChatGet() error = %v", err)
	}

	m.db.EXPECT().ChatGet(ctx, id).Return(&chat.Chat{}, nil)
	m.db.EXPECT().ChatDelete(ctx, id).Return(nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, uuid.Nil, chat.EventTypeChatDeleted, gomock.Any())
	if _, err := h.ChatDelete(ctx, id); err != nil {
		t.Fatalf("ChatDelete() error = %v", err)
	}

	customerID := uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795")
	creatorID := uuid.FromStringOrNil("a1b2c3d4-0003-11ef-8e95-a3bc92af9795")
	m.util.EXPECT().UUIDCreate().Return(id)
	m.db.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
	m.participant.EXPECT().ParticipantAdd(ctx, id, creatorID, "agent").Return(&participant.Participant{}, nil)
	m.db.EXPECT().ChatGet(ctx, id).Return(&chat.Chat{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, chat.EventTypeChatCreated, gomock.Any())
	if _, err := h.ChatCreate(ctx, customerID, chat.TypeTalk, "", "", "agent", creatorID, nil); err != nil {
		t.Fatalf("ChatCreate() error = %v", err)
	}
}

func Test_ChatCreate_creatorWithoutID_error(t *testing.T) {
	h, _ := newChatHandlerWithMocks(t)

	// a creator type without a creator id is rejected before anything is looked up or stored
	res, err := h.ChatCreate(context.Background(),
		uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795"), chat.TypeGroup, "", "", "agent", uuid.Nil, nil)
	if err == nil || res != nil {
		t.Errorf("ChatCreate() = %v, %v, want an error and no chat", res, err)
	}
}

func Test_ChatCreate_direct_lookupFailureCreatesNewChat(t *testing.T) {
	h, m := newChatHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795")
	creatorID := uuid.FromStringOrNil("a1b2c3d4-0003-11ef-8e95-a3bc92af9795")
	otherID := uuid.FromStringOrNil("a1b2c3d4-0004-11ef-8e95-a3bc92af9795")
	newChatID := uuid.FromStringOrNil("a1b2c3d4-0005-11ef-8e95-a3bc92af9795")
	created := &chat.Chat{Identity: commonidentity.Identity{ID: newChatID, CustomerID: customerID}, Type: chat.TypeDirect}

	// the duplicate check is best effort: when it fails the chat is created anyway
	m.db.EXPECT().FindDirectChatByParticipants(ctx, customerID, "agent", creatorID, "agent", otherID).Return(nil, errors.New("lookup failure"))
	m.util.EXPECT().UUIDCreate().Return(newChatID)
	m.db.EXPECT().ChatCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *chat.Chat) error {
		if c.ID != newChatID || c.CustomerID != customerID || c.Type != chat.TypeDirect {
			t.Errorf("stored chat = %+v", c)
		}
		return nil
	})
	gomock.InOrder(
		m.participant.EXPECT().ParticipantAdd(ctx, newChatID, creatorID, "agent").Return(&participant.Participant{}, nil),
		m.participant.EXPECT().ParticipantAdd(ctx, newChatID, otherID, "agent").Return(&participant.Participant{}, nil),
	)
	m.db.EXPECT().ChatGet(ctx, newChatID).Return(created, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, chat.EventTypeChatCreated, created)

	res, err := h.ChatCreate(ctx, customerID, chat.TypeDirect, "", "", "agent", creatorID,
		[]participant.ParticipantInput{{OwnerType: "agent", OwnerID: otherID}})
	if err != nil || res != created {
		t.Errorf("ChatCreate() = %v, %v, want the new chat", res, err)
	}
}

func Test_ChatCreate_direct_existingChatReloadFails(t *testing.T) {
	h, m := newChatHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795")
	creatorID := uuid.FromStringOrNil("a1b2c3d4-0003-11ef-8e95-a3bc92af9795")
	otherID := uuid.FromStringOrNil("a1b2c3d4-0004-11ef-8e95-a3bc92af9795")
	existing := &chat.Chat{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-0006-11ef-8e95-a3bc92af9795"), CustomerID: customerID}}

	m.db.EXPECT().FindDirectChatByParticipants(ctx, customerID, "agent", creatorID, "agent", otherID).Return(existing, nil)
	m.db.EXPECT().ChatGet(ctx, existing.ID).Return(nil, errors.New("reload failure"))

	// nothing is created or announced; the chat that was found is returned as it is
	res, err := h.ChatCreate(ctx, customerID, chat.TypeDirect, "", "", "agent", creatorID,
		[]participant.ParticipantInput{{OwnerType: "agent", OwnerID: otherID}})
	if err != nil || res != existing {
		t.Errorf("ChatCreate() = %v, %v, want the existing chat", res, err)
	}
}

func Test_ChatCreate_participants(t *testing.T) {
	customerID := uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795")
	creatorID := uuid.FromStringOrNil("a1b2c3d4-0003-11ef-8e95-a3bc92af9795")
	memberA := uuid.FromStringOrNil("a1b2c3d4-0004-11ef-8e95-a3bc92af9795")
	memberB := uuid.FromStringOrNil("a1b2c3d4-0007-11ef-8e95-a3bc92af9795")
	newChatID := uuid.FromStringOrNil("a1b2c3d4-0005-11ef-8e95-a3bc92af9795")

	errAdd := fmt.Errorf("participant failure")

	tests := []struct {
		name         string
		creatorType  string
		creatorID    uuid.UUID
		participants []participant.ParticipantInput

		// the participants that must be added, in this order, and the failures to inject
		wantAdds []uuid.UUID
		failAdd  map[uuid.UUID]bool
		// reloadFails makes the final ChatGet fail
		reloadFails bool
	}{
		{
			name:        "creator is added before the listed participants",
			creatorType: "agent", creatorID: creatorID,
			participants: []participant.ParticipantInput{{OwnerType: "agent", OwnerID: memberA}, {OwnerType: "agent", OwnerID: memberB}},
			wantAdds:     []uuid.UUID{creatorID, memberA, memberB},
		},
		{
			name:        "creator listed among the participants is added once",
			creatorType: "agent", creatorID: creatorID,
			participants: []participant.ParticipantInput{{OwnerType: "agent", OwnerID: memberA}, {OwnerType: "agent", OwnerID: creatorID}},
			wantAdds:     []uuid.UUID{memberA, creatorID},
		},
		{
			name:         "a chat without a creator only has the listed participants",
			participants: []participant.ParticipantInput{{OwnerType: "agent", OwnerID: memberA}},
			wantAdds:     []uuid.UUID{memberA},
		},
		{
			name:        "failing to add the creator does not stop the others",
			creatorType: "agent", creatorID: creatorID,
			participants: []participant.ParticipantInput{{OwnerType: "agent", OwnerID: memberA}},
			wantAdds:     []uuid.UUID{creatorID, memberA},
			failAdd:      map[uuid.UUID]bool{creatorID: true},
		},
		{
			name:        "failing to add a listed participant does not stop the others",
			creatorType: "agent", creatorID: creatorID,
			participants: []participant.ParticipantInput{{OwnerType: "agent", OwnerID: memberA}, {OwnerType: "agent", OwnerID: memberB}},
			wantAdds:     []uuid.UUID{creatorID, memberA, memberB},
			failAdd:      map[uuid.UUID]bool{memberA: true},
		},
		{
			name:        "the stored chat is announced when it cannot be reloaded",
			creatorType: "agent", creatorID: creatorID,
			wantAdds:    []uuid.UUID{creatorID},
			reloadFails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newChatHandlerWithMocks(t)
			ctx := context.Background()

			m.util.EXPECT().UUIDCreate().Return(newChatID)
			m.db.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)

			calls := []any{}
			for _, ownerID := range tt.wantAdds {
				var err error
				if tt.failAdd[ownerID] {
					err = errAdd
				}
				calls = append(calls, m.participant.EXPECT().ParticipantAdd(ctx, newChatID, ownerID, "agent").Return(&participant.Participant{}, err))
			}
			gomock.InOrder(calls...)

			reloaded := &chat.Chat{Identity: commonidentity.Identity{ID: newChatID, CustomerID: customerID}, Name: "reloaded"}
			var announced *chat.Chat
			if tt.reloadFails {
				m.db.EXPECT().ChatGet(ctx, newChatID).Return(nil, errors.New("reload failure"))
			} else {
				m.db.EXPECT().ChatGet(ctx, newChatID).Return(reloaded, nil)
			}
			m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), chat.EventTypeChatCreated, gomock.Any()).
				Do(func(_ context.Context, _ uuid.UUID, _ string, data notifyhandler.WebhookEventMessage) {
					announced, _ = data.(*chat.Chat)
				})

			res, err := h.ChatCreate(ctx, customerID, chat.TypeGroup, "stored", "", tt.creatorType, tt.creatorID, tt.participants)
			if err != nil {
				t.Fatalf("ChatCreate() error = %v", err)
			}

			if announced == nil || announced != res {
				t.Fatalf("the announced chat %v is not the returned chat %v", announced, res)
			}
			if tt.reloadFails {
				if res.Name != "stored" || res.ID != newChatID || res.CustomerID != customerID {
					t.Errorf("fallback chat = %+v, want the chat that was stored", res)
				}
			} else if res != reloaded {
				t.Errorf("returned chat = %v, want the reloaded chat", res)
			}
		})
	}
}

func Test_ChatCreate_creatorIsMatchedByOwnerType(t *testing.T) {
	h, m := newChatHandlerWithMocks(t)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a1b2c3d4-0002-11ef-8e95-a3bc92af9795")
	sharedID := uuid.FromStringOrNil("a1b2c3d4-0003-11ef-8e95-a3bc92af9795")
	newChatID := uuid.FromStringOrNil("a1b2c3d4-0005-11ef-8e95-a3bc92af9795")

	// an agent and a customer may carry the same id; only the agent is the creator, so the agent
	// is still added next to the listed customer.
	m.util.EXPECT().UUIDCreate().Return(newChatID)
	m.db.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
	gomock.InOrder(
		m.participant.EXPECT().ParticipantAdd(ctx, newChatID, sharedID, "agent").Return(&participant.Participant{}, nil),
		m.participant.EXPECT().ParticipantAdd(ctx, newChatID, sharedID, "customer").Return(&participant.Participant{}, nil),
	)
	m.db.EXPECT().ChatGet(ctx, newChatID).Return(&chat.Chat{Identity: commonidentity.Identity{ID: newChatID, CustomerID: customerID}}, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, chat.EventTypeChatCreated, gomock.Any())

	_, err := h.ChatCreate(ctx, customerID, chat.TypeGroup, "", "", "agent", sharedID,
		[]participant.ParticipantInput{{OwnerType: "customer", OwnerID: sharedID}})
	if err != nil {
		t.Fatalf("ChatCreate() error = %v", err)
	}
}

func Test_ChatGet_notFound(t *testing.T) {
	h, m := newChatHandlerWithMocks(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("a1b2c3d4-0001-11ef-8e95-a3bc92af9795")

	m.db.EXPECT().ChatGet(ctx, id).Return(nil, fmt.Errorf("lookup: %w", dbhandler.ErrNotFound))

	res, err := h.ChatGet(ctx, id)
	if res != nil {
		t.Errorf("chat = %v, want none", res)
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want a typed error", err)
	}
	if ve.Reason != "CHAT_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
		t.Errorf("typed error = %s / %v, want CHAT_NOT_FOUND", ve.Reason, ve.Status)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("the cause is lost: %v", err)
	}
}
