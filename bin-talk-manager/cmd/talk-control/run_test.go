package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/chathandler"
	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
)

const (
	testCustomerID    = "5e4a0680-804e-11ec-8477-2fea5968d85b"
	testChatID        = "7fcd7990-42eb-11ed-9fa6-b4cd93af9796"
	testOwnerID       = "a1b2c3d4-0000-4000-8000-000000000001"
	testMessageID     = "a1b2c3d4-0000-4000-8000-000000000002"
	testParentID      = "a1b2c3d4-0000-4000-8000-000000000003"
	testParticipantID = "a1b2c3d4-0000-4000-8000-000000000004"
)

// handlerMocks holds the mock handlers that replace the real ones built by initHandlers.
type handlerMocks struct {
	chat        *chathandler.MockChatHandler
	message     *messagehandler.MockMessageHandler
	participant *participanthandler.MockParticipantHandler
	reaction    *reactionhandler.MockReactionHandler
}

// useMockHandlers replaces initHandlers with a function that returns mock handlers, and puts
// the original back when the test ends. The replaced variable, os.Stdout and viper are all
// global, so the tests that use it must not run in parallel.
func useMockHandlers(t *testing.T, mc *gomock.Controller) *handlerMocks {
	t.Helper()

	m := &handlerMocks{
		chat:        chathandler.NewMockChatHandler(mc),
		message:     messagehandler.NewMockMessageHandler(mc),
		participant: participanthandler.NewMockParticipantHandler(mc),
		reaction:    reactionhandler.NewMockReactionHandler(mc),
	}

	orig := initHandlers
	t.Cleanup(func() { initHandlers = orig })
	initHandlers = func() (chathandler.ChatHandler, messagehandler.MessageHandler, participanthandler.ParticipantHandler, reactionhandler.ReactionHandler, error) {
		return m.chat, m.message, m.participant, m.reaction, nil
	}

	return m
}

// executeCapture runs the command and returns what it printed together with its error.
func executeCapture(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var err error
	out := captureStdout(t, func() {
		err = execute(t, args...)
	})

	return out, err
}

// decodeObject decodes a JSON object printed by a command.
func decodeObject(t *testing.T, out string) map[string]any {
	t.Helper()

	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("output is not a JSON object: %v, output: %q", err, out)
	}

	return res
}

// decodeList decodes a JSON array printed by a command.
func decodeList(t *testing.T, out string) []map[string]any {
	t.Helper()

	var res []map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("output is not a JSON array: %v, output: %q", err, out)
	}

	return res
}

func strPtr(s string) *string { return &s }

func testChat(id string, name string) *chat.Chat {
	return &chat.Chat{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(id), CustomerID: uuid.FromStringOrNil(testCustomerID)},
		Type:     chat.TypeGroup,
		Name:     name,
	}
}

func testMessage(id string, text string) *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(id), CustomerID: uuid.FromStringOrNil(testCustomerID)},
		ChatID:   uuid.FromStringOrNil(testChatID),
		Text:     text,
	}
}

func testParticipant(id string) *participant.Participant {
	return &participant.Participant{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(id), CustomerID: uuid.FromStringOrNil(testCustomerID)},
		ChatID:   uuid.FromStringOrNil(testChatID),
	}
}

func Test_chatCommands(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(testCustomerID)
	chatID := uuid.FromStringOrNil(testChatID)
	ownerID := uuid.FromStringOrNil(testOwnerID)

	tests := []struct {
		name  string
		args  []string
		setup func(m *handlerMocks)

		wantID   string
		wantName string
		wantList []string
	}{
		{
			name: "create",
			args: []string{"chat", "create", "--customer-id", testCustomerID, "--type", "group", "--name", "support",
				"--detail", "support room", "--creator-type", "agent", "--creator-id", testOwnerID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatCreate(ctx, customerID, chat.TypeGroup, "support", "support room", "agent", ownerID,
					[]participant.ParticipantInput{}).Return(testChat(testChatID, "support"), nil)
			},
			wantID:   testChatID,
			wantName: "support",
		},
		{
			name: "get",
			args: []string{"chat", "get", "--id", testChatID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatGet(ctx, chatID).Return(testChat(testChatID, "lobby"), nil)
			},
			wantID:   testChatID,
			wantName: "lobby",
		},
		{
			name: "delete",
			args: []string{"chat", "delete", "--id", testChatID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatDelete(ctx, chatID).Return(testChat(testChatID, "gone"), nil)
			},
			wantID:   testChatID,
			wantName: "gone",
		},
		{
			name: "update with name only leaves detail nil",
			args: []string{"chat", "update", "--id", testChatID, "--name", "renamed"},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatUpdate(ctx, chatID, strPtr("renamed"), (*string)(nil)).Return(testChat(testChatID, "renamed"), nil)
			},
			wantID:   testChatID,
			wantName: "renamed",
		},
		{
			name: "update with an empty detail passes a pointer to an empty string",
			args: []string{"chat", "update", "--id", testChatID, "--detail", ""},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatUpdate(ctx, chatID, (*string)(nil), strPtr("")).Return(testChat(testChatID, "same"), nil)
			},
			wantID:   testChatID,
			wantName: "same",
		},
		{
			name: "update with both fields",
			args: []string{"chat", "update", "--id", testChatID, "--name", "n", "--detail", "d"},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatUpdate(ctx, chatID, strPtr("n"), strPtr("d")).Return(testChat(testChatID, "n"), nil)
			},
			wantID:   testChatID,
			wantName: "n",
		},
		{
			name: "list uses the default size",
			args: []string{"chat", "list", "--customer-id", testCustomerID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatList(ctx, map[chat.Field]any{chat.FieldCustomerID: customerID}, "", uint64(100)).
					Return([]*chat.Chat{testChat(testChatID, "a"), testChat(testOwnerID, "b")}, nil)
			},
			wantList: []string{testChatID, testOwnerID},
		},
		{
			name: "list with size and token",
			args: []string{"chat", "list", "--customer-id", testCustomerID, "--size", "5", "--token", "2026-01-01T00:00:00.000000Z"},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatList(ctx, map[chat.Field]any{chat.FieldCustomerID: customerID}, "2026-01-01T00:00:00.000000Z", uint64(5)).
					Return([]*chat.Chat{}, nil)
			},
			wantList: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			tt.setup(useMockHandlers(t, mc))

			out, err := executeCapture(t, tt.args...)
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			if tt.wantList != nil {
				got := decodeList(t, out)
				if len(got) != len(tt.wantList) {
					t.Fatalf("listed %d chats, want %d, output: %s", len(got), len(tt.wantList), out)
				}
				for i, id := range tt.wantList {
					if got[i]["id"] != id {
						t.Errorf("chat %d id = %v, want %s", i, got[i]["id"], id)
					}
				}
				return
			}

			got := decodeObject(t, out)
			if got["id"] != tt.wantID || got["name"] != tt.wantName {
				t.Errorf("printed id = %v, name = %v, want %s, %s", got["id"], got["name"], tt.wantID, tt.wantName)
			}
		})
	}
}

func Test_chatCommands_error(t *testing.T) {
	ctx := context.Background()
	chatID := uuid.FromStringOrNil(testChatID)
	boom := fmt.Errorf("boom")

	tests := []struct {
		name    string
		args    []string
		setup   func(m *handlerMocks)
		wantErr string
	}{
		{
			name: "create fails in the handler",
			args: []string{"chat", "create", "--customer-id", testCustomerID, "--type", "group", "--creator-type", "agent", "--creator-id", testOwnerID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatCreate(ctx, uuid.FromStringOrNil(testCustomerID), chat.TypeGroup, "", "", "agent", uuid.FromStringOrNil(testOwnerID), []participant.ParticipantInput{}).Return(nil, boom)
			},
			wantErr: "failed to create chat: boom",
		},
		{
			name:    "get without an id",
			args:    []string{"chat", "get"},
			setup:   func(m *handlerMocks) {},
			wantErr: "failed to resolve chat ID: Chat ID is required",
		},
		{
			name: "get fails in the handler",
			args: []string{"chat", "get", "--id", testChatID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatGet(ctx, chatID).Return(nil, boom)
			},
			wantErr: "failed to retrieve chat: boom",
		},
		{
			name:    "list without a customer id",
			args:    []string{"chat", "list"},
			setup:   func(m *handlerMocks) {},
			wantErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name: "list fails in the handler",
			args: []string{"chat", "list", "--customer-id", testCustomerID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatList(ctx, map[chat.Field]any{chat.FieldCustomerID: uuid.FromStringOrNil(testCustomerID)}, "", uint64(100)).Return(nil, boom)
			},
			wantErr: "failed to list chats: boom",
		},
		{
			name:    "update with an invalid id",
			args:    []string{"chat", "update", "--id", "bad"},
			setup:   func(m *handlerMocks) {},
			wantErr: "failed to resolve chat ID: invalid format for Chat ID",
		},
		{
			name: "update fails in the handler",
			args: []string{"chat", "update", "--id", testChatID, "--name", "x"},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatUpdate(ctx, chatID, strPtr("x"), (*string)(nil)).Return(nil, boom)
			},
			wantErr: "failed to update chat: boom",
		},
		{
			name:    "delete without an id",
			args:    []string{"chat", "delete"},
			setup:   func(m *handlerMocks) {},
			wantErr: "failed to resolve chat ID: Chat ID is required",
		},
		{
			name: "delete fails in the handler",
			args: []string{"chat", "delete", "--id", testChatID},
			setup: func(m *handlerMocks) {
				m.chat.EXPECT().ChatDelete(ctx, chatID).Return(nil, boom)
			},
			wantErr: "failed to delete chat: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			tt.setup(useMockHandlers(t, mc))

			out, err := executeCapture(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if out != "" {
				t.Errorf("output = %q, want nothing to be printed", out)
			}
		})
	}
}

func Test_messageCommands(t *testing.T) {
	ctx := context.Background()
	chatID := uuid.FromStringOrNil(testChatID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	messageID := uuid.FromStringOrNil(testMessageID)
	parentID := uuid.FromStringOrNil(testParentID)

	tests := []struct {
		name  string
		args  []string
		setup func(m *handlerMocks)

		wantID   string
		wantText string
		wantList []string
	}{
		{
			name: "create without a parent",
			args: []string{"message", "create", "--chat-id", testChatID, "--owner-type", "agent", "--owner-id", testOwnerID,
				"--type", "normal", "--text", "hello"},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageCreate(ctx, messagehandler.MessageCreateRequest{
					ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: "normal", Text: "hello", Medias: []message.Media{},
				}).Return(testMessage(testMessageID, "hello"), nil)
			},
			wantID:   testMessageID,
			wantText: "hello",
		},
		{
			name: "create with a parent",
			args: []string{"message", "create", "--chat-id", testChatID, "--parent-id", testParentID, "--owner-type", "agent",
				"--owner-id", testOwnerID, "--type", "normal", "--text", "reply"},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageCreate(ctx, messagehandler.MessageCreateRequest{
					ChatID: chatID, ParentID: &parentID, OwnerType: "agent", OwnerID: ownerID, Type: "normal", Text: "reply", Medias: []message.Media{},
				}).Return(testMessage(testMessageID, "reply"), nil)
			},
			wantID:   testMessageID,
			wantText: "reply",
		},
		{
			name: "get",
			args: []string{"message", "get", "--id", testMessageID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageGet(ctx, messageID).Return(testMessage(testMessageID, "found"), nil)
			},
			wantID:   testMessageID,
			wantText: "found",
		},
		{
			name: "delete",
			args: []string{"message", "delete", "--id", testMessageID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageDelete(ctx, messageID).Return(testMessage(testMessageID, "deleted"), nil)
			},
			wantID:   testMessageID,
			wantText: "deleted",
		},
		{
			name: "list uses the default size",
			args: []string{"message", "list", "--chat-id", testChatID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageList(ctx, map[message.Field]any{message.FieldChatID: chatID}, "", uint64(100)).
					Return([]*message.Message{testMessage(testMessageID, "a"), testMessage(testParentID, "b")}, nil)
			},
			wantList: []string{testMessageID, testParentID},
		},
		{
			name: "list with size and token",
			args: []string{"message", "list", "--chat-id", testChatID, "--size", "7", "--token", "tok"},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageList(ctx, map[message.Field]any{message.FieldChatID: chatID}, "tok", uint64(7)).
					Return([]*message.Message{}, nil)
			},
			wantList: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			tt.setup(useMockHandlers(t, mc))

			out, err := executeCapture(t, tt.args...)
			if err != nil {
				t.Fatalf("error = %v", err)
			}

			if tt.wantList != nil {
				got := decodeList(t, out)
				if len(got) != len(tt.wantList) {
					t.Fatalf("listed %d messages, want %d, output: %s", len(got), len(tt.wantList), out)
				}
				for i, id := range tt.wantList {
					if got[i]["id"] != id {
						t.Errorf("message %d id = %v, want %s", i, got[i]["id"], id)
					}
				}
				return
			}

			got := decodeObject(t, out)
			if got["id"] != tt.wantID || got["text"] != tt.wantText {
				t.Errorf("printed id = %v, text = %v, want %s, %s", got["id"], got["text"], tt.wantID, tt.wantText)
			}
		})
	}
}

func Test_messageCommands_error(t *testing.T) {
	ctx := context.Background()
	messageID := uuid.FromStringOrNil(testMessageID)
	boom := fmt.Errorf("boom")

	createArgs := func(drop string, extra ...string) []string {
		all := map[string]string{
			"--chat-id": testChatID, "--owner-type": "agent", "--owner-id": testOwnerID, "--type": "normal", "--text": "hello",
		}
		args := []string{"message", "create"}
		for _, k := range []string{"--chat-id", "--owner-type", "--owner-id", "--type", "--text"} {
			if k == drop {
				continue
			}
			args = append(args, k, all[k])
		}
		return append(args, extra...)
	}

	tests := []struct {
		name    string
		args    []string
		setup   func(m *handlerMocks)
		wantErr string
	}{
		{"create without a chat id", createArgs("--chat-id"), func(m *handlerMocks) {}, "failed to resolve chat ID: Chat ID is required"},
		{"create without an owner type", createArgs("--owner-type"), func(m *handlerMocks) {}, "owner type is required"},
		{"create without an owner id", createArgs("--owner-id"), func(m *handlerMocks) {}, "failed to resolve owner ID: Owner ID is required"},
		{"create without a type", createArgs("--type"), func(m *handlerMocks) {}, "message type is required"},
		{"create without a text", createArgs("--text"), func(m *handlerMocks) {}, "message text is required"},
		{"create with an invalid parent id", createArgs("", "--parent-id", "bad"), func(m *handlerMocks) {}, "failed to resolve parent ID: invalid format for Parent ID"},
		{
			name: "create fails in the handler",
			args: createArgs(""),
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageCreate(ctx, messagehandler.MessageCreateRequest{
					ChatID: uuid.FromStringOrNil(testChatID), OwnerType: "agent", OwnerID: uuid.FromStringOrNil(testOwnerID), Type: "normal", Text: "hello", Medias: []message.Media{},
				}).Return(nil, boom)
			},
			wantErr: "failed to create message: boom",
		},
		{"get without an id", []string{"message", "get"}, func(m *handlerMocks) {}, "failed to resolve message ID: Message ID is required"},
		{
			name: "get fails in the handler",
			args: []string{"message", "get", "--id", testMessageID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageGet(ctx, messageID).Return(nil, boom)
			},
			wantErr: "failed to retrieve message: boom",
		},
		{"list without a chat id", []string{"message", "list"}, func(m *handlerMocks) {}, "failed to resolve chat ID: Chat ID is required"},
		{
			name: "list fails in the handler",
			args: []string{"message", "list", "--chat-id", testChatID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageList(ctx, map[message.Field]any{message.FieldChatID: uuid.FromStringOrNil(testChatID)}, "", uint64(100)).Return(nil, boom)
			},
			wantErr: "failed to list messages: boom",
		},
		{"delete without an id", []string{"message", "delete"}, func(m *handlerMocks) {}, "failed to resolve message ID: Message ID is required"},
		{
			name: "delete fails in the handler",
			args: []string{"message", "delete", "--id", testMessageID},
			setup: func(m *handlerMocks) {
				m.message.EXPECT().MessageDelete(ctx, messageID).Return(nil, boom)
			},
			wantErr: "failed to delete message: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			tt.setup(useMockHandlers(t, mc))

			out, err := executeCapture(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if out != "" {
				t.Errorf("output = %q, want nothing to be printed", out)
			}
		})
	}
}

func Test_participantCommands(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(testCustomerID)
	chatID := uuid.FromStringOrNil(testChatID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	participantID := uuid.FromStringOrNil(testParticipantID)

	t.Run("add", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := useMockHandlers(t, mc)
		m.participant.EXPECT().ParticipantAdd(ctx, chatID, ownerID, "agent").Return(testParticipant(testParticipantID), nil)

		out, err := executeCapture(t, "participant", "add", "--chat-id", testChatID, "--owner-type", "agent", "--owner-id", testOwnerID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		got := decodeObject(t, out)
		if got["id"] != testParticipantID || got["chat_id"] != testChatID {
			t.Errorf("printed id = %v, chat_id = %v, want %s, %s", got["id"], got["chat_id"], testParticipantID, testChatID)
		}
	})

	t.Run("list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := useMockHandlers(t, mc)
		m.participant.EXPECT().ParticipantList(ctx, customerID, chatID).
			Return([]*participant.Participant{testParticipant(testParticipantID), testParticipant(testOwnerID)}, nil)

		out, err := executeCapture(t, "participant", "list", "--customer-id", testCustomerID, "--chat-id", testChatID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		got := decodeList(t, out)
		if len(got) != 2 || got[0]["id"] != testParticipantID || got[1]["id"] != testOwnerID {
			t.Errorf("printed list = %v, want the two participants in order", got)
		}
	})

	t.Run("remove prints a fixed status", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := useMockHandlers(t, mc)
		m.participant.EXPECT().ParticipantRemove(ctx, customerID, participantID).Return(nil)

		out, err := executeCapture(t, "participant", "remove", "--customer-id", testCustomerID, "--participant-id", testParticipantID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		want := "{\"status\": \"success\", \"message\": \"Participant removed\"}\n"
		if out != want {
			t.Errorf("output = %q, want %q", out, want)
		}
	})
}

func Test_participantCommands_error(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil(testCustomerID)
	chatID := uuid.FromStringOrNil(testChatID)
	boom := fmt.Errorf("boom")

	tests := []struct {
		name    string
		args    []string
		setup   func(m *handlerMocks)
		wantErr string
	}{
		{"add without a chat id", []string{"participant", "add"}, func(m *handlerMocks) {}, "failed to resolve chat ID: Chat ID is required"},
		{"add without an owner type", []string{"participant", "add", "--chat-id", testChatID}, func(m *handlerMocks) {}, "owner type is required"},
		{"add without an owner id", []string{"participant", "add", "--chat-id", testChatID, "--owner-type", "agent"}, func(m *handlerMocks) {}, "failed to resolve owner ID: Owner ID is required"},
		{
			name: "add fails in the handler",
			args: []string{"participant", "add", "--chat-id", testChatID, "--owner-type", "agent", "--owner-id", testOwnerID},
			setup: func(m *handlerMocks) {
				m.participant.EXPECT().ParticipantAdd(ctx, chatID, uuid.FromStringOrNil(testOwnerID), "agent").Return(nil, boom)
			},
			wantErr: "failed to add participant: boom",
		},
		{"list without a customer id", []string{"participant", "list"}, func(m *handlerMocks) {}, "failed to resolve customer ID: Customer ID is required"},
		{"list without a chat id", []string{"participant", "list", "--customer-id", testCustomerID}, func(m *handlerMocks) {}, "failed to resolve chat ID: Chat ID is required"},
		{
			name: "list fails in the handler",
			args: []string{"participant", "list", "--customer-id", testCustomerID, "--chat-id", testChatID},
			setup: func(m *handlerMocks) {
				m.participant.EXPECT().ParticipantList(ctx, customerID, chatID).Return(nil, boom)
			},
			wantErr: "failed to list participants: boom",
		},
		{"remove without a customer id", []string{"participant", "remove"}, func(m *handlerMocks) {}, "failed to resolve customer ID: Customer ID is required"},
		{"remove without a participant id", []string{"participant", "remove", "--customer-id", testCustomerID}, func(m *handlerMocks) {}, "failed to resolve participant ID: Participant ID is required"},
		{
			name: "remove fails in the handler",
			args: []string{"participant", "remove", "--customer-id", testCustomerID, "--participant-id", testParticipantID},
			setup: func(m *handlerMocks) {
				m.participant.EXPECT().ParticipantRemove(ctx, customerID, uuid.FromStringOrNil(testParticipantID)).Return(boom)
			},
			wantErr: "failed to remove participant: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			tt.setup(useMockHandlers(t, mc))

			out, err := executeCapture(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if out != "" {
				t.Errorf("output = %q, want nothing to be printed", out)
			}
		})
	}
}

func Test_reactionCommands(t *testing.T) {
	ctx := context.Background()
	messageID := uuid.FromStringOrNil(testMessageID)
	ownerID := uuid.FromStringOrNil(testOwnerID)

	for _, verb := range []string{"add", "remove"} {
		t.Run(verb, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := useMockHandlers(t, mc)

			res := testMessage(testMessageID, "reacted")
			if verb == "add" {
				m.reaction.EXPECT().ReactionAdd(ctx, messageID, "thumbsup", "agent", ownerID).Return(res, nil)
			} else {
				m.reaction.EXPECT().ReactionRemove(ctx, messageID, "thumbsup", "agent", ownerID).Return(res, nil)
			}

			out, err := executeCapture(t, "reaction", verb, "--message-id", testMessageID, "--emoji", "thumbsup",
				"--owner-type", "agent", "--owner-id", testOwnerID)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			got := decodeObject(t, out)
			if got["id"] != testMessageID || got["text"] != "reacted" {
				t.Errorf("printed id = %v, text = %v, want %s, reacted", got["id"], got["text"], testMessageID)
			}
		})
	}
}

func Test_reactionCommands_error(t *testing.T) {
	ctx := context.Background()
	messageID := uuid.FromStringOrNil(testMessageID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	boom := fmt.Errorf("boom")

	for _, verb := range []string{"add", "remove"} {
		full := map[string]string{
			"--message-id": testMessageID, "--emoji": "thumbsup", "--owner-type": "agent", "--owner-id": testOwnerID,
		}
		argsWithout := func(drop string) []string {
			args := []string{"reaction", verb}
			for _, k := range []string{"--message-id", "--emoji", "--owner-type", "--owner-id"} {
				if k == drop {
					continue
				}
				args = append(args, k, full[k])
			}
			return args
		}

		tests := []struct {
			name    string
			args    []string
			setup   func(m *handlerMocks)
			wantErr string
		}{
			{"without a message id", argsWithout("--message-id"), func(m *handlerMocks) {}, "failed to resolve message ID: Message ID is required"},
			{"without an emoji", argsWithout("--emoji"), func(m *handlerMocks) {}, "emoji is required"},
			{"without an owner type", argsWithout("--owner-type"), func(m *handlerMocks) {}, "owner type is required"},
			{"without an owner id", argsWithout("--owner-id"), func(m *handlerMocks) {}, "failed to resolve owner ID: Owner ID is required"},
			{
				name: "fails in the handler",
				args: argsWithout(""),
				setup: func(m *handlerMocks) {
					if verb == "add" {
						m.reaction.EXPECT().ReactionAdd(ctx, messageID, "thumbsup", "agent", ownerID).Return(nil, boom)
					} else {
						m.reaction.EXPECT().ReactionRemove(ctx, messageID, "thumbsup", "agent", ownerID).Return(nil, boom)
					}
				},
				wantErr: "boom",
			},
		}

		for _, tt := range tests {
			t.Run(verb+" "+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				tt.setup(useMockHandlers(t, mc))

				out, err := executeCapture(t, tt.args...)
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				if out != "" {
					t.Errorf("output = %q, want nothing to be printed", out)
				}
			})
		}
	}
}

func Test_chatCreate_handlerInitError(t *testing.T) {
	orig := initHandlers
	t.Cleanup(func() { initHandlers = orig })
	initHandlers = func() (chathandler.ChatHandler, messagehandler.MessageHandler, participanthandler.ParticipantHandler, reactionhandler.ReactionHandler, error) {
		return nil, nil, nil, nil, fmt.Errorf("no handlers")
	}

	err := execute(t, "chat", "create", "--customer-id", testCustomerID, "--type", "group", "--creator-type", "agent", "--creator-id", testOwnerID)
	if err == nil || !strings.Contains(err.Error(), "failed to initialize handlers: no handlers") {
		t.Errorf("error = %v, want the handler initialization failure", err)
	}
}
