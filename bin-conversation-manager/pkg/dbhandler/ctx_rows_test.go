package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// failingConnector is a minimal database/sql/driver implementation whose
// queries return rows that emit `emit` all-NULL rows and then fail with
// errRowsBroken. sql.Rows.Next then returns false and the cause is only
// reported by rows.Err(), which is the failure shape the dbhandler must not
// hide. All column values are NULL, which scans into every model field.
type failingConnector struct {
	columns []string
	emit    int
}

func (c *failingConnector) Connect(context.Context) (driver.Conn, error) {
	return &failingConn{c: c}, nil
}
func (c *failingConnector) Driver() driver.Driver { return nil }

type failingConn struct{ c *failingConnector }

func (f *failingConn) Prepare(query string) (driver.Stmt, error) {
	return &failingStmt{c: f.c}, nil
}
func (f *failingConn) Close() error              { return nil }
func (f *failingConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }

type failingStmt struct{ c *failingConnector }

func (s *failingStmt) Close() error  { return nil }
func (s *failingStmt) NumInput() int { return -1 }
func (s *failingStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("not supported")
}
func (s *failingStmt) Query([]driver.Value) (driver.Rows, error) {
	return &failingRows{columns: s.c.columns, emit: s.c.emit}, nil
}

type failingRows struct {
	columns []string
	emit    int
	n       int
}

func (r *failingRows) Columns() []string { return r.columns }
func (r *failingRows) Close() error      { return nil }
func (r *failingRows) Next(dest []driver.Value) error {
	if r.n < r.emit {
		r.n++
		for i := range dest {
			dest[i] = nil
		}
		return nil
	}
	return errRowsBroken
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	accountCols := commondatabasehandler.GetDBFields(&account.Account{})
	conversationCols := commondatabasehandler.GetDBFields(&conversation.Conversation{})
	mediaCols := commondatabasehandler.GetDBFields(&media.Media{})
	messageCols := commondatabasehandler.GetDBFields(&message.Message{})

	tests := []struct {
		name    string
		columns []string
		emit    int // rows returned before the failure
		fn      func(h *handler) error
	}{
		{"get account", accountCols, 0, func(h *handler) error { _, err := h.accountGetFromDB(ctx, id); return err }},
		{"get conversation", conversationCols, 0, func(h *handler) error { _, err := h.conversationGetFromDB(ctx, id); return err }},
		{"get conversation by self and peer", conversationCols, 0, func(h *handler) error {
			_, err := h.ConversationGetBySelfAndPeer(ctx, commonaddress.Address{}, commonaddress.Address{})
			return err
		}},
		{"get media", mediaCols, 0, func(h *handler) error { _, err := h.mediaGetFromDB(ctx, id); return err }},
		{"get message", messageCols, 0, func(h *handler) error { _, err := h.messageGetFromDB(ctx, id); return err }},
		{"list account", accountCols, 1, func(h *handler) error { _, err := h.AccountList(ctx, 10, "t", nil); return err }},
		{"list conversation", conversationCols, 1, func(h *handler) error { _, err := h.ConversationList(ctx, 10, "t", nil); return err }},
		{"list message", messageCols, 1, func(h *handler) error { _, err := h.MessageList(ctx, "t", 10, nil); return err }},
		{"list message by transaction id", messageCols, 1, func(h *handler) error {
			_, err := h.MessageGetsByTransactionID(ctx, "tx", "t", 10)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := sql.OpenDB(&failingConnector{columns: tt.columns, emit: tt.emit})
			defer func() { _ = db.Close() }()

			h := &handler{db: db}

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
		})
	}
}

// Test_ConversationUpdate_ContextCancel verifies that the request context
// reaches the database call: a cancelled context aborts the update instead of
// the statement running to completion.
func Test_ConversationUpdate_ContextCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")
	err := h.ConversationUpdate(ctx, id, map[conversation.Field]any{conversation.FieldName: "name"})
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Errorf("expected a context canceled error, got: %v", err)
	}
}
