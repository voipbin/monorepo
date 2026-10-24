package linehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

func testHookAccount() *account.Account {
	return &account.Account{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000002"),
		},
		Secret: "test-secret",
		Token:  "test-token",
	}
}

// testHookConversation returns a conversation whose Self and Peer differ, so a swapped source/destination is detectable.
func testHookConversation() conversation.Conversation {
	return conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000002"),
		},
		Type:     conversation.TypeLine,
		DialogID: "Utest-user",
		Self:     commonaddress.Address{Type: commonaddress.TypeLine, Target: "", TargetName: "Me"},
		Peer:     commonaddress.Address{Type: commonaddress.TypeLine, Target: "Utest-user", TargetName: "Test User"},
	}
}

func Test_Hook_invalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := NewLineHandler(requesthandler.NewMockRequestHandler(mc))

	res, err := h.Hook(context.Background(), testHookAccount(), []byte(`{"events": [`))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_Hook(t *testing.T) {
	cv := testHookConversation()
	createdMessage := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000001"),
		},
	}

	tests := []struct {
		name string
		data string

		expectList   bool
		expectText   string
		expectResult int
	}{
		{
			name:         "empty events returns an empty result",
			data:         `{"events":[]}`,
			expectResult: 0,
		},
		{
			name:         "unsupported event type is skipped",
			data:         `{"events":[{"type":"unfollow","source":{"type":"user","userId":"Utest-user"}}]}`,
			expectResult: 0,
		},
		{
			name:         "follow event without a usable dialog id is skipped before any client call",
			data:         `{"events":[{"type":"follow","source":{"type":"unknown"}}]}`,
			expectResult: 0,
		},
		{
			name:         "text message creates an incoming message",
			data:         `{"events":[{"type":"message","source":{"type":"user","userId":"Utest-user"},"message":{"type":"text","id":"1","text":"hello line"}}]}`,
			expectList:   true,
			expectText:   "hello line",
			expectResult: 1,
		},
		{
			name:         "non-text message creates an incoming message with empty text",
			data:         `{"events":[{"type":"message","source":{"type":"user","userId":"Utest-user"},"message":{"type":"image","id":"2","contentProvider":{"type":"line"}}}]}`,
			expectList:   true,
			expectText:   "",
			expectResult: 1,
		},
		{
			name:         "unsupported event is skipped while the following message is still handled",
			data:         `{"events":[{"type":"unfollow","source":{"type":"user","userId":"Utest-user"}},{"type":"message","source":{"type":"user","userId":"Utest-user"},"message":{"type":"text","id":"3","text":"second"}}]}`,
			expectList:   true,
			expectText:   "second",
			expectResult: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &lineHandler{reqHandler: mockReq}

			ctx := context.Background()
			ac := testHookAccount()

			if tt.expectList {
				mockReq.EXPECT().ConversationV1ConversationList(ctx, "", uint64(1), map[conversation.Field]any{
					conversation.FieldDeleted:  false,
					conversation.FieldType:     conversation.TypeLine,
					conversation.FieldDialogID: "Utest-user",
				}).Return([]conversation.Conversation{cv}, nil)

				mockReq.EXPECT().ConversationV1MessageCreate(
					ctx,
					uuid.Nil,
					cv.CustomerID,
					cv.ID,
					message.DirectionIncoming,
					message.StatusDone,
					message.ReferenceTypeLine,
					uuid.Nil,
					"",
					tt.expectText,
					[]media.Media{},
					cv.Peer,
					cv.Self,
				).Return(createdMessage, nil)
			}

			res, err := h.Hook(ctx, ac, []byte(tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectResult {
				t.Fatalf("Wrong result count. expect: %d, got: %d", tt.expectResult, len(res))
			}
			if tt.expectResult == 0 {
				return
			}

			if !reflect.DeepEqual(res[0].Conversation, &cv) {
				t.Errorf("Wrong conversation. expect: %v, got: %v", &cv, res[0].Conversation)
			}
			if res[0].Message != createdMessage {
				t.Errorf("Wrong message. expect: %v, got: %v", createdMessage, res[0].Message)
			}
		})
	}
}

func Test_hookEventTypeMessage(t *testing.T) {
	cv := testHookConversation()

	textEvent := func(source *linebot.EventSource) *linebot.Event {
		return &linebot.Event{
			Type:    linebot.EventTypeMessage,
			Source:  source,
			Message: &linebot.TextMessage{Text: "hi"},
		}
	}

	tests := []struct {
		name  string
		event *linebot.Event

		expectDialogID string
		listRes        []conversation.Conversation
		listErr        error
		createErr      error

		expectErr bool
	}{
		{
			name:      "source type that carries no dialog id returns error without any request",
			event:     textEvent(&linebot.EventSource{Type: linebot.EventSourceType("unknown")}),
			expectErr: true,
		},
		{
			name:           "conversation list error is returned",
			event:          textEvent(&linebot.EventSource{Type: linebot.EventSourceTypeUser, UserID: "Utest-user"}),
			expectDialogID: "Utest-user",
			listErr:        fmt.Errorf("list failed"),
			expectErr:      true,
		},
		{
			name:           "no conversation found returns error",
			event:          textEvent(&linebot.EventSource{Type: linebot.EventSourceTypeUser, UserID: "Utest-user"}),
			expectDialogID: "Utest-user",
			listRes:        []conversation.Conversation{},
			expectErr:      true,
		},
		{
			name:           "message create error is returned",
			event:          textEvent(&linebot.EventSource{Type: linebot.EventSourceTypeGroup, GroupID: "Gtest-group"}),
			expectDialogID: "Gtest-group",
			listRes:        []conversation.Conversation{cv},
			createErr:      fmt.Errorf("create failed"),
			expectErr:      true,
		},
		{
			name:           "room source uses the room id as dialog id",
			event:          textEvent(&linebot.EventSource{Type: linebot.EventSourceTypeRoom, RoomID: "Rtest-room"}),
			expectDialogID: "Rtest-room",
			listRes:        []conversation.Conversation{cv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &lineHandler{reqHandler: mockReq}

			ctx := context.Background()

			if tt.expectDialogID != "" {
				mockReq.EXPECT().ConversationV1ConversationList(ctx, "", uint64(1), map[conversation.Field]any{
					conversation.FieldDeleted:  false,
					conversation.FieldType:     conversation.TypeLine,
					conversation.FieldDialogID: tt.expectDialogID,
				}).Return(tt.listRes, tt.listErr)
			}
			if len(tt.listRes) > 0 {
				var created *message.Message
				if tt.createErr == nil {
					created = &message.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000002")}}
				}
				mockReq.EXPECT().ConversationV1MessageCreate(
					ctx, uuid.Nil, cv.CustomerID, cv.ID,
					message.DirectionIncoming, message.StatusDone, message.ReferenceTypeLine,
					uuid.Nil, "", "hi", []media.Media{}, cv.Peer, cv.Self,
				).Return(created, tt.createErr)
			}

			res, err := h.hookEventTypeMessage(ctx, testHookAccount(), tt.event)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return
			}
			if res == nil || res.Conversation == nil || res.Message == nil {
				t.Fatalf("Wrong result. expect conversation and message, got: %v", res)
			}
		})
	}
}

func Test_hookEventTypeFollow_noDialogID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no request is expected: the handler must stop before looking up the peer.
	h := &lineHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

	e := &linebot.Event{
		Type:   linebot.EventTypeFollow,
		Source: &linebot.EventSource{Type: linebot.EventSourceType("unknown")},
	}

	if err := h.hookEventTypeFollow(context.Background(), testHookAccount(), e); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_getDialogID(t *testing.T) {
	tests := []struct {
		name   string
		source *linebot.EventSource

		expectRes string
	}{
		{
			name:      "user",
			source:    &linebot.EventSource{Type: linebot.EventSourceTypeUser, UserID: "Utest", GroupID: "Gignored", RoomID: "Rignored"},
			expectRes: "Utest",
		},
		{
			name:      "group",
			source:    &linebot.EventSource{Type: linebot.EventSourceTypeGroup, UserID: "Uignored", GroupID: "Gtest", RoomID: "Rignored"},
			expectRes: "Gtest",
		},
		{
			name:      "room",
			source:    &linebot.EventSource{Type: linebot.EventSourceTypeRoom, UserID: "Uignored", GroupID: "Gignored", RoomID: "Rtest"},
			expectRes: "Rtest",
		},
		{
			name:      "unknown source type",
			source:    &linebot.EventSource{Type: linebot.EventSourceType("unknown"), UserID: "Uignored"},
			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &lineHandler{}

			res := h.getDialogID(&linebot.Event{Source: tt.source})
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

func Test_Hook_malformedEvents(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		expectErr bool
	}{
		{"null event", `{"events":[null]}`, false},
		{"event without sub object", `{"events":[{"type":"message"}]}`, true},
		{"event without source", `{"events":[{"type":"follow"}]}`, false},
		{"message event with unknown message type", `{"events":[{"type":"message","source":{"type":"user","userId":"Utest-user"},"message":{"type":"unknown"}}]}`, false},
	}

	for _, tt := range tests {
		t.Run("handles malformed events safely/"+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no request is expected for any malformed input
			h := &lineHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

			res, err := h.Hook(context.Background(), testHookAccount(), []byte(tt.data))
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if len(res) != 0 {
				t.Errorf("Wrong match. expect: no result, got: %v", res)
			}
		})
	}
}
