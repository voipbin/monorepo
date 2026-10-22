package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

var errIterate = errors.New("injected iteration error")

// failingRowsDriver is a database/sql driver whose queries return a result set that fails
// on the first Next, like a connection lost while the rows are read.
type failingRowsDriver struct{}

func (failingRowsDriver) Open(string) (driver.Conn, error) { return failingRowsConn{}, nil }

type failingRowsConn struct{}

func (failingRowsConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (failingRowsConn) Close() error                        { return nil }
func (failingRowsConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }

func (failingRowsConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return failingRows{}, nil
}

type failingRows struct{}

func (failingRows) Columns() []string         { return []string{"id"} }
func (failingRows) Close() error              { return nil }
func (failingRows) Next([]driver.Value) error { return errIterate }

func Test_rowsErrIsReturned(t *testing.T) {
	sql.Register("talk_failing_rows", failingRowsDriver{})
	db, err := sql.Open("talk_failing_rows", "")
	if err != nil {
		t.Fatalf("could not open the db: %v", err)
	}
	defer func() { _ = db.Close() }()

	h := &dbHandler{db: db}
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b4a3d6e-8c3a-11ef-9f2f-6f1c1d9c3e10")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ChatGet", func() error { _, e := h.ChatGet(ctx, id); return e }},
		{"ChatList", func() error { _, e := h.ChatList(ctx, map[chat.Field]any{}, "", 10); return e }},
		{"FindDirectChatByParticipants", func() error {
			_, e := h.FindDirectChatByParticipants(ctx, id, "agent", id, "agent", id)
			return e
		}},
		{"MessageGet", func() error { _, e := h.MessageGet(ctx, id); return e }},
		{"MessageList", func() error { _, e := h.MessageList(ctx, map[message.Field]any{}, "", 10); return e }},
		{"ParticipantGet", func() error { _, e := h.ParticipantGet(ctx, id); return e }},
		{"ParticipantList", func() error { _, e := h.ParticipantList(ctx, map[participant.Field]any{}); return e }},
		{"ParticipantListByChatIDs", func() error { _, e := h.ParticipantListByChatIDs(ctx, []uuid.UUID{id}); return e }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); !errors.Is(got, errIterate) {
				t.Errorf("%s() error = %v, want the iteration error", tt.name, got)
			}
		})
	}
}
