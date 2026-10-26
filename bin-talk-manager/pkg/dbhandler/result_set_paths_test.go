package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
)

var errCloseInjected = errors.New("injected close failure")

// closeFailRows wraps driver rows and fails Close, like a connection that drops while the
// result set is still open.
type closeFailRows struct {
	driver.Rows
}

func (closeFailRows) Close() error { return errCloseInjected }

// captureLogs replaces the hooks of the standard logger with one recording hook and restores
// the previous hooks when the test ends. The standard logger is global, so the tests that use
// it must not run in parallel.
func captureLogs(t *testing.T) *logrustest.Hook {
	t.Helper()

	logger := logrus.StandardLogger()
	previous := logger.ReplaceHooks(make(logrus.LevelHooks))
	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logger.ReplaceHooks(previous) })

	return hook
}

// A single row with one column cannot be scanned into any of the models. The row is not read
// to the end, so the close failure of the driver reaches the caller of rows.Close.
func Test_resultSet_scanFailure_error(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0f8c1e-9a7d-11ef-8c2b-0f6f3c1a7d21")

	oneColumn := func(string) (driver.Rows, error) {
		return closeFailRows{rowsOf([]string{"id"}, []driver.Value{[]byte("x")})}, nil
	}

	tests := []struct {
		name string
		fn   func(h *dbHandler) (any, error)
	}{
		{"ChatGet", func(h *dbHandler) (any, error) { return h.ChatGet(ctx, id) }},
		{"ChatList", func(h *dbHandler) (any, error) { return h.ChatList(ctx, map[chat.Field]any{}, "", 10) }},
		{"FindDirectChatByParticipants", func(h *dbHandler) (any, error) {
			return h.FindDirectChatByParticipants(ctx, id, "agent", id, "agent", id)
		}},
		{"MessageGet", func(h *dbHandler) (any, error) { return h.MessageGet(ctx, id) }},
		{"MessageList", func(h *dbHandler) (any, error) { return h.MessageList(ctx, map[message.Field]any{}, "", 10) }},
		{"ParticipantGet", func(h *dbHandler) (any, error) { return h.ParticipantGet(ctx, id) }},
		{"ParticipantList", func(h *dbHandler) (any, error) { return h.ParticipantList(ctx, map[participant.Field]any{}) }},
		{"ParticipantListByChatIDs", func(h *dbHandler) (any, error) { return h.ParticipantListByChatIDs(ctx, []uuid.UUID{id}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := captureLogs(t)
			h := newScriptedHandler(t, &script{query: oneColumn})

			res, err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), "scan failed") {
				t.Errorf("%s() error = %v, want a scan failure", tt.name, err)
			}
			if !isNilResult(res) {
				t.Errorf("%s() result = %v, want none", tt.name, res)
			}

			// the close failure is only logged: the scan failure stays the returned error
			var closeLogs int
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.ErrorLevel && strings.Contains(entry.Message, "Failed to close rows") &&
					strings.Contains(entry.Message, errCloseInjected.Error()) {
					closeLogs++
				}
			}
			if closeLogs != 1 {
				t.Errorf("%s() logged the close failure %d times, want 1", tt.name, closeLogs)
			}
		})
	}
}

func isNilResult(res any) bool {
	switch v := res.(type) {
	case nil:
		return true
	case *chat.Chat:
		return v == nil
	case []*chat.Chat:
		return v == nil
	case *message.Message:
		return v == nil
	case []*message.Message:
		return v == nil
	case *participant.Participant:
		return v == nil
	case []*participant.Participant:
		return v == nil
	}

	return false
}

func Test_MessageList_token(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		token       string
		expectToken bool
	}{
		{"with token", "2024-01-15 10:30:00.000000", true},
		{"without token", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sqlText string
			h := newScriptedHandler(t, &script{query: func(query string) (driver.Rows, error) {
				sqlText = query
				return rowsOf([]string{"id"}), nil
			}})

			res, err := h.MessageList(ctx, map[message.Field]any{}, tt.token, 10)
			if err != nil || len(res) != 0 {
				t.Fatalf("MessageList() = %v, %v, want no messages and no error", res, err)
			}

			if got := strings.Contains(sqlText, "tm_create <"); got != tt.expectToken {
				t.Errorf("query = %q, token condition present = %v, want %v", sqlText, got, tt.expectToken)
			}
		})
	}
}

func Test_ParticipantListByChatIDs_empty(t *testing.T) {
	var queries atomic.Int32
	h := newScriptedHandler(t, &script{query: func(string) (driver.Rows, error) {
		queries.Add(1)
		return nil, errInjected
	}})

	res, err := h.ParticipantListByChatIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ParticipantListByChatIDs() error = %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("ParticipantListByChatIDs() = %v, want an empty, non-nil list", res)
	}
	if n := queries.Load(); n != 0 {
		t.Errorf("the database was queried %d times, want 0", n)
	}
}

// A filter value of a type that cannot be turned into a condition is rejected before any
// statement is sent.
func Test_list_unsupportedFilter_error(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(h *dbHandler) error
	}{
		{"ChatList", func(h *dbHandler) error {
			_, err := h.ChatList(ctx, map[chat.Field]any{chat.FieldName: struct{}{}}, "", 10)
			return err
		}},
		{"MessageList", func(h *dbHandler) error {
			_, err := h.MessageList(ctx, map[message.Field]any{message.FieldText: struct{}{}}, "", 10)
			return err
		}},
		{"ParticipantList", func(h *dbHandler) error {
			_, err := h.ParticipantList(ctx, map[participant.Field]any{participant.FieldOwnerType: struct{}{}})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queries atomic.Int32
			h := newScriptedHandler(t, &script{query: func(string) (driver.Rows, error) {
				queries.Add(1)
				return rowsOf([]string{"id"}), nil
			}})

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), "unsupported filter type") {
				t.Errorf("%s() error = %v, want the unsupported filter type error", tt.name, err)
			}
			if n := queries.Load(); n != 0 {
				t.Errorf("%s() queried the database %d times, want 0", tt.name, n)
			}
		})
	}
}

// An update value that cannot be encoded is rejected before any statement is sent.
func Test_update_unencodableValue_error(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0f8c1e-9a7d-11ef-8c2b-0f6f3c1a7d22")
	unencodable := []any{make(chan int)}

	tests := []struct {
		name string
		fn   func(h *dbHandler) error
	}{
		{"ChatUpdate", func(h *dbHandler) error {
			return h.ChatUpdate(ctx, id, map[chat.Field]any{chat.FieldName: unencodable})
		}},
		{"MessageUpdate", func(h *dbHandler) error {
			return h.MessageUpdate(ctx, id, map[message.Field]any{message.FieldText: unencodable})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &script{}
			h := newScriptedHandler(t, s)

			if err := tt.fn(h); err == nil {
				t.Errorf("%s() error = nil, want an encoding error", tt.name)
			}
			if n := len(s.recordedExecs()); n != 0 {
				t.Errorf("%s() sent %d statements, want 0", tt.name, n)
			}
		})
	}
}

func Test_New(t *testing.T) {
	db := &sql.DB{}
	redisClient := &redis.Client{}
	utilHandler := commonutil.NewUtilHandler()

	h, ok := New(db, redisClient, utilHandler).(*dbHandler)
	if !ok {
		t.Fatalf("New() did not return a *dbHandler")
	}

	if h.db != db || h.redis != redisClient || h.utilHandler != utilHandler {
		t.Errorf("New() did not keep its collaborators: db=%v redis=%v util=%v", h.db == db, h.redis == redisClient, h.utilHandler == utilHandler)
	}
}
