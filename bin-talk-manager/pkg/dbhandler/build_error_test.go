package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
)

var errBuild = errors.New("injected query build failure")

// failingValuer is a filter value that cannot be turned into a query argument.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) { return nil, errBuild }

// failingSqlizer is an update value that cannot be turned into SQL. Its kind is string, so the
// field conversion passes it through unchanged.
type failingSqlizer string

func (failingSqlizer) ToSql() (string, []any, error) { return "", nil, errBuild }

// newBuildHandler returns a handler without a database. Every function under test must stop at
// the query build error before it touches the database, so a missing guard shows up as a panic.
func newBuildHandler() *dbHandler {
	return &dbHandler{utilHandler: commonutil.NewUtilHandler()}
}

func Test_dbHandler_queryBuild_error(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")

	tests := []struct {
		name string
		fn   func(h *dbHandler) error
	}{
		{"ChatList", func(h *dbHandler) error {
			_, err := h.ChatList(ctx, map[chat.Field]any{chat.FieldName: commondb.NotEq{Value: failingValuer{}}}, "", 10)
			return err
		}},
		{"MessageList", func(h *dbHandler) error {
			_, err := h.MessageList(ctx, map[message.Field]any{message.FieldText: commondb.NotEq{Value: failingValuer{}}}, "", 10)
			return err
		}},
		{"ParticipantList", func(h *dbHandler) error {
			_, err := h.ParticipantList(ctx, map[participant.Field]any{participant.FieldOwnerType: commondb.NotEq{Value: failingValuer{}}})
			return err
		}},
		{"ChatUpdate", func(h *dbHandler) error {
			return h.ChatUpdate(ctx, id, map[chat.Field]any{chat.FieldName: failingSqlizer("x")})
		}},
		{"MessageUpdate", func(h *dbHandler) error {
			return h.MessageUpdate(ctx, id, map[message.Field]any{message.FieldText: failingSqlizer("x")})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(newBuildHandler()); !errors.Is(err, errBuild) {
				t.Errorf("error = %v, want the query build failure", err)
			}
		})
	}
}

// A reaction time outside of the years 0 to 9999 cannot be encoded as JSON, so the message
// must be rejected before any query is built.
func Test_MessageCreate_unencodableReaction_error(t *testing.T) {
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	m := &message.Message{
		Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "smile", TMCreate: &outOfRange}}},
	}

	err := newBuildHandler().MessageCreate(context.Background(), m)
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("error = %v, want the year range failure", err)
	}
}

// The stored metadata holds a reaction time that decodes but cannot be encoded again (the
// offset hour is out of range), so the reaction functions must fail without writing.
func Test_reactionAtomic_unencodableStoredMetadata_error(t *testing.T) {
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")
	stored := `{"reactions":[{"emoji":"smile","owner_type":"agent","owner_id":"2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10","tm_create":"2020-01-01T00:00:00+24:00"}]}`
	reaction := `{"emoji":"clap","owner_type":"agent","owner_id":"2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e11"}`

	tests := []struct {
		name string
		fn   func(h *dbHandler) error
	}{
		{"add", func(h *dbHandler) error { return h.MessageAddReactionAtomic(context.Background(), id, reaction) }},
		{"remove", func(h *dbHandler) error {
			// the stored reaction is not the one removed, so it stays in the list and must be encoded
			return h.MessageRemoveReactionAtomic(context.Background(), id, "clap", "agent", id)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{query: func(string) (driver.Rows, error) {
				return rowsOf([]string{"metadata"}, []driver.Value{stored}), nil
			}}
			h := newScriptedHandler(t, s)

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), "timezone hour outside of range") {
				t.Errorf("error = %v, want the encode failure", err)
			}
			if n := len(s.recordedExecs()); n != 0 {
				t.Errorf("the message was written %d times although the metadata could not be encoded", n)
			}
		})
	}
}
