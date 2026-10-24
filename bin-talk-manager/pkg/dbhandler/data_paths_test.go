package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
)

// The tests below share the in-memory database of TestMain with the older tests, which neither
// use unique customers nor delete what they insert. Every test here therefore works on its own
// uuids and removes its rows through raw DELETE statements (the handler's own deletes are soft
// deletes), which keeps the tests safe for -count=N and for the other tests' row counts.

const (
	seedTime1 = "2020-01-01T00:00:00.000000Z"
	seedTime2 = "2020-01-02T00:00:00.000000Z"
	seedTime3 = "2020-01-03T00:00:00.000000Z"
)

func cleanupCustomer(t *testing.T, customerID uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		for _, table := range []string{"talk_participants", "talk_messages", "talk_chats"} {
			if _, err := dbTest.Exec("DELETE FROM "+table+" WHERE customer_id = ?", customerID.Bytes()); err != nil {
				t.Errorf("could not clean %s: %v", table, err)
			}
		}
	})
}

func seedChat(t *testing.T, id, customerID uuid.UUID, chatType chat.Type, tmCreate string, deleted bool) {
	t.Helper()

	var tmDelete any
	if deleted {
		tmDelete = tmCreate
	}

	_, err := dbTest.Exec(
		`INSERT INTO talk_chats (id, customer_id, type, name, detail, member_count, tm_create, tm_update, tm_delete) VALUES (?, ?, ?, '', '', 0, ?, ?, ?)`,
		id.Bytes(), customerID.Bytes(), string(chatType), tmCreate, tmCreate, tmDelete,
	)
	if err != nil {
		t.Fatalf("could not seed the chat: %v", err)
	}
}

func seedParticipant(t *testing.T, id, customerID, chatID uuid.UUID, ownerType string, ownerID uuid.UUID, tmJoined string) {
	t.Helper()

	_, err := dbTest.Exec(
		`INSERT INTO talk_participants (id, customer_id, chat_id, owner_type, owner_id, tm_joined) VALUES (?, ?, ?, ?, ?, ?)`,
		id.Bytes(), customerID.Bytes(), chatID.Bytes(), ownerType, ownerID.Bytes(), tmJoined,
	)
	if err != nil {
		t.Fatalf("could not seed the participant: %v", err)
	}
}

func chatIDs(chats []*chat.Chat) []uuid.UUID {
	res := []uuid.UUID{}
	for _, c := range chats {
		res = append(res, c.ID)
	}
	return res
}

func sameUUIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Test_ChatList_filters(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0001-11ef-9f2f-6f1c1d9c3e01")
	chat1 := uuid.FromStringOrNil("c1a0f000-0011-11ef-9f2f-6f1c1d9c3e01")
	chat2 := uuid.FromStringOrNil("c1a0f000-0012-11ef-9f2f-6f1c1d9c3e01")
	chat3 := uuid.FromStringOrNil("c1a0f000-0013-11ef-9f2f-6f1c1d9c3e01")
	ownerX := uuid.FromStringOrNil("c1a0f000-0021-11ef-9f2f-6f1c1d9c3e01")
	ownerY := uuid.FromStringOrNil("c1a0f000-0022-11ef-9f2f-6f1c1d9c3e01")
	cleanupCustomer(t, customerID)

	// chat1 (oldest) and chat3 (newest) have the participant X, chat2 (soft deleted) has Y
	seedChat(t, chat1, customerID, chat.TypeGroup, seedTime1, false)
	seedChat(t, chat2, customerID, chat.TypeGroup, seedTime2, true)
	seedChat(t, chat3, customerID, chat.TypeGroup, seedTime3, false)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0031-11ef-9f2f-6f1c1d9c3e01"), customerID, chat1, "agent", ownerX, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0032-11ef-9f2f-6f1c1d9c3e01"), customerID, chat2, "agent", ownerY, seedTime2)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0033-11ef-9f2f-6f1c1d9c3e01"), customerID, chat3, "agent", ownerX, seedTime3)

	tests := []struct {
		name    string
		filters func() map[chat.Field]any
		token   string
		size    uint64

		want []uuid.UUID
	}{
		{
			name:    "customer only, newest first, deleted included",
			filters: func() map[chat.Field]any { return map[chat.Field]any{chat.FieldCustomerID: customerID} },
			size:    10,
			want:    []uuid.UUID{chat3, chat2, chat1},
		},
		{
			name: "deleted false",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: customerID, chat.FieldDeleted: false}
			},
			size: 10,
			want: []uuid.UUID{chat3, chat1},
		},
		{
			name: "deleted true",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: customerID, chat.FieldDeleted: true}
			},
			size: 10,
			want: []uuid.UUID{chat2},
		},
		{
			name:    "size limits the page",
			filters: func() map[chat.Field]any { return map[chat.Field]any{chat.FieldCustomerID: customerID} },
			size:    1,
			want:    []uuid.UUID{chat3},
		},
		{
			name:    "token returns only older chats",
			filters: func() map[chat.Field]any { return map[chat.Field]any{chat.FieldCustomerID: customerID} },
			token:   seedTime3,
			size:    10,
			want:    []uuid.UUID{chat2, chat1},
		},
		{
			name:    "owner id filter joins the participants",
			filters: func() map[chat.Field]any { return map[chat.Field]any{chat.FieldOwnerID: ownerX} },
			size:    10,
			want:    []uuid.UUID{chat3, chat1},
		},
		{
			name: "owner type and owner id",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldOwnerType: "agent", chat.FieldOwnerID: ownerY}
			},
			size: 10,
			want: []uuid.UUID{chat2},
		},
		{
			name: "owner type that nobody has",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldOwnerType: "no-such-owner-type", chat.FieldOwnerID: ownerX}
			},
			size: 10,
			want: []uuid.UUID{},
		},
		{
			name: "customer and owner id",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: customerID, chat.FieldOwnerID: ownerX}
			},
			size: 10,
			want: []uuid.UUID{chat3, chat1},
		},
		{
			name: "customer, owner type and owner id",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: customerID, chat.FieldOwnerType: "agent", chat.FieldOwnerID: ownerY}
			},
			size: 10,
			want: []uuid.UUID{chat2},
		},
		{
			name: "customer, owner id and deleted",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: customerID, chat.FieldOwnerID: ownerX, chat.FieldDeleted: false}
			},
			size: 10,
			want: []uuid.UUID{chat3, chat1},
		},
		{
			name: "another customer and owner id",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldCustomerID: uuid.FromStringOrNil("c1a0f000-0009-11ef-9f2f-6f1c1d9c3e01"), chat.FieldOwnerID: ownerX}
			},
			size: 10,
			want: []uuid.UUID{},
		},
		{
			name: "owner filter combined with deleted",
			filters: func() map[chat.Field]any {
				return map[chat.Field]any{chat.FieldOwnerID: ownerY, chat.FieldDeleted: false}
			},
			size: 10,
			want: []uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}

			res, err := h.ChatList(context.Background(), tt.filters(), tt.token, tt.size)
			if err != nil {
				t.Fatalf("ChatList() error = %v", err)
			}
			if got := chatIDs(res); !sameUUIDs(got, tt.want) {
				t.Errorf("chats = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ChatList_invalidFilter_error(t *testing.T) {
	h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}

	if _, err := h.ChatList(context.Background(), map[chat.Field]any{chat.Field("no_such_column"): "x"}, "", 10); err == nil {
		t.Errorf("expected an error for a filter that is not a chat column")
	}
}

func Test_FindDirectChatByParticipants_noMatch(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0101-11ef-9f2f-6f1c1d9c3e02")
	otherCustomer := uuid.FromStringOrNil("c1a0f000-0102-11ef-9f2f-6f1c1d9c3e02")
	ownerA := uuid.FromStringOrNil("c1a0f000-0121-11ef-9f2f-6f1c1d9c3e02")
	ownerB := uuid.FromStringOrNil("c1a0f000-0122-11ef-9f2f-6f1c1d9c3e02")
	ownerC := uuid.FromStringOrNil("c1a0f000-0123-11ef-9f2f-6f1c1d9c3e02")
	cleanupCustomer(t, customerID)
	cleanupCustomer(t, otherCustomer)

	chatAB := uuid.FromStringOrNil("c1a0f000-0111-11ef-9f2f-6f1c1d9c3e02")      // direct, deleted
	chatABC := uuid.FromStringOrNil("c1a0f000-0112-11ef-9f2f-6f1c1d9c3e02")     // direct, three participants
	chatGroupAB := uuid.FromStringOrNil("c1a0f000-0113-11ef-9f2f-6f1c1d9c3e02") // group with exactly A and B
	chatAC := uuid.FromStringOrNil("c1a0f000-0114-11ef-9f2f-6f1c1d9c3e02")      // direct, A and C only

	seedChat(t, chatAB, customerID, chat.TypeDirect, seedTime1, true)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0131-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAB, "agent", ownerA, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0132-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAB, "agent", ownerB, seedTime1)

	seedChat(t, chatABC, customerID, chat.TypeDirect, seedTime1, false)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0133-11ef-9f2f-6f1c1d9c3e02"), customerID, chatABC, "agent", ownerA, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0134-11ef-9f2f-6f1c1d9c3e02"), customerID, chatABC, "agent", ownerB, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0135-11ef-9f2f-6f1c1d9c3e02"), customerID, chatABC, "agent", ownerC, seedTime1)

	seedChat(t, chatGroupAB, customerID, chat.TypeGroup, seedTime1, false)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0136-11ef-9f2f-6f1c1d9c3e02"), customerID, chatGroupAB, "agent", ownerA, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0137-11ef-9f2f-6f1c1d9c3e02"), customerID, chatGroupAB, "agent", ownerB, seedTime1)

	seedChat(t, chatAC, customerID, chat.TypeDirect, seedTime1, false)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0138-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAC, "agent", ownerA, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0139-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAC, "agent", ownerC, seedTime1)

	tests := []struct {
		name       string
		customerID uuid.UUID
		type1      string
		owner1     uuid.UUID
		type2      string
		owner2     uuid.UUID
	}{
		{"the only direct chat of the pair is soft deleted, the others have a different shape", customerID, "agent", ownerA, "agent", ownerB},
		{"the two owners do not share a chat", customerID, "agent", ownerB, "agent", ownerC},
		{"the owner type differs", customerID, "agent", ownerA, "customer", ownerC},
		{"another customer", otherCustomer, "agent", ownerA, "agent", ownerC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}

			res, err := h.FindDirectChatByParticipants(context.Background(), tt.customerID, tt.type1, tt.owner1, tt.type2, tt.owner2)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if res != nil {
				t.Errorf("chat = %v, want none", res.ID)
			}
		})
	}
}

func Test_FindDirectChatByParticipants_match(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0141-11ef-9f2f-6f1c1d9c3e02")
	ownerA := uuid.FromStringOrNil("c1a0f000-0161-11ef-9f2f-6f1c1d9c3e02")
	ownerB := uuid.FromStringOrNil("c1a0f000-0162-11ef-9f2f-6f1c1d9c3e02")
	chatAB := uuid.FromStringOrNil("c1a0f000-0151-11ef-9f2f-6f1c1d9c3e02")
	cleanupCustomer(t, customerID)

	seedChat(t, chatAB, customerID, chat.TypeDirect, seedTime1, false)
	if _, err := dbTest.Exec("UPDATE talk_chats SET member_count = 2 WHERE id = ?", chatAB.Bytes()); err != nil {
		t.Fatalf("could not set the member count: %v", err)
	}
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0171-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAB, "agent", ownerA, seedTime1)
	seedParticipant(t, uuid.FromStringOrNil("c1a0f000-0172-11ef-9f2f-6f1c1d9c3e02"), customerID, chatAB, "agent", ownerB, seedTime1)

	tests := []struct {
		name   string
		owner1 uuid.UUID
		owner2 uuid.UUID
	}{
		{"the pair in the stored order", ownerA, ownerB},
		{"the pair in the reverse order", ownerB, ownerA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}

			res, err := h.FindDirectChatByParticipants(context.Background(), customerID, "agent", tt.owner1, "agent", tt.owner2)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if res == nil {
				t.Fatalf("chat = nil, want %v", chatAB)
			}
			if res.ID != chatAB || res.CustomerID != customerID || res.Type != chat.TypeDirect || res.MemberCount != 2 {
				t.Errorf("chat = %+v, want id %v with member count 2", res, chatAB)
			}
		})
	}
}

func Test_ParticipantListByChatIDs_multiple(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0201-11ef-9f2f-6f1c1d9c3e03")
	chat1 := uuid.FromStringOrNil("c1a0f000-0211-11ef-9f2f-6f1c1d9c3e03")
	chat2 := uuid.FromStringOrNil("c1a0f000-0212-11ef-9f2f-6f1c1d9c3e03")
	chat3 := uuid.FromStringOrNil("c1a0f000-0213-11ef-9f2f-6f1c1d9c3e03")
	owner := uuid.FromStringOrNil("c1a0f000-0221-11ef-9f2f-6f1c1d9c3e03")
	cleanupCustomer(t, customerID)

	p1 := uuid.FromStringOrNil("c1a0f000-0231-11ef-9f2f-6f1c1d9c3e03")
	p2 := uuid.FromStringOrNil("c1a0f000-0232-11ef-9f2f-6f1c1d9c3e03")
	p3 := uuid.FromStringOrNil("c1a0f000-0233-11ef-9f2f-6f1c1d9c3e03")
	seedParticipant(t, p1, customerID, chat1, "agent", owner, seedTime1)
	seedParticipant(t, p2, customerID, chat2, "agent", owner, seedTime3)
	seedParticipant(t, p3, customerID, chat3, "agent", owner, seedTime2) // not requested

	h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}

	res, err := h.ParticipantListByChatIDs(context.Background(), []uuid.UUID{chat1, chat2})
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	got := []uuid.UUID{}
	for _, p := range res {
		got = append(got, p.ID)
	}
	// newest join first, and the participant of the chat that was not asked for is absent
	if want := []uuid.UUID{p2, p1}; !sameUUIDs(got, want) {
		t.Errorf("participants = %v, want %v", got, want)
	}
}

func Test_MessageCreate_parent(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0301-11ef-9f2f-6f1c1d9c3e04")
	chatID := uuid.FromStringOrNil("c1a0f000-0311-11ef-9f2f-6f1c1d9c3e04")
	parentID := uuid.FromStringOrNil("c1a0f000-0321-11ef-9f2f-6f1c1d9c3e04")
	childID := uuid.FromStringOrNil("c1a0f000-0322-11ef-9f2f-6f1c1d9c3e04")
	owner := uuid.FromStringOrNil("c1a0f000-0331-11ef-9f2f-6f1c1d9c3e04")
	cleanupCustomer(t, customerID)

	h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}
	ctx := context.Background()

	newMsg := func(id uuid.UUID, parent *uuid.UUID) *message.Message {
		return &message.Message{
			Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: owner},
			ChatID:   chatID,
			ParentID: parent,
			Type:     message.TypeNormal,
			Text:     "text",
		}
	}

	if err := h.MessageCreate(ctx, newMsg(parentID, nil)); err != nil {
		t.Fatalf("could not create the parent: %v", err)
	}
	if err := h.MessageCreate(ctx, newMsg(childID, &parentID)); err != nil {
		t.Fatalf("could not create the child: %v", err)
	}

	parent, err := h.MessageGet(ctx, parentID)
	if err != nil {
		t.Fatalf("MessageGet(parent) error = %v", err)
	}
	if parent.ParentID != nil {
		t.Errorf("the root message has the parent %v", *parent.ParentID)
	}

	child, err := h.MessageGet(ctx, childID)
	if err != nil {
		t.Fatalf("MessageGet(child) error = %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parentID {
		t.Errorf("parent of the reply = %v, want %v", child.ParentID, parentID)
	}
}

func Test_MessageReactions_selective(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1a0f000-0401-11ef-9f2f-6f1c1d9c3e05")
	chatID := uuid.FromStringOrNil("c1a0f000-0411-11ef-9f2f-6f1c1d9c3e05")
	messageID := uuid.FromStringOrNil("c1a0f000-0421-11ef-9f2f-6f1c1d9c3e05")
	ownerX := uuid.FromStringOrNil("c1a0f000-0431-11ef-9f2f-6f1c1d9c3e05")
	ownerY := uuid.FromStringOrNil("c1a0f000-0432-11ef-9f2f-6f1c1d9c3e05")
	cleanupCustomer(t, customerID)

	h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}
	ctx := context.Background()

	err := h.MessageCreate(ctx, &message.Message{
		Identity: commonidentity.Identity{ID: messageID, CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: ownerX},
		ChatID:   chatID,
		Type:     message.TypeNormal,
	})
	if err != nil {
		t.Fatalf("could not create the message: %v", err)
	}

	reaction := func(emoji string, owner uuid.UUID) string {
		return fmt.Sprintf(`{"emoji":%q,"owner_type":"agent","owner_id":%q}`, emoji, owner.String())
	}
	for _, r := range []string{reaction("smile", ownerX), reaction("smile", ownerY), reaction("heart", ownerX)} {
		if err := h.MessageAddReactionAtomic(ctx, messageID, r); err != nil {
			t.Fatalf("could not add the reaction %s: %v", r, err)
		}
	}

	type key struct {
		emoji string
		owner uuid.UUID
	}
	reactionsOf := func() []key {
		m, err := h.MessageGet(ctx, messageID)
		if err != nil {
			t.Fatalf("MessageGet() error = %v", err)
		}
		res := []key{}
		for _, r := range m.Metadata.Reactions {
			res = append(res, key{r.Emoji, r.OwnerID})
		}
		return res
	}
	equal := func(a, b []key) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	if got, want := reactionsOf(), []key{{"smile", ownerX}, {"smile", ownerY}, {"heart", ownerX}}; !equal(got, want) {
		t.Fatalf("after adding: %v, want %v", got, want)
	}

	// only the reaction that matches the emoji, the owner type and the owner is removed
	if err := h.MessageRemoveReactionAtomic(ctx, messageID, "smile", "agent", ownerX); err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if got, want := reactionsOf(), []key{{"smile", ownerY}, {"heart", ownerX}}; !equal(got, want) {
		t.Errorf("after removing smile of X: %v, want %v", got, want)
	}

	// the same emoji of another owner type is not the same reaction
	if err := h.MessageRemoveReactionAtomic(ctx, messageID, "heart", "customer", ownerX); err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if got, want := reactionsOf(), []key{{"smile", ownerY}, {"heart", ownerX}}; !equal(got, want) {
		t.Errorf("after removing a reaction of another owner type: %v, want %v", got, want)
	}

	// removing the remaining ones leaves none
	if err := h.MessageRemoveReactionAtomic(ctx, messageID, "smile", "agent", ownerY); err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if err := h.MessageRemoveReactionAtomic(ctx, messageID, "heart", "agent", ownerX); err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if got := reactionsOf(); len(got) != 0 {
		t.Errorf("reactions left = %v, want none", got)
	}
}

func Test_MessageReactions_unknownMessage_error(t *testing.T) {
	h := &dbHandler{db: dbTest, utilHandler: commonutil.NewUtilHandler()}
	ctx := context.Background()
	missing := uuid.FromStringOrNil("c1a0f000-0499-11ef-9f2f-6f1c1d9c3e05")

	if err := h.MessageAddReactionAtomic(ctx, missing, `{"emoji":"smile","owner_type":"agent","owner_id":"c1a0f000-0431-11ef-9f2f-6f1c1d9c3e05"}`); err == nil {
		t.Errorf("adding a reaction to a message that does not exist must fail")
	}
	if err := h.MessageRemoveReactionAtomic(ctx, missing, "smile", "agent", missing); err == nil {
		t.Errorf("removing a reaction from a message that does not exist must fail")
	}
}

var (
	registerGreatestOnce sync.Once
	greatestDBSeq        atomic.Int64
)

// openGreatestDB opens a private in-memory sqlite database that has the talk schema and the
// MySQL GREATEST function that ChatMemberCountDecrement relies on.
func openGreatestDB(t *testing.T) *sql.DB {
	t.Helper()

	registerGreatestOnce.Do(func() {
		sql.Register("sqlite3_talk_greatest", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("GREATEST", func(a, b int64) int64 {
					if a > b {
						return a
					}
					return b
				}, true)
			},
		})
	})

	db, err := sql.Open("sqlite3_talk_greatest", fmt.Sprintf("file:talk_greatest_%d?mode=memory&cache=shared", greatestDBSeq.Add(1)))
	if err != nil {
		t.Fatalf("could not open the db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("..", "..", "scripts", "database_scripts", "table_chats.sql"))
	if err != nil {
		t.Fatalf("could not read the schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("could not create the schema: %v", err)
	}

	return db
}

func Test_ChatMemberCountDecrement(t *testing.T) {
	chatID := uuid.FromStringOrNil("c1a0f000-0501-11ef-9f2f-6f1c1d9c3e06")
	otherID := uuid.FromStringOrNil("c1a0f000-0502-11ef-9f2f-6f1c1d9c3e06")
	customerID := uuid.FromStringOrNil("c1a0f000-0503-11ef-9f2f-6f1c1d9c3e06")

	tests := []struct {
		name  string
		start int

		want int
	}{
		{"decrements by one", 3, 2},
		{"does not go below zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openGreatestDB(t)

			for _, id := range []uuid.UUID{chatID, otherID} {
				_, err := db.Exec(
					`INSERT INTO talk_chats (id, customer_id, type, member_count, tm_create, tm_update) VALUES (?, ?, 'group', ?, ?, ?)`,
					id.Bytes(), customerID.Bytes(), tt.start, seedTime1, seedTime1,
				)
				if err != nil {
					t.Fatalf("could not seed the chat: %v", err)
				}
			}

			mc := gomock.NewController(t)
			mockUtil := commonutil.NewMockUtilHandler(mc)
			now := time.Date(2022, 5, 6, 7, 8, 9, 0, time.UTC)
			mockUtil.EXPECT().TimeNow().Return(&now)

			h := &dbHandler{db: db, utilHandler: mockUtil}
			if err := h.ChatMemberCountDecrement(context.Background(), chatID); err != nil {
				t.Fatalf("error = %v", err)
			}

			row := func(id uuid.UUID) (int, string) {
				var count int
				var tmUpdate string
				if err := db.QueryRow(`SELECT member_count, tm_update FROM talk_chats WHERE id = ?`, id.Bytes()).Scan(&count, &tmUpdate); err != nil {
					t.Fatalf("could not read the chat: %v", err)
				}
				return count, tmUpdate
			}

			count, tmUpdate := row(chatID)
			if count != tt.want {
				t.Errorf("member_count = %d, want %d", count, tt.want)
			}
			if !strings.HasPrefix(tmUpdate, "2022-05-06 07:08:09") {
				t.Errorf("tm_update = %s, want the time of the update (2022-05-06 07:08:09)", tmUpdate)
			}

			// the other chat is untouched
			otherCount, otherUpdate := row(otherID)
			if otherCount != tt.start || otherUpdate != seedTime1 {
				t.Errorf("the other chat changed: member_count = %d, tm_update = %s", otherCount, otherUpdate)
			}
		})
	}
}
