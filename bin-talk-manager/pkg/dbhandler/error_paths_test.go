package dbhandler

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

var errInjected = errors.New("injected database failure")

func Test_dbHandler_error(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")
	customerID := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e11")

	newChat := func() *chat.Chat {
		return &chat.Chat{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Type: chat.TypeGroup}
	}
	newParticipant := func() *participant.Participant {
		return &participant.Participant{
			Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: id},
			ChatID:   id,
		}
	}
	newMessage := func() *message.Message {
		return &message.Message{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, ChatID: id, Type: message.TypeNormal}
	}

	// metadata row served to the reaction methods before their write fails
	metadataRow := func(string) (driver.Rows, error) {
		return rowsOf([]string{"metadata"}, []driver.Value{`{"reactions":[]}`}), nil
	}
	reactionJSON := `{"emoji":"smile","owner_type":"agent","owner_id":"2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10"}`

	tests := []struct {
		name string
		// failExec makes every ExecContext fail, failQuery every QueryContext.
		script *script
		fn     func(h *dbHandler) error
	}{
		// writes: the failing statement is the only one the method issues
		{"ChatCreate", &script{exec: failExec}, func(h *dbHandler) error { return h.ChatCreate(ctx, newChat()) }},
		{"ChatUpdate", &script{exec: failExec}, func(h *dbHandler) error {
			return h.ChatUpdate(ctx, id, map[chat.Field]any{chat.FieldName: "n"})
		}},
		{"ChatDelete", &script{exec: failExec}, func(h *dbHandler) error { return h.ChatDelete(ctx, id) }},
		{"ChatMemberCountIncrement", &script{exec: failExec}, func(h *dbHandler) error { return h.ChatMemberCountIncrement(ctx, id) }},
		{"ChatMemberCountDecrement", &script{exec: failExec}, func(h *dbHandler) error { return h.ChatMemberCountDecrement(ctx, id) }},
		{"ParticipantCreate", &script{exec: failExec}, func(h *dbHandler) error { return h.ParticipantCreate(ctx, newParticipant()) }},
		{"ParticipantDelete", &script{exec: failExec}, func(h *dbHandler) error { return h.ParticipantDelete(ctx, id) }},
		{"MessageCreate", &script{exec: failExec}, func(h *dbHandler) error { return h.MessageCreate(ctx, newMessage()) }},
		{"MessageUpdate", &script{exec: failExec}, func(h *dbHandler) error {
			return h.MessageUpdate(ctx, id, map[message.Field]any{message.FieldText: "t"})
		}},
		{"MessageDelete", &script{exec: failExec}, func(h *dbHandler) error { return h.MessageDelete(ctx, id) }},

		// reads: the query itself fails
		{"ChatGet", &script{query: failQuery}, func(h *dbHandler) error { _, e := h.ChatGet(ctx, id); return e }},
		{"ChatList", &script{query: failQuery}, func(h *dbHandler) error {
			_, e := h.ChatList(ctx, map[chat.Field]any{}, "", 10)
			return e
		}},
		{"FindDirectChatByParticipants", &script{query: failQuery}, func(h *dbHandler) error {
			_, e := h.FindDirectChatByParticipants(ctx, customerID, "agent", id, "agent", customerID)
			return e
		}},
		{"ParticipantGet", &script{query: failQuery}, func(h *dbHandler) error { _, e := h.ParticipantGet(ctx, id); return e }},
		{"ParticipantList", &script{query: failQuery}, func(h *dbHandler) error {
			_, e := h.ParticipantList(ctx, map[participant.Field]any{})
			return e
		}},
		{"ParticipantListByChatIDs", &script{query: failQuery}, func(h *dbHandler) error {
			_, e := h.ParticipantListByChatIDs(ctx, []uuid.UUID{id})
			return e
		}},
		{"MessageGet", &script{query: failQuery}, func(h *dbHandler) error { _, e := h.MessageGet(ctx, id); return e }},
		{"MessageList", &script{query: failQuery}, func(h *dbHandler) error {
			_, e := h.MessageList(ctx, map[message.Field]any{}, "", 10)
			return e
		}},

		// reactions read the message first and then write it back
		{"MessageAddReactionAtomic read", &script{query: failQuery}, func(h *dbHandler) error {
			return h.MessageAddReactionAtomic(ctx, id, reactionJSON)
		}},
		{"MessageAddReactionAtomic write", &script{query: metadataRow, exec: failExec}, func(h *dbHandler) error {
			return h.MessageAddReactionAtomic(ctx, id, reactionJSON)
		}},
		{"MessageRemoveReactionAtomic read", &script{query: failQuery}, func(h *dbHandler) error {
			return h.MessageRemoveReactionAtomic(ctx, id, "smile", "agent", id)
		}},
		{"MessageRemoveReactionAtomic write", &script{query: metadataRow, exec: failExec}, func(h *dbHandler) error {
			return h.MessageRemoveReactionAtomic(ctx, id, "smile", "agent", id)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newScriptedHandler(t, tt.script)

			if err := tt.fn(h); !errors.Is(err, errInjected) {
				t.Errorf("error = %v, want the injected database failure", err)
			}
		})
	}
}

func failExec(int, string) (driver.Result, error) { return nil, errInjected }
func failQuery(string) (driver.Rows, error)       { return nil, errInjected }

func Test_dbHandler_corruptedMetadata_error(t *testing.T) {
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")
	validReaction := `{"emoji":"smile","owner_type":"agent","owner_id":"2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10"}`

	tests := []struct {
		name         string
		storedJSON   string
		reactionJSON string
		fn           func(h *dbHandler, reactionJSON string) error
	}{
		{
			name: "add: stored metadata is not json", storedJSON: `not json`, reactionJSON: validReaction,
			fn: func(h *dbHandler, r string) error { return h.MessageAddReactionAtomic(context.Background(), id, r) },
		},
		{
			name: "add: reaction is not json", storedJSON: `{"reactions":[]}`, reactionJSON: `not json`,
			fn: func(h *dbHandler, r string) error { return h.MessageAddReactionAtomic(context.Background(), id, r) },
		},
		{
			name: "remove: stored metadata is not json", storedJSON: `not json`,
			fn: func(h *dbHandler, _ string) error {
				return h.MessageRemoveReactionAtomic(context.Background(), id, "smile", "agent", id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{query: func(string) (driver.Rows, error) {
				return rowsOf([]string{"metadata"}, []driver.Value{tt.storedJSON}), nil
			}}
			h := newScriptedHandler(t, s)

			if err := tt.fn(h, tt.reactionJSON); err == nil {
				t.Errorf("expected an error")
			}
			if n := len(s.recordedExecs()); n != 0 {
				t.Errorf("the message was written %d times although the input was invalid", n)
			}
		})
	}

}

func Test_ParticipantCreate_duplicate(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")
	chatID := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e11")
	ownerID := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e12")
	p := func() *participant.Participant {
		return &participant.Participant{
			Identity: commonidentity.Identity{ID: id, CustomerID: id},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: ownerID},
			ChatID:   chatID,
		}
	}

	errDuplicate := errors.New("Error 1062 (23000): Duplicate entry for key 'unique_participant'")

	tests := []struct {
		name string
		// results of the successive Exec calls
		results []error

		wantErr   error
		wantExecs int
	}{
		{"duplicate then update succeeds", []error{errDuplicate, nil}, nil, 2},
		{"duplicate then update fails", []error{errDuplicate, errInjected}, errInjected, 2},
		{"other failure is not retried", []error{errInjected}, errInjected, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{exec: func(n int, _ string) (driver.Result, error) {
				if n < len(tt.results) && tt.results[n] != nil {
					return nil, tt.results[n]
				}
				return driver.RowsAffected(1), nil
			}}
			h := newScriptedHandler(t, s)

			if err := h.ParticipantCreate(ctx, p()); !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			execs := s.recordedExecs()
			if len(execs) != tt.wantExecs {
				t.Fatalf("exec calls = %d, want %d", len(execs), tt.wantExecs)
			}
			if tt.wantExecs == 2 {
				if !strings.HasPrefix(execs[0].query, "INSERT") || !strings.HasPrefix(execs[1].query, "UPDATE") {
					t.Errorf("statements = %q then %q, want an INSERT then an UPDATE", execs[0].query, execs[1].query)
				}

				// the update must be limited to the one participant: the chat, the owner type and the owner
				for _, col := range []string{"chat_id = ?", "owner_type = ?", "owner_id = ?"} {
					if !strings.Contains(execs[1].query, col) {
						t.Errorf("update %q is not limited by %s", execs[1].query, col)
					}
				}
				tail := execs[1].args[len(execs[1].args)-3:]
				if got, ok := tail[0].Value.([]byte); !ok || !bytes.Equal(got, chatID.Bytes()) {
					t.Errorf("chat id argument = %v, want %v", tail[0].Value, chatID.Bytes())
				}
				if tail[1].Value != "agent" {
					t.Errorf("owner type argument = %v, want agent", tail[1].Value)
				}
				if got, ok := tail[2].Value.([]byte); !ok || !bytes.Equal(got, ownerID.Bytes()) {
					t.Errorf("owner id argument = %v, want %v", tail[2].Value, ownerID.Bytes())
				}
			}
		})
	}
}

func Test_ChatMemberCount_statements(t *testing.T) {
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")

	tests := []struct {
		name string
		fn   func(h *dbHandler) error
		want string
	}{
		{"increment", func(h *dbHandler) error { return h.ChatMemberCountIncrement(context.Background(), id) }, "member_count + 1"},
		{"decrement", func(h *dbHandler) error { return h.ChatMemberCountDecrement(context.Background(), id) }, "GREATEST(member_count - 1, 0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{}
			h := newScriptedHandler(t, s)

			if err := tt.fn(h); err != nil {
				t.Fatalf("error = %v", err)
			}

			execs := s.recordedExecs()
			if len(execs) != 1 {
				t.Fatalf("exec calls = %d, want 1", len(execs))
			}
			if !strings.Contains(execs[0].query, tt.want) || !strings.HasSuffix(strings.TrimSpace(execs[0].query), "WHERE id = ?") {
				t.Errorf("query = %q, want it to contain %q and filter by id", execs[0].query, tt.want)
			}

			// the arguments are the update time and then the 16 byte chat id
			if len(execs[0].args) != 2 {
				t.Fatalf("args = %d, want 2", len(execs[0].args))
			}
			gotID, ok := execs[0].args[1].Value.([]byte)
			if !ok || !strings.EqualFold(uuid.FromBytesOrNil(gotID).String(), id.String()) {
				t.Errorf("second argument = %v, want the chat id bytes", execs[0].args[1].Value)
			}
		})
	}
}
