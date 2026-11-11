package eventhandler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/models/notification"
	"monorepo/bin-push-manager/pkg/notificationhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

// every id is a made up value
var (
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testOtherCust  = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d31")
	testChatID     = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")
	testMessageID  = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e12")
	testConvID     = uuid.FromStringOrNil("6e8a0c2d-4f5b-4d79-9b1f-3d5f7b9d1f21")
	testCaseID     = uuid.FromStringOrNil("7f9b1d3e-5a6c-4e8a-8c2a-4e6a8c0e2a31")
	testNoteID     = uuid.FromStringOrNil("7f9b1d3e-5a6c-4e8a-8c2a-4e6a8c0e2a32")
)

func agentUUID(n int) uuid.UUID {
	return uuid.FromStringOrNil(fmt.Sprintf("b1d3f5a7-9c6e-4f08-9a4a-%012d", n))
}

type testEnv struct {
	h        *eventHandler
	mockReq  *requesthandler.MockRequestHandler
	mockSet  *settingshandler.MockSettingsHandler
	mockNoti *notificationhandler.MockNotificationHandler
	sent     *sentLog
}

// sentLog records the Send calls: the mock of Send runs in goroutines.
type sentLog struct {
	mu    sync.Mutex
	items []sentItem
}

type sentItem struct {
	CustomerID uuid.UUID
	AgentID    uuid.UUID
	DeviceID   uuid.UUID
	Kind       notification.Kind
	Data       map[string]string
}

func (l *sentLog) add(i sentItem) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, i)
}

// agents returns the sorted agent ids of the sends of the kind.
func (l *sentLog) agents(kind notification.Kind) []uuid.UUID {
	l.mu.Lock()
	defer l.mu.Unlock()

	res := []uuid.UUID{}
	for _, i := range l.items {
		if i.Kind == kind {
			res = append(res, i.AgentID)
		}
	}
	sort.Slice(res, func(a, b int) bool { return res[a].String() < res[b].String() })

	return res
}

func (l *sentLog) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.items)
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	e := &testEnv{
		mockReq:  requesthandler.NewMockRequestHandler(mc),
		mockSet:  settingshandler.NewMockSettingsHandler(mc),
		mockNoti: notificationhandler.NewMockNotificationHandler(mc),
		sent:     &sentLog{},
	}
	e.h = &eventHandler{
		reqHandler:          e.mockReq,
		settingsHandler:     e.mockSet,
		notificationHandler: e.mockNoti,
		eventTimeout:        5 * time.Second,
		parallelism:         4,
		maxRecipients:       200,
	}

	return e
}

// expectSend accepts any number of sends and records them.
func (e *testEnv) expectSend() {
	e.mockNoti.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, customerID, agentID, deviceID uuid.UUID, kind notification.Kind, data map[string]string) (*notification.SendResult, error) {
			e.sent.add(sentItem{customerID, agentID, deviceID, kind, data})
			return &notification.SendResult{DeviceCount: 1, SentCount: 1}, nil
		}).AnyTimes()
}

// passAll makes FilterRecipients keep every agent, and records the arguments.
func (e *testEnv) passAll(kind notification.Kind, chatID uuid.UUID, got *[]uuid.UUID) {
	e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, gomock.Any(), kind, chatID).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, ids []uuid.UUID, _ notification.Kind, _ uuid.UUID) ([]uuid.UUID, error) {
			if got != nil {
				*got = append([]uuid.UUID{}, ids...)
			}
			return ids, nil
		})
}

func counterValue(t *testing.T, kind notification.Kind) float64 {
	t.Helper()

	var m dto.Metric
	if err := promRecipientTruncated.WithLabelValues(string(kind)).Write(&m); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	return m.GetCounter().GetValue()
}

func participant(id uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID, joined time.Time, customerID uuid.UUID) *tkparticipant.Participant {
	return &tkparticipant.Participant{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Owner:    commonidentity.Owner{OwnerType: ownerType, OwnerID: ownerID},
		ChatID:   testChatID,
		TMJoined: &joined,
	}
}

func mentionText(ids ...uuid.UUID) string {
	s := "hello"
	for i, id := range ids {
		s += fmt.Sprintf(" [@name%d](mention:%s)", i, id)
	}
	return s
}

func chatMessage(author uuid.UUID, text string) *tkmessage.Message {
	return &tkmessage.Message{
		Identity: commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: author},
		ChatID:   testChatID,
		Type:     tkmessage.TypeNormal,
		Text:     text,
	}
}

func Test_EventChatMessageCreated_recipients(t *testing.T) {
	author := agentUUID(1)
	mentioned := agentUUID(2)         // mentioned and a participant
	mentionedOutsider := agentUUID(3) // mentioned, not a participant
	plain1 := agentUUID(4)
	plain2 := agentUUID(5)
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	e := newTestEnv(t)
	e.expectSend()

	participants := []*tkparticipant.Participant{
		participant(agentUUID(101), commonidentity.OwnerTypeAgent, author, t0, testCustomerID),
		participant(agentUUID(102), commonidentity.OwnerTypeAgent, mentioned, t0, testCustomerID),
		participant(agentUUID(103), commonidentity.OwnerTypeAgent, plain1, t0.Add(time.Minute), testCustomerID),
		participant(agentUUID(104), commonidentity.OwnerTypeAgent, plain2, t0.Add(2*time.Minute), testCustomerID),
		participant(agentUUID(105), commonidentity.OwnerTypeAgent, plain2, t0.Add(3*time.Minute), testCustomerID), // duplicate owner
		participant(agentUUID(106), commonidentity.OwnerType("customer"), agentUUID(9), t0, testCustomerID),       // not an agent
		participant(agentUUID(107), commonidentity.OwnerTypeAgent, uuid.Nil, t0, testCustomerID),                  // no owner id
		participant(agentUUID(108), commonidentity.OwnerTypeAgent, agentUUID(10), t0, testOtherCust),              // another customer
	}

	var gotMention, gotMessage []uuid.UUID
	e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return(participants, nil)
	e.passAll(notification.KindChatMentioned, uuid.Nil, &gotMention)
	e.passAll(notification.KindChatMessage, testChatID, &gotMessage)

	// the author mentions himself, a participant and an outsider
	text := mentionText(author, mentioned, mentionedOutsider)
	if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, text)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the mention goes to the mentioned agents even if they are not in the chat, never to the author
	if want := []uuid.UUID{mentioned, mentionedOutsider}; fmt.Sprint(gotMention) != fmt.Sprint(want) {
		t.Errorf("Wrong match. mention candidates expect: %v, got: %v", want, gotMention)
	}
	// the chat message goes to the other participants: not the author, not the mentioned, once each, newest first
	if want := []uuid.UUID{plain2, plain1}; fmt.Sprint(gotMessage) != fmt.Sprint(want) {
		t.Errorf("Wrong match. message candidates expect: %v, got: %v", want, gotMessage)
	}

	mentionSent := e.sent.agents(notification.KindChatMentioned)
	if len(mentionSent) != 2 {
		t.Errorf("Wrong match. got: %v", mentionSent)
	}
	messageSent := e.sent.agents(notification.KindChatMessage)
	if len(messageSent) != 2 {
		t.Errorf("Wrong match. got: %v", messageSent)
	}

	// the two sets never meet, and every send is scoped by the customer of the event and carries ids only
	for _, a := range mentionSent {
		for _, b := range messageSent {
			if a == b {
				t.Errorf("Wrong match. %s gets both kinds for one message", a)
			}
		}
	}
	for _, i := range e.sent.items {
		if i.CustomerID != testCustomerID {
			t.Errorf("Wrong match. customer: %s", i.CustomerID)
		}
		want := map[string]string{"chat_id": testChatID.String(), "message_id": testMessageID.String()}
		if fmt.Sprint(i.Data) != fmt.Sprint(want) {
			t.Errorf("Wrong match. data expect: %v, got: %v", want, i.Data)
		}
	}
}

func Test_EventChatMessageCreated_mentionSwitchedOffGetsNothingForThatMessage(t *testing.T) {
	author := agentUUID(1)
	m1 := agentUUID(2)
	p1 := agentUUID(3)
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	e := newTestEnv(t)
	e.expectSend()

	e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return([]*tkparticipant.Participant{
		participant(agentUUID(101), commonidentity.OwnerTypeAgent, m1, t0, testCustomerID),
		participant(agentUUID(102), commonidentity.OwnerTypeAgent, p1, t0, testCustomerID),
	}, nil)

	// m1 switched chat_mentioned off: the filter removes him from the mention
	e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{m1}, notification.KindChatMentioned, uuid.Nil).Return([]uuid.UUID{}, nil)
	// ... and he is not a candidate of chat_message either
	e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{p1}, notification.KindChatMessage, testChatID).Return([]uuid.UUID{p1}, nil)

	if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, mentionText(m1))); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := e.sent.agents(notification.KindChatMentioned); len(got) != 0 {
		t.Errorf("Wrong match. got: %v", got)
	}
	if got := e.sent.agents(notification.KindChatMessage); len(got) != 1 || got[0] != p1 {
		t.Errorf("Wrong match. got: %v", got)
	}
}

func Test_EventChatMessageCreated_ignored(t *testing.T) {
	tests := []struct {
		name string
		mod  func(m *tkmessage.Message)
	}{
		{"a system message", func(m *tkmessage.Message) { m.Type = tkmessage.TypeSystem }},
		{"an empty type", func(m *tkmessage.Message) { m.Type = "" }},
		{"no customer", func(m *tkmessage.Message) { m.CustomerID = uuid.Nil }},
		{"no chat", func(m *tkmessage.Message) { m.ChatID = uuid.Nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEnv(t) // strict mocks: any call fails the test
			m := chatMessage(agentUUID(1), mentionText(agentUUID(2)))
			tt.mod(m)

			if err := e.h.EventChatMessageCreated(context.Background(), m); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventChatMessageCreated_noAgentParticipantNoFilterCall(t *testing.T) {
	e := newTestEnv(t)
	e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return([]*tkparticipant.Participant{}, nil)

	// no mention and no candidate: the settings are not read at all
	if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(agentUUID(1), "hello")); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventChatMessageCreated_errors(t *testing.T) {
	author := agentUUID(1)
	m1 := agentUUID(2)

	t.Run("the participant list fails: the mention is still sent and the error is returned", func(t *testing.T) {
		e := newTestEnv(t)
		e.expectSend()

		e.passAll(notification.KindChatMentioned, uuid.Nil, nil)
		e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return(nil, fmt.Errorf("boom"))

		if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, mentionText(m1))); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
		if got := e.sent.agents(notification.KindChatMentioned); len(got) != 1 {
			t.Errorf("Wrong match. got: %v", got)
		}
	})

	t.Run("the filter of the mention fails: the chat message is still sent", func(t *testing.T) {
		e := newTestEnv(t)
		e.expectSend()

		t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
		p1 := agentUUID(3)
		e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return([]*tkparticipant.Participant{
			participant(agentUUID(101), commonidentity.OwnerTypeAgent, p1, t0, testCustomerID),
		}, nil)
		e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{m1}, notification.KindChatMentioned, uuid.Nil).Return(nil, fmt.Errorf("boom"))
		e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{p1}, notification.KindChatMessage, testChatID).Return([]uuid.UUID{p1}, nil)

		if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, mentionText(m1))); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
		if got := e.sent.agents(notification.KindChatMessage); len(got) != 1 {
			t.Errorf("Wrong match. got: %v", got)
		}
	})

	t.Run("a failing send is not an error of the event", func(t *testing.T) {
		e := newTestEnv(t)
		e.passAll(notification.KindChatMentioned, uuid.Nil, nil)
		e.mockNoti.EXPECT().Send(gomock.Any(), testCustomerID, m1, uuid.Nil, notification.KindChatMentioned, gomock.Any()).Return(nil, fmt.Errorf("boom"))
		e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return([]*tkparticipant.Participant{}, nil)

		if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, mentionText(m1))); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

// More recipients than the limit: the settings and mutes are applied first, then the oldest participants are cut.
func Test_EventChatMessageCreated_limit(t *testing.T) {
	author := agentUUID(1)
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	e := newTestEnv(t)
	e.expectSend()

	// 300 participants, joined one minute apart: participant i joined at t0 + i minutes. The agent of
	// participant i is agentUUID(1000+i). The agents 1000..1049 switched the kind off or muted the chat.
	participants := []*tkparticipant.Participant{}
	for i := 0; i < 300; i++ {
		participants = append(participants, participant(agentUUID(5000+i), commonidentity.OwnerTypeAgent, agentUUID(1000+i), t0.Add(time.Duration(i)*time.Minute), testCustomerID))
	}
	e.mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), testChatID).Return(participants, nil)

	e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, gomock.Any(), notification.KindChatMessage, testChatID).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, ids []uuid.UUID, _ notification.Kind, _ uuid.UUID) ([]uuid.UUID, error) {
			if len(ids) != 300 {
				t.Errorf("Wrong match. the filter must see all candidates, got: %d", len(ids))
			}
			off := map[uuid.UUID]bool{}
			for i := 0; i < 50; i++ {
				off[agentUUID(1000+i)] = true
			}
			res := []uuid.UUID{}
			for _, id := range ids {
				if !off[id] {
					res = append(res, id)
				}
			}
			return res, nil
		})

	before := counterValue(t, notification.KindChatMessage)
	if err := e.h.EventChatMessageCreated(context.Background(), chatMessage(author, "hello")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// 300 - 50 = 250 want it; the limit is 200; the 50 oldest of them are cut: agents 1050..1099
	got := e.sent.agents(notification.KindChatMessage)
	if len(got) != 200 {
		t.Fatalf("Wrong match. expect: 200 sends, got: %d", len(got))
	}
	cut := map[uuid.UUID]bool{}
	for i := 0; i < 100; i++ {
		cut[agentUUID(1000+i)] = true // the 50 switched off and the 50 oldest
	}
	for _, id := range got {
		if cut[id] {
			t.Errorf("Wrong match. %s must not be sent", id)
		}
	}
	if after := counterValue(t, notification.KindChatMessage); after-before != 1 {
		t.Errorf("Wrong match. the cut must be counted once, before: %v, after: %v", before, after)
	}
}

func Test_sortParticipants(t *testing.T) {
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	a := participant(agentUUID(1), commonidentity.OwnerTypeAgent, agentUUID(11), t0.Add(2*time.Minute), testCustomerID)
	b := participant(agentUUID(2), commonidentity.OwnerTypeAgent, agentUUID(12), t0.Add(time.Minute), testCustomerID)
	c1 := participant(agentUUID(3), commonidentity.OwnerTypeAgent, agentUUID(13), t0, testCustomerID)
	c2 := participant(agentUUID(4), commonidentity.OwnerTypeAgent, agentUUID(14), t0, testCustomerID)
	none := participant(agentUUID(5), commonidentity.OwnerTypeAgent, agentUUID(15), t0, testCustomerID)
	none.TMJoined = nil

	list := []*tkparticipant.Participant{none, c2, b, c1, a}
	sortParticipants(list)

	want := []*tkparticipant.Participant{a, b, c1, c2, none}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("Wrong match. index: %d, expect: %s, got: %s", i, want[i].ID, list[i].ID)
		}
	}
}

func Test_sendAll(t *testing.T) {
	agents := func(n int) []uuid.UUID {
		res := make([]uuid.UUID, 0, n)
		for i := 0; i < n; i++ {
			res = append(res, agentUUID(2000+i))
		}
		return res
	}

	t.Run("the number of sends running at once is bounded", func(t *testing.T) {
		e := newTestEnv(t)
		e.h.parallelism = 2

		var running, peak int32
		e.mockNoti.EXPECT().Send(gomock.Any(), testCustomerID, gomock.Any(), uuid.Nil, notification.KindCaseNote, gomock.Any()).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, notification.Kind, map[string]string) (*notification.SendResult, error) {
				n := atomic.AddInt32(&running, 1)
				for {
					p := atomic.LoadInt32(&peak)
					if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				atomic.AddInt32(&running, -1)
				return nil, nil
			}).Times(10)

		e.h.sendAll(context.Background(), testCustomerID, agents(10), notification.KindCaseNote, map[string]string{"case_id": "x"})
		if p := atomic.LoadInt32(&peak); p > 2 || p < 1 {
			t.Errorf("Wrong match. expect: peak <= 2, got: %d", p)
		}
	})

	t.Run("an expired event stops the sends that did not start", func(t *testing.T) {
		e := newTestEnv(t)
		e.h.parallelism = 1

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		e.mockNoti.EXPECT().Send(gomock.Any(), testCustomerID, gomock.Any(), uuid.Nil, notification.KindCaseNote, gomock.Any()).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, notification.Kind, map[string]string) (*notification.SendResult, error) {
				time.Sleep(60 * time.Millisecond)
				return nil, nil
			}).Times(1)

		skipped := e.h.sendAll(ctx, testCustomerID, agents(5), notification.KindCaseNote, nil)
		if skipped != 4 {
			t.Errorf("Wrong match. expect: 4 skipped, got: %d", skipped)
		}
	})

	t.Run("a panic in a send does not stop the others", func(t *testing.T) {
		e := newTestEnv(t)
		var calls int32
		e.mockNoti.EXPECT().Send(gomock.Any(), testCustomerID, gomock.Any(), uuid.Nil, notification.KindCaseNote, gomock.Any()).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, notification.Kind, map[string]string) (*notification.SendResult, error) {
				if atomic.AddInt32(&calls, 1) == 1 {
					panic("boom")
				}
				return nil, nil
			}).Times(3)

		e.h.sendAll(context.Background(), testCustomerID, agents(3), notification.KindCaseNote, nil)
	})

	t.Run("no agent no send", func(t *testing.T) {
		e := newTestEnv(t)
		if skipped := e.h.sendAll(context.Background(), testCustomerID, nil, notification.KindCaseNote, nil); skipped != 0 {
			t.Errorf("Wrong match. got: %d", skipped)
		}
	})
}

func conversationMessage(direction cvmessage.Direction) *cvmessage.Message {
	return &cvmessage.Message{
		Identity:       commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
		ConversationID: testConvID,
		Direction:      direction,
	}
}

func Test_EventConversationMessageCreated(t *testing.T) {
	owner := agentUUID(1)

	conversation := func(customerID uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) *cvconversation.Conversation {
		return &cvconversation.Conversation{
			Identity: commonidentity.Identity{ID: testConvID, CustomerID: customerID},
			Owner:    commonidentity.Owner{OwnerType: ownerType, OwnerID: ownerID},
		}
	}

	tests := []struct {
		name         string
		msg          *cvmessage.Message
		conv         *cvconversation.Conversation
		getErr       error
		filterKeeps  bool
		expectGet    bool
		expectFilter bool
		expectSend   bool
		expectErr    bool
	}{
		{"incoming to the agent owner", conversationMessage(cvmessage.DirectionIncoming), conversation(testCustomerID, commonidentity.OwnerTypeAgent, owner), nil, true, true, true, true, false},
		{"the owner switched the kind off", conversationMessage(cvmessage.DirectionIncoming), conversation(testCustomerID, commonidentity.OwnerTypeAgent, owner), nil, false, true, true, false, false},
		{"outgoing is ignored", conversationMessage(cvmessage.DirectionOutgoing), nil, nil, true, false, false, false, false},
		{"no direction is ignored", conversationMessage(cvmessage.DirectionNond), nil, nil, true, false, false, false, false},
		{"no owner", conversationMessage(cvmessage.DirectionIncoming), conversation(testCustomerID, commonidentity.OwnerTypeNone, uuid.Nil), nil, true, true, false, false, false},
		{"owner type agent without an id", conversationMessage(cvmessage.DirectionIncoming), conversation(testCustomerID, commonidentity.OwnerTypeAgent, uuid.Nil), nil, true, true, false, false, false},
		{"an owner type other than agent with an id", conversationMessage(cvmessage.DirectionIncoming), conversation(testCustomerID, commonidentity.OwnerType("ai"), owner), nil, true, true, false, false, false},
		{"the conversation belongs to another customer", conversationMessage(cvmessage.DirectionIncoming), conversation(testOtherCust, commonidentity.OwnerTypeAgent, owner), nil, true, true, false, false, false},
		{"the conversation lookup fails", conversationMessage(cvmessage.DirectionIncoming), nil, fmt.Errorf("boom"), true, true, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEnv(t)
			e.expectSend()

			if tt.expectGet {
				e.mockReq.EXPECT().ConversationV1ConversationGet(gomock.Any(), testConvID).Return(tt.conv, tt.getErr)
			}
			if tt.expectFilter {
				result := []uuid.UUID{}
				if tt.filterKeeps {
					result = []uuid.UUID{owner}
				}
				e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{owner}, notification.KindConversationMessage, uuid.Nil).Return(result, nil)
			}

			err := e.h.EventConversationMessageCreated(context.Background(), tt.msg)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if got := e.sent.count(); (got == 1) != tt.expectSend || got > 1 {
				t.Errorf("Wrong match. expect send: %v, got %d sends", tt.expectSend, got)
			}
			if tt.expectSend {
				i := e.sent.items[0]
				want := map[string]string{"conversation_id": testConvID.String(), "message_id": testMessageID.String()}
				if i.AgentID != owner || i.Kind != notification.KindConversationMessage || fmt.Sprint(i.Data) != fmt.Sprint(want) {
					t.Errorf("Wrong match. got: %v", i)
				}
			}
		})
	}

	t.Run("no customer", func(t *testing.T) {
		e := newTestEnv(t)
		m := conversationMessage(cvmessage.DirectionIncoming)
		m.CustomerID = uuid.Nil
		if err := e.h.EventConversationMessageCreated(context.Background(), m); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func caseNote(authorType string, author *uuid.UUID) *cmcasenote.CaseNote {
	return &cmcasenote.CaseNote{
		ID:         testNoteID,
		CustomerID: testCustomerID,
		CaseID:     testCaseID,
		AuthorType: authorType,
		AuthorID:   author,
	}
}

func caseWithOwner(customerID uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) *cmkase.Case {
	return &cmkase.Case{
		ID:         testCaseID,
		CustomerID: customerID,
		Owner:      commonidentity.Owner{OwnerType: ownerType, OwnerID: ownerID},
	}
}

func Test_EventCaseNoteCreated(t *testing.T) {
	owner := agentUUID(1)
	author := agentUUID(2)
	nilID := uuid.Nil

	tests := []struct {
		name         string
		note         *cmcasenote.CaseNote
		kase         *cmkase.Case
		getErr       error
		filterKeeps  bool
		expectGet    bool
		expectFilter bool
		expectSend   bool
		expectErr    bool
	}{
		{"an agent note goes to the owner", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testCustomerID, commonidentity.OwnerTypeAgent, owner), nil, true, true, true, true, false},
		{"the owner switched the kind off", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testCustomerID, commonidentity.OwnerTypeAgent, owner), nil, false, true, true, false, false},
		{"the owner wrote the note", caseNote(cmcasenote.AuthorTypeAgent, &owner), caseWithOwner(testCustomerID, commonidentity.OwnerTypeAgent, owner), nil, true, true, false, false, false},
		{"a system note", caseNote(cmcasenote.AuthorTypeSystem, nil), nil, nil, true, false, false, false, false},
		{"an unknown author type", caseNote("robot", &author), nil, nil, true, false, false, false, false},
		{"an agent note without an author id", caseNote(cmcasenote.AuthorTypeAgent, nil), nil, nil, true, false, false, false, false},
		{"an agent note with the nil author id", caseNote(cmcasenote.AuthorTypeAgent, &nilID), nil, nil, true, false, false, false, false},
		{"the case has no owner", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testCustomerID, commonidentity.OwnerTypeNone, uuid.Nil), nil, true, true, false, false, false},
		{"owner type agent without an id", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testCustomerID, commonidentity.OwnerTypeAgent, uuid.Nil), nil, true, true, false, false, false},
		{"an owner type other than agent with an id", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testCustomerID, commonidentity.OwnerType("ai"), owner), nil, true, true, false, false, false},
		{"the case belongs to another customer", caseNote(cmcasenote.AuthorTypeAgent, &author), caseWithOwner(testOtherCust, commonidentity.OwnerTypeAgent, owner), nil, true, true, false, false, false},
		{"the case lookup fails", caseNote(cmcasenote.AuthorTypeAgent, &author), nil, fmt.Errorf("boom"), true, true, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEnv(t)
			e.expectSend()

			if tt.expectGet {
				e.mockReq.EXPECT().ContactV1CaseGet(gomock.Any(), testCustomerID, testCaseID).Return(tt.kase, tt.getErr)
			}
			if tt.expectFilter {
				result := []uuid.UUID{}
				if tt.filterKeeps {
					result = []uuid.UUID{owner}
				}
				e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{owner}, notification.KindCaseNote, uuid.Nil).Return(result, nil)
			}

			err := e.h.EventCaseNoteCreated(context.Background(), tt.note)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if got := e.sent.count(); (got == 1) != tt.expectSend || got > 1 {
				t.Errorf("Wrong match. expect send: %v, got %d sends", tt.expectSend, got)
			}
			if tt.expectSend {
				i := e.sent.items[0]
				want := map[string]string{"case_id": testCaseID.String(), "note_id": testNoteID.String()}
				if i.AgentID != owner || i.Kind != notification.KindCaseNote || fmt.Sprint(i.Data) != fmt.Sprint(want) {
					t.Errorf("Wrong match. got: %v", i)
				}
			}
		})
	}
}

func Test_EventCaseAssigned(t *testing.T) {
	owner := agentUUID(1)

	tests := []struct {
		name         string
		event        *cmkase.CaseAssignedEvent
		filterKeeps  bool
		expectFilter bool
		expectSend   bool
	}{
		{"the new owner is told", &cmkase.CaseAssignedEvent{CaseID: testCaseID, CustomerID: testCustomerID, OwnerID: owner}, true, true, true},
		{"the new owner switched the kind off", &cmkase.CaseAssignedEvent{CaseID: testCaseID, CustomerID: testCustomerID, OwnerID: owner}, false, true, false},
		{"no owner", &cmkase.CaseAssignedEvent{CaseID: testCaseID, CustomerID: testCustomerID}, true, false, false},
		{"no customer", &cmkase.CaseAssignedEvent{CaseID: testCaseID, OwnerID: owner}, true, false, false},
		{"no case", &cmkase.CaseAssignedEvent{CustomerID: testCustomerID, OwnerID: owner}, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEnv(t)
			e.expectSend()

			if tt.expectFilter {
				result := []uuid.UUID{}
				if tt.filterKeeps {
					result = []uuid.UUID{owner}
				}
				e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{owner}, notification.KindCaseAssigned, uuid.Nil).Return(result, nil)
			}

			if err := e.h.EventCaseAssigned(context.Background(), tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if got := e.sent.count(); (got == 1) != tt.expectSend || got > 1 {
				t.Errorf("Wrong match. expect send: %v, got %d sends", tt.expectSend, got)
			}
			if tt.expectSend {
				i := e.sent.items[0]
				if i.AgentID != owner || i.Kind != notification.KindCaseAssigned || fmt.Sprint(i.Data) != fmt.Sprint(map[string]string{"case_id": testCaseID.String()}) {
					t.Errorf("Wrong match. got: %v", i)
				}
			}
		})
	}

	t.Run("the filter fails", func(t *testing.T) {
		e := newTestEnv(t)
		e.mockSet.EXPECT().FilterRecipients(gomock.Any(), testCustomerID, []uuid.UUID{owner}, notification.KindCaseAssigned, uuid.Nil).Return(nil, fmt.Errorf("boom"))

		if err := e.h.EventCaseAssigned(context.Background(), &cmkase.CaseAssignedEvent{CaseID: testCaseID, CustomerID: testCustomerID, OwnerID: owner}); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})
}

func Test_newEventHandler_defaults(t *testing.T) {
	h, ok := NewEventHandler(nil, nil, nil).(*eventHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *eventHandler")
	}
	if h.parallelism != defaultParallelism || h.maxRecipients != defaultMaxRecipients || h.eventTimeout != defaultEventTimeout {
		t.Errorf("Wrong match. got: %v, %v, %v", h.parallelism, h.maxRecipients, h.eventTimeout)
	}
	if defaultParallelism != 8 || defaultMaxRecipients != 200 || defaultEventTimeout != 30*time.Second {
		t.Errorf("Wrong match. the initial values are 8, 200 and 30s")
	}
}
