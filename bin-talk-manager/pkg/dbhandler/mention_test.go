package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/mention"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

// sqliteTimeLayout is the form in which the sqlite test driver stores a time, so that a page token
// built from it compares equal to the stored value (the MySQL deployment compares datetime(6) values).
const sqliteTimeLayout = "2006-01-02 15:04:05.999999999-07:00"

func mentionTestHandler() *dbHandler {
	return &dbHandler{
		db:          dbTest,
		redis:       nil,
		utilHandler: commonutil.NewUtilHandler(),
	}
}

func mentionTestChat(t *testing.T, h *dbHandler, customerID uuid.UUID, chatType chat.Type, participantAgentIDs ...uuid.UUID) *chat.Chat {
	t.Helper()
	ctx := context.Background()

	c := &chat.Chat{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		Type:     chatType,
	}
	if err := h.ChatCreate(ctx, c); err != nil {
		t.Fatalf("Could not create the chat. err: %v", err)
	}

	for _, agentID := range participantAgentIDs {
		p := &participant.Participant{
			Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
			Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentID},
			ChatID:   c.ID,
		}
		if err := h.ParticipantCreate(ctx, p); err != nil {
			t.Fatalf("Could not create the participant. err: %v", err)
		}
	}

	return c
}

// mentionTestMessage stores a message that mentions the given agents through MessageCreateWithMentions.
func mentionTestMessage(t *testing.T, h *dbHandler, c *chat.Chat, text string, agentIDs ...uuid.UUID) *message.Message {
	t.Helper()

	m := &message.Message{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: c.CustomerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.Must(uuid.NewV4())},
		ChatID:   c.ID,
		Type:     message.TypeNormal,
		Text:     text,
	}
	mentions := []*mention.Mention{}
	for _, agentID := range agentIDs {
		// customer id and chat id are left empty: the store takes them from the message
		mentions = append(mentions, &mention.Mention{
			Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
			AgentID:  agentID,
		})
	}

	if err := h.MessageCreateWithMentions(context.Background(), m, mentions); err != nil {
		t.Fatalf("Could not create the message with mentions. err: %v", err)
	}
	// the stored creation times must differ for the ordering and the paging
	time.Sleep(3 * time.Millisecond)

	return m
}

func mentionTestIDs(list []*message.Message) []uuid.UUID {
	res := []uuid.UUID{}
	for _, m := range list {
		res = append(res, m.ID)
	}
	return res
}

func mentionTestExpectIDs(t *testing.T, got []*message.Message, expect ...*message.Message) {
	t.Helper()

	if len(got) != len(expect) {
		t.Fatalf("Wrong match. expect: %d messages, got: %d (%v)", len(expect), len(got), mentionTestIDs(got))
	}
	for i := range expect {
		if got[i].ID != expect[i].ID {
			t.Errorf("Wrong match at %d. expect: %s, got: %s", i, expect[i].ID, got[i].ID)
		}
	}
}

func Test_MessageCreateWithMentions(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	agentA := uuid.Must(uuid.NewV4())
	agentB := uuid.Must(uuid.NewV4())
	c := mentionTestChat(t, h, customerID, chat.TypeTalk)

	m := mentionTestMessage(t, h, c, "hello", agentA, agentB)

	// the message is stored
	got, err := h.MessageGet(ctx, m.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Text != "hello" {
		t.Errorf("Wrong match. expect: hello, got: %s", got.Text)
	}

	// one mention row per agent, carrying the message id, the chat id and the message creation time
	var msgTMCreate string
	if err := h.db.QueryRowContext(ctx, "SELECT tm_create FROM talk_messages WHERE id = ?", m.ID.Bytes()).Scan(&msgTMCreate); err != nil {
		t.Fatalf("Could not read the message time. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, "SELECT message_id, chat_id, agent_id, tm_create, customer_id FROM talk_mentions WHERE message_id = ? ORDER BY agent_id", m.ID.Bytes())
	if err != nil {
		t.Fatalf("Could not query the mentions. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	count := 0
	for rows.Next() {
		var messageID, chatID, agentID, rowCustomerID []byte
		var tmCreate string
		if err := rows.Scan(&messageID, &chatID, &agentID, &tmCreate, &rowCustomerID); err != nil {
			t.Fatalf("Could not scan. err: %v", err)
		}
		count++
		if uuid.FromBytesOrNil(messageID) != m.ID || uuid.FromBytesOrNil(chatID) != c.ID || uuid.FromBytesOrNil(rowCustomerID) != customerID {
			t.Errorf("Wrong ids. message: %x, chat: %x, customer: %x", messageID, chatID, rowCustomerID)
		}
		if a := uuid.FromBytesOrNil(agentID); a != agentA && a != agentB {
			t.Errorf("Wrong agent id: %s", a)
		}
		if tmCreate != msgTMCreate {
			t.Errorf("Wrong tm_create. expect the message's: %s, got: %s", msgTMCreate, tmCreate)
		}
	}
	if count != 2 {
		t.Errorf("Wrong match. expect: 2 mention rows, got: %d", count)
	}
}

// Test_MessageCreateWithMentions_atomic checks that a failing mention insert leaves no message behind.
func Test_MessageCreateWithMentions_atomic(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	agentA := uuid.Must(uuid.NewV4())
	c := mentionTestChat(t, h, customerID, chat.TypeTalk)

	m := &message.Message{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.Must(uuid.NewV4())},
		ChatID:   c.ID,
		Type:     message.TypeNormal,
		Text:     "should not remain",
	}
	// the same agent twice violates the unique key (message_id, agent_id) on the second insert
	mentions := []*mention.Mention{
		{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID}, ChatID: c.ID, AgentID: agentA},
		{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID}, ChatID: c.ID, AgentID: agentA},
	}

	if err := h.MessageCreateWithMentions(ctx, m, mentions); err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}

	// the failed transaction must give its connection back
	if inUse := h.db.Stats().InUse; inUse != 0 {
		t.Fatalf("Wrong match. expect: 0 connections in use after the failure, got: %d", inUse)
	}

	if _, err := h.MessageGet(ctx, m.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound (message rolled back), got: %v", err)
	}
	var n int
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM talk_mentions WHERE message_id = ?", m.ID.Bytes()).Scan(&n); err != nil {
		t.Fatalf("Could not count. err: %v", err)
	}
	if n != 0 {
		t.Errorf("Wrong match. expect: 0 mention rows, got: %d", n)
	}
}

func Test_MentionList(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())
	me := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())

	talkChat := mentionTestChat(t, h, customerID, chat.TypeTalk)
	directMine := mentionTestChat(t, h, customerID, chat.TypeDirect, me, other)
	directNotMine := mentionTestChat(t, h, customerID, chat.TypeDirect, other)

	m1 := mentionTestMessage(t, h, talkChat, "m1", me)
	m2 := mentionTestMessage(t, h, directMine, "m2", me, other)
	_ = mentionTestMessage(t, h, directNotMine, "not accessible", me)
	_ = mentionTestMessage(t, h, talkChat, "only other", other)
	m5 := mentionTestMessage(t, h, talkChat, "m5", me)

	tests := []struct {
		name       string
		customerID uuid.UUID
		agentID    uuid.UUID
		token      string
		size       uint64
		expect     []*message.Message
	}{
		{"the caller's mentions, newest first, inaccessible direct chat and others' mentions excluded", customerID, me, "", 100, []*message.Message{m5, m2, m1}},
		{"a different customer sees nothing", otherCustomerID, me, "", 100, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.MentionList(ctx, tt.customerID, tt.agentID, tt.token, tt.size)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			mentionTestExpectIDs(t, got, tt.expect...)
		})
	}

	t.Run("another agent sees only the messages that mention it", func(t *testing.T) {
		got, err := h.MentionList(ctx, customerID, other, "", 100)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		// m2 (direct chat with other as a participant) and the "only other" message in the talk chat
		if len(got) != 2 {
			t.Fatalf("Wrong match. expect: 2, got: %d", len(got))
		}
		for _, g := range got {
			if g.Text != "m2" && g.Text != "only other" {
				t.Errorf("Wrong message for the other agent: %s", g.Text)
			}
		}
	})

	t.Run("page boundary", func(t *testing.T) {
		page1, err := h.MentionList(ctx, customerID, me, "", 2)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		mentionTestExpectIDs(t, page1, m5, m2)

		token := page1[len(page1)-1].TMCreate.UTC().Format(sqliteTimeLayout)
		page2, err := h.MentionList(ctx, customerID, me, token, 2)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		mentionTestExpectIDs(t, page2, m1)

		token = page2[len(page2)-1].TMCreate.UTC().Format(sqliteTimeLayout)
		page3, err := h.MentionList(ctx, customerID, me, token, 2)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		mentionTestExpectIDs(t, page3)
	})
}

func Test_MentionList_accessRule(t *testing.T) {
	ctx := context.Background()

	type tc struct {
		name         string
		chatType     chat.Type
		participant  bool
		chatCustomer string // "same" or "other"
		deleteChat   bool
		deleteMsg    bool
		expectListed bool
	}
	tests := []tc{
		{"talk, same customer, non participant", chat.TypeTalk, false, "same", false, false, true},
		{"talk, same customer, participant", chat.TypeTalk, true, "same", false, false, true},
		{"direct, participant", chat.TypeDirect, true, "same", false, false, true},
		{"direct, non participant", chat.TypeDirect, false, "same", false, false, false},
		{"group, participant", chat.TypeGroup, true, "same", false, false, true},
		{"group, non participant", chat.TypeGroup, false, "same", false, false, false},
		{"talk, other customer's chat", chat.TypeTalk, false, "other", false, false, false},
		{"talk, other customer's chat, participant", chat.TypeTalk, true, "other", false, false, false},
		{"direct, other customer's chat, participant", chat.TypeDirect, true, "other", false, false, false},
		{"talk, chat deleted", chat.TypeTalk, false, "same", true, false, false},
		{"direct, participant, chat deleted", chat.TypeDirect, true, "same", true, false, false},
		{"talk, message deleted", chat.TypeTalk, false, "same", false, true, false},
		{"group, participant, message deleted", chat.TypeGroup, true, "same", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := mentionTestHandler()

			customerID := uuid.Must(uuid.NewV4())
			me := uuid.Must(uuid.NewV4())
			chatCustomerID := customerID
			if tt.chatCustomer == "other" {
				chatCustomerID = uuid.Must(uuid.NewV4())
			}

			participants := []uuid.UUID{}
			if tt.participant {
				participants = append(participants, me)
			}
			c := mentionTestChat(t, h, chatCustomerID, tt.chatType, participants...)

			m := &message.Message{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.Must(uuid.NewV4())},
				ChatID:   c.ID,
				Type:     message.TypeNormal,
				Text:     "x",
			}
			mentions := []*mention.Mention{{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
				ChatID:   c.ID,
				AgentID:  me,
			}}
			if err := h.MessageCreateWithMentions(ctx, m, mentions); err != nil {
				t.Fatalf("Could not create. err: %v", err)
			}

			if tt.deleteChat {
				if err := h.ChatDelete(ctx, c.ID); err != nil {
					t.Fatalf("Could not delete the chat. err: %v", err)
				}
			}
			if tt.deleteMsg {
				if err := h.MessageDelete(ctx, m.ID); err != nil {
					t.Fatalf("Could not delete the message. err: %v", err)
				}
			}

			got, err := h.MentionList(ctx, customerID, me, "", 100)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if listed := len(got) == 1; listed != tt.expectListed {
				t.Errorf("Wrong match. expect listed: %v, got: %v (%d rows)", tt.expectListed, listed, len(got))
			}
		})
	}
}

// Test_MentionList_participantLeaveRejoin checks that the mentions of a direct/group chat disappear
// when the agent leaves and come back when the agent joins again.
func Test_MentionList_participantLeaveRejoin(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	me := uuid.Must(uuid.NewV4())
	c := mentionTestChat(t, h, customerID, chat.TypeGroup, me)
	_ = mentionTestMessage(t, h, c, "hi", me)

	count := func() int {
		got, err := h.MentionList(ctx, customerID, me, "", 100)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		return len(got)
	}

	if count() != 1 {
		t.Fatalf("Wrong match. expect: 1 before leaving")
	}

	ps, err := h.ParticipantListByChatIDs(ctx, []uuid.UUID{c.ID})
	if err != nil || len(ps) != 1 {
		t.Fatalf("Could not list the participants. err: %v, len: %d", err, len(ps))
	}
	if err := h.ParticipantDelete(ctx, ps[0].ID); err != nil {
		t.Fatalf("Could not delete the participant. err: %v", err)
	}
	if count() != 0 {
		t.Errorf("Wrong match. expect: 0 after leaving")
	}

	p := &participant.Participant{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: me},
		ChatID:   c.ID,
	}
	if err := h.ParticipantCreate(ctx, p); err != nil {
		t.Fatalf("Could not rejoin. err: %v", err)
	}
	if count() != 1 {
		t.Errorf("Wrong match. expect: 1 after rejoining")
	}
}

// a participant of another owner type with the same id must not satisfy the agent participant condition
func Test_MentionList_nonAgentParticipant(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	me := uuid.Must(uuid.NewV4())
	c := mentionTestChat(t, h, customerID, chat.TypeDirect)

	p := &participant.Participant{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerType("customer"), OwnerID: me},
		ChatID:   c.ID,
	}
	if err := h.ParticipantCreate(ctx, p); err != nil {
		t.Fatalf("Could not create the participant. err: %v", err)
	}
	_ = mentionTestMessage(t, h, c, "x", me)

	got, err := h.MentionList(ctx, customerID, me, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(got))
	}
}

// a mention row of another customer must not be returned even when its message and chat belong to the caller
func Test_MentionList_mentionRowCustomer(t *testing.T) {
	h := mentionTestHandler()
	ctx := context.Background()

	customerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())
	me := uuid.Must(uuid.NewV4())
	c := mentionTestChat(t, h, customerID, chat.TypeTalk)
	m := mentionTestMessage(t, h, c, "x", me)

	if _, err := h.db.ExecContext(ctx, "UPDATE talk_mentions SET customer_id = ? WHERE message_id = ?", otherCustomerID.Bytes(), m.ID.Bytes()); err != nil {
		t.Fatalf("Could not update. err: %v", err)
	}

	got, err := h.MentionList(ctx, customerID, me, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(got))
	}
}
