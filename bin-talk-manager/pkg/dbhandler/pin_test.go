package dbhandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/pin"
)

// pinTestSeq makes deterministic, unique fixed-format UUIDs for the pin tests. The in-memory
// database is shared by all tests of the package, so every case needs its own ids.
var pinTestSeq atomic.Uint64

func pinTestID() uuid.UUID {
	n := pinTestSeq.Add(1)
	return uuid.FromStringOrNil(fmt.Sprintf("a1b2c3d4-0000-4000-8000-%012x", n))
}

func pinTestHandler() *dbHandler {
	return &dbHandler{
		db:          dbTest,
		redis:       nil,
		utilHandler: commonutil.NewUtilHandler(),
	}
}

func pinTestChat(t *testing.T, h *dbHandler, customerID uuid.UUID, chatType chat.Type) *chat.Chat {
	t.Helper()
	c := &chat.Chat{
		Identity: commonidentity.Identity{ID: pinTestID(), CustomerID: customerID},
		Type:     chatType,
		Name:     "pin test chat",
	}
	if err := h.ChatCreate(context.Background(), c); err != nil {
		t.Fatalf("Could not create chat. err: %v", err)
	}
	return c
}

func pinTestParticipant(t *testing.T, h *dbHandler, c *chat.Chat, ownerType string, ownerID uuid.UUID) *participant.Participant {
	t.Helper()
	p := &participant.Participant{
		Identity: commonidentity.Identity{ID: pinTestID(), CustomerID: c.CustomerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerType(ownerType), OwnerID: ownerID},
		ChatID:   c.ID,
	}
	if err := h.ParticipantCreate(context.Background(), p); err != nil {
		t.Fatalf("Could not create participant. err: %v", err)
	}
	return p
}

func pinTestMessage(t *testing.T, h *dbHandler, c *chat.Chat) *message.Message {
	t.Helper()
	m := &message.Message{
		Identity: commonidentity.Identity{ID: pinTestID(), CustomerID: c.CustomerID},
		Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: pinTestID()},
		ChatID:   c.ID,
		Type:     message.TypeNormal,
		Text:     "pinned text",
	}
	if err := h.MessageCreate(context.Background(), m); err != nil {
		t.Fatalf("Could not create message. err: %v", err)
	}
	return m
}

func pinTestNew(c *chat.Chat, ownerID uuid.UUID, targetType pin.TargetType, targetID uuid.UUID) *pin.Pin {
	return &pin.Pin{
		Identity:   commonidentity.Identity{ID: pinTestID(), CustomerID: c.CustomerID},
		Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
		ChatID:     c.ID,
		TargetType: targetType,
		TargetID:   targetID,
	}
}

func Test_PinCreate(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)
	m := pinTestMessage(t, h, c)

	tests := []struct {
		name string
		data *pin.Pin
	}{
		{"chat pin", pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)},
		{"message pin", pinTestNew(c, agentID, pin.TargetTypeMessage, m.ID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.PinCreate(ctx, tt.data)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.TMCreate == nil {
				t.Errorf("Wrong match. expect: tm_create set, got: nil")
			}

			got, err := h.PinGet(ctx, tt.data.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			got.TMCreate = nil
			expected := *tt.data
			expected.TMCreate = nil
			if !reflect.DeepEqual(&expected, got) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", &expected, got)
			}
		})
	}
}

// Verify (1): a duplicate create returns the existing row and leaves exactly one row.
func Test_PinCreate_duplicateReturnsExisting(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	c := pinTestChat(t, h, pinTestID(), chat.TypeTalk)
	agentID := pinTestID()

	first, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeChat, c.ID))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	second := pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)
	res, err := h.PinCreate(ctx, second)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != first.ID {
		t.Errorf("Wrong match. expect: %s, got: %s", first.ID, res.ID)
	}

	// the second id was never stored
	if _, err := h.PinGet(ctx, second.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}

	list, err := h.PinList(ctx, c.CustomerID, "agent", agentID, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", len(list))
	}
}

// A unique violation that is not the owner/target key (here the primary key) must not be reported
// as success: the re-read by owner and target finds nothing, so an error is returned.
func Test_PinCreate_duplicateWithoutExistingRow(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	c := pinTestChat(t, h, pinTestID(), chat.TypeTalk)
	first := pinTestNew(c, pinTestID(), pin.TargetTypeChat, c.ID)
	if _, err := h.PinCreate(ctx, first); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// same primary key, different owner: violates the primary key only
	clash := pinTestNew(c, pinTestID(), pin.TargetTypeChat, c.ID)
	clash.ID = first.ID
	res, err := h.PinCreate(ctx, clash)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: %v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_PinGet_notFound(t *testing.T) {
	h := pinTestHandler()
	if _, err := h.PinGet(context.Background(), pinTestID()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

// Verify (5): delete is scoped to the owner.
func Test_PinDelete(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)
	ownerID := pinTestID()
	otherID := pinTestID()

	tests := []struct {
		name       string
		customerID uuid.UUID
		ownerType  string
		ownerID    uuid.UUID
		unknownID  bool
		expectErr  error
		deleted    bool
	}{
		{"own pin", customerID, "agent", ownerID, false, nil, true},
		{"other agent", customerID, "agent", otherID, false, ErrNotFound, false},
		{"other customer", pinTestID(), "agent", ownerID, false, ErrNotFound, false},
		{"other owner type", customerID, "customer", ownerID, false, ErrNotFound, false},
		{"unknown id", customerID, "agent", ownerID, true, ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := h.PinCreate(ctx, pinTestNew(c, ownerID, pin.TargetTypeChat, c.ID))
			if err != nil {
				// the owner/target pair is reused by every case; the first deletion removes it
				t.Fatalf("Could not create pin. err: %v", err)
			}

			id := p.ID
			if tt.unknownID {
				id = pinTestID()
			}

			err = h.PinDelete(ctx, id, tt.customerID, tt.ownerType, tt.ownerID)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}

			_, errGet := h.PinGet(ctx, p.ID)
			if tt.deleted && !errors.Is(errGet, ErrNotFound) {
				t.Errorf("Wrong match. expect: deleted, got: %v", errGet)
			}
			if !tt.deleted {
				if errGet != nil {
					t.Errorf("Wrong match. expect: kept, got: %v", errGet)
				}
				// clean up for the next case
				if err := h.PinDelete(ctx, p.ID, customerID, "agent", ownerID); err != nil {
					t.Fatalf("Could not clean up. err: %v", err)
				}
			}
		})
	}
}

// Verify (7), the rule R side: the list shows a pin only if the chat is alive, belongs to the
// calling customer, and (the chat is a talk chat or the owner is an agent participant of it).
// The expected values are the rule R verdicts; the canAccessChat verdicts are in api-manager
// (serviceagent_talk_pin_test.go) with the same rows. The two differ only on two axes: a deleted
// chat and a customer mismatch.
func Test_PinList_ruleR(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	chatTypes := []chat.Type{chat.TypeTalk, chat.TypeDirect, chat.TypeGroup}
	for _, chatType := range chatTypes {
		for _, isParticipant := range []bool{true, false} {
			for _, chatDeleted := range []bool{false, true} {
				for _, customerMatch := range []bool{true, false} {
					name := fmt.Sprintf("%s participant=%t deleted=%t customerMatch=%t", chatType, isParticipant, chatDeleted, customerMatch)
					t.Run(name, func(t *testing.T) {
						callerCustomer := pinTestID()
						chatCustomer := callerCustomer
						if !customerMatch {
							chatCustomer = pinTestID()
						}

						c := pinTestChat(t, h, chatCustomer, chatType)
						agentID := pinTestID()
						if isParticipant {
							pinTestParticipant(t, h, c, "agent", agentID)
						}
						msg := pinTestMessage(t, h, c)

						// the pins are stored for the calling customer (the api layer sets it)
						chatPin := pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)
						chatPin.CustomerID = callerCustomer
						msgPin := pinTestNew(c, agentID, pin.TargetTypeMessage, msg.ID)
						msgPin.CustomerID = callerCustomer
						for _, p := range []*pin.Pin{chatPin, msgPin} {
							if _, err := h.PinCreate(ctx, p); err != nil {
								t.Fatalf("Could not create pin. err: %v", err)
							}
						}

						if chatDeleted {
							if err := h.ChatDelete(ctx, c.ID); err != nil {
								t.Fatalf("Could not delete chat. err: %v", err)
							}
						}

						res, err := h.PinList(ctx, callerCustomer, "agent", agentID, "", uuid.Nil, "", 100)
						if err != nil {
							t.Fatalf("Wrong match. expect: ok, got: %v", err)
						}

						visible := !chatDeleted && customerMatch && (chatType == chat.TypeTalk || isParticipant)
						expected := 0
						if visible {
							expected = 2
						}
						if len(res) != expected {
							t.Errorf("Wrong match. expect: %d, got: %d", expected, len(res))
						}
					})
				}
			}
		}
	}
}

// Verify (4): after the participant is removed the pins of a direct/group chat disappear, and they
// come back when the participant joins again.
func Test_PinList_participantRemovedAndRejoined(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeGroup)
	part := pinTestParticipant(t, h, c, "agent", agentID)

	if _, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	count := func() int {
		res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, "", 100)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		return len(res)
	}

	if got := count(); got != 1 {
		t.Fatalf("Wrong match. expect: 1, got: %d", got)
	}

	if err := h.ParticipantDelete(ctx, part.ID); err != nil {
		t.Fatalf("Could not remove participant. err: %v", err)
	}
	if got := count(); got != 0 {
		t.Errorf("Wrong match after removal. expect: 0, got: %d", got)
	}

	pinTestParticipant(t, h, c, "agent", agentID)
	if got := count(); got != 1 {
		t.Errorf("Wrong match after rejoin. expect: 1, got: %d", got)
	}
}

// A participant row of another owner type with the same id does not grant access.
func Test_PinList_participantOwnerTypeMatters(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeDirect)
	pinTestParticipant(t, h, c, "customer", agentID)

	if _, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(res))
	}
}

// Verify (3): a deleted message pin is not listed, a live one carries its message.
func Test_PinList_message(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)
	live := pinTestMessage(t, h, c)
	gone := pinTestMessage(t, h, c)

	for _, m := range []*message.Message{live, gone} {
		if _, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeMessage, m.ID)); err != nil {
			t.Fatalf("Could not create pin. err: %v", err)
		}
	}
	chatPin, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeChat, c.ID))
	if err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	if err := h.MessageDelete(ctx, gone.ID); err != nil {
		t.Fatalf("Could not delete message. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2, got: %d", len(res))
	}

	for _, p := range res {
		switch p.TargetType {
		case pin.TargetTypeMessage:
			if p.TargetID != live.ID {
				t.Errorf("Wrong match. expect: %s, got: %s", live.ID, p.TargetID)
			}
			if p.Message == nil || p.Message.ID != live.ID || p.Message.Text != "pinned text" {
				t.Errorf("Wrong match. expect: the pinned message, got: %v", p.Message)
			}
		case pin.TargetTypeChat:
			if p.ID != chatPin.ID {
				t.Errorf("Wrong match. expect: %s, got: %s", chatPin.ID, p.ID)
			}
			if p.Message != nil {
				t.Errorf("Wrong match. expect: nil message for a chat pin, got: %v", p.Message)
			}
		default:
			t.Errorf("Wrong match. unexpected target type: %s", p.TargetType)
		}
	}
}

// A pin owned by someone else never shows up, and neither does one of another customer.
func Test_PinList_ownerScope(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentA := pinTestID()
	agentB := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)

	pa, err := h.PinCreate(ctx, pinTestNew(c, agentA, pin.TargetTypeChat, c.ID))
	if err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}
	if _, err := h.PinCreate(ctx, pinTestNew(c, agentB, pin.TargetTypeChat, c.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", agentA, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[0].ID != pa.ID {
		t.Errorf("Wrong match. expect: only %s, got: %v", pa.ID, res)
	}

	res, err = h.PinList(ctx, pinTestID(), "agent", agentA, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0 for another customer, got: %d", len(res))
	}
}

func Test_PinList_filters(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c1 := pinTestChat(t, h, customerID, chat.TypeTalk)
	c2 := pinTestChat(t, h, customerID, chat.TypeTalk)
	m1 := pinTestMessage(t, h, c1)

	for _, p := range []*pin.Pin{
		pinTestNew(c1, agentID, pin.TargetTypeChat, c1.ID),
		pinTestNew(c1, agentID, pin.TargetTypeMessage, m1.ID),
		pinTestNew(c2, agentID, pin.TargetTypeChat, c2.ID),
	} {
		if _, err := h.PinCreate(ctx, p); err != nil {
			t.Fatalf("Could not create pin. err: %v", err)
		}
	}

	tests := []struct {
		name       string
		targetType pin.TargetType
		chatID     uuid.UUID
		expect     int
	}{
		{"no filter", "", uuid.Nil, 3},
		{"target_type chat", pin.TargetTypeChat, uuid.Nil, 2},
		{"target_type message", pin.TargetTypeMessage, uuid.Nil, 1},
		{"chat_id c1", "", c1.ID, 2},
		{"chat_id c2", "", c2.ID, 1},
		{"chat_id c1 and message", pin.TargetTypeMessage, c1.ID, 1},
		{"chat_id c2 and message", pin.TargetTypeMessage, c2.ID, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.PinList(ctx, customerID, "agent", agentID, tt.targetType, tt.chatID, "", 100)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, len(res))
			}
		})
	}
}

// Verify (6): newest first, the token is the tm_create of the last item of the previous page, and
// the page boundary neither repeats nor skips a pin.
func Test_PinList_pagination(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)

	var created []*pin.Pin
	for i := 0; i < 5; i++ {
		m := pinTestMessage(t, h, c)
		p, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeMessage, m.ID))
		if err != nil {
			t.Fatalf("Could not create pin. err: %v", err)
		}
		created = append(created, p)
		// distinct tm_create values; the token is tm_create only
		time.Sleep(2 * time.Millisecond)
	}

	// newest first
	expectOrder := []uuid.UUID{}
	for i := len(created) - 1; i >= 0; i-- {
		expectOrder = append(expectOrder, created[i].ID)
	}

	var got []uuid.UUID
	token := ""
	pages := 0
	for {
		res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, token, 2)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) == 0 {
			break
		}
		if len(res) > 2 {
			t.Fatalf("Wrong match. expect: at most 2, got: %d", len(res))
		}
		for _, p := range res {
			got = append(got, p.ID)
		}
		// the sqlite driver stores a time as this text
		token = res[len(res)-1].TMCreate.UTC().Format("2006-01-02 15:04:05.999999999-07:00")
		pages++
		if pages > 10 {
			t.Fatalf("Wrong match. pagination does not terminate")
		}
	}

	if !reflect.DeepEqual(expectOrder, got) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectOrder, got)
	}
	if pages != 3 {
		t.Errorf("Wrong match. expect: 3 pages, got: %d", pages)
	}
}

// A message pin is visible only if the message belongs to the chat of the pin.
func Test_PinList_messageOfAnotherChat(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c1 := pinTestChat(t, h, customerID, chat.TypeTalk)
	c2 := pinTestChat(t, h, customerID, chat.TypeTalk)
	m2 := pinTestMessage(t, h, c2)

	// the pin says chat c1, the message lives in chat c2
	if _, err := h.PinCreate(ctx, pinTestNew(c1, agentID, pin.TargetTypeMessage, m2.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(res))
	}
}

func Test_isPinDuplicateError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil", nil, false},
		{"mysql 1062", errors.New("Error 1062 (23000): Duplicate entry 'x' for key 'uq_talk_pins_owner_target'"), true},
		{"mysql duplicate text only", errors.New("Duplicate entry 'x' for key 'y'"), true},
		{"mysql code only", errors.New("Error 1062"), true},
		{"sqlite unique", errors.New("UNIQUE constraint failed: talk_pins.owner_type"), true},
		{"other error", errors.New("Error 1452 (23000): Cannot add or update a child row"), false},
		{"connection error", errors.New("driver: bad connection"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPinDuplicateError(tt.err); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

// pinAttachMessages drops the pin of a message that is deleted after the list query ran, and
// attaches the live message to its pin. It is called directly because PinList filters deleted
// messages in its own query already.
func Test_pinAttachMessages(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)
	live := pinTestMessage(t, h, c)
	gone := pinTestMessage(t, h, c)
	if err := h.MessageDelete(ctx, gone.ID); err != nil {
		t.Fatalf("Could not delete message. err: %v", err)
	}

	livePin := pinTestNew(c, agentID, pin.TargetTypeMessage, live.ID)
	gonePin := pinTestNew(c, agentID, pin.TargetTypeMessage, gone.ID)
	chatPin := pinTestNew(c, agentID, pin.TargetTypeChat, c.ID)

	res, err := h.pinAttachMessages(ctx, []*pin.Pin{gonePin, livePin, chatPin})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[0] != livePin || res[1] != chatPin {
		t.Fatalf("Wrong match. expect: [live message pin, chat pin], got: %v", res)
	}
	if livePin.Message == nil || livePin.Message.ID != live.ID {
		t.Errorf("Wrong match. expect: the live message, got: %v", livePin.Message)
	}
	if gonePin.Message != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", gonePin.Message)
	}
	if chatPin.Message != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", chatPin.Message)
	}
}

// The list query itself drops the pin of a deleted message, so such a pin does not take a slot
// of the page: with a page size of 2, the two live pins are returned although the deleted
// message's pin is the newest one.
func Test_PinList_deletedMessageDoesNotTakePageSlot(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	agentID := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeTalk)

	var live []uuid.UUID
	for i := 0; i < 2; i++ {
		m := pinTestMessage(t, h, c)
		p, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeMessage, m.ID))
		if err != nil {
			t.Fatalf("Could not create pin. err: %v", err)
		}
		live = append([]uuid.UUID{p.ID}, live...)
		time.Sleep(2 * time.Millisecond)
	}
	gone := pinTestMessage(t, h, c)
	if _, err := h.PinCreate(ctx, pinTestNew(c, agentID, pin.TargetTypeMessage, gone.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}
	if err := h.MessageDelete(ctx, gone.ID); err != nil {
		t.Fatalf("Could not delete message. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", agentID, "", uuid.Nil, "", 2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	var got []uuid.UUID
	for _, p := range res {
		got = append(got, p.ID)
	}
	if !reflect.DeepEqual(live, got) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", live, got)
	}
}

// A pin is hidden when the owner is not a participant, even if another agent is one. The
// participant condition must match the owner, not just any participant of the chat.
func Test_PinList_otherAgentParticipantDoesNotShowPin(t *testing.T) {
	h := pinTestHandler()
	ctx := context.Background()

	customerID := pinTestID()
	owner := pinTestID()
	other := pinTestID()
	c := pinTestChat(t, h, customerID, chat.TypeGroup)
	m := pinTestMessage(t, h, c)
	pinTestParticipant(t, h, c, "agent", other)
	if _, err := h.PinCreate(ctx, pinTestNew(c, owner, pin.TargetTypeMessage, m.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}
	if _, err := h.PinCreate(ctx, pinTestNew(c, owner, pin.TargetTypeChat, c.ID)); err != nil {
		t.Fatalf("Could not create pin. err: %v", err)
	}

	res, err := h.PinList(ctx, customerID, "agent", owner, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(res))
	}

	pinTestParticipant(t, h, c, "agent", owner)
	res, err = h.PinList(ctx, customerID, "agent", owner, "", uuid.Nil, "", 100)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("Wrong match. expect: 2 once the owner is a participant, got: %d", len(res))
	}
}
