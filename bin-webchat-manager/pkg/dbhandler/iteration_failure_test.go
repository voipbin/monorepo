package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
)

// errRowsConnector is a database/sql connector whose queries succeed but whose
// rows fail on the first iteration, which a real in-memory database can not do.
type errRowsConnector struct {
	err error
}

func (c *errRowsConnector) Connect(context.Context) (driver.Conn, error) {
	return &errRowsConn{err: c.err}, nil
}

func (c *errRowsConnector) Driver() driver.Driver {
	return errRowsDriver{}
}

type errRowsDriver struct{}

func (errRowsDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("not supported")
}

type errRowsConn struct {
	err error
}

func (c *errRowsConn) Prepare(string) (driver.Stmt, error) {
	return &errRowsStmt{err: c.err}, nil
}

func (c *errRowsConn) Close() error {
	return nil
}

func (c *errRowsConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not supported")
}

type errRowsStmt struct {
	err error
}

func (s *errRowsStmt) Close() error {
	return nil
}

func (s *errRowsStmt) NumInput() int {
	return -1
}

func (s *errRowsStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("not supported")
}

func (s *errRowsStmt) Query([]driver.Value) (driver.Rows, error) {
	return &errRows{err: s.err}, nil
}

type errRows struct {
	err error
}

func (r *errRows) Columns() []string {
	return []string{"id"}
}

func (r *errRows) Close() error {
	return nil
}

func (r *errRows) Next([]driver.Value) error {
	return r.err
}

func newErrRowsDB(t *testing.T, err error) *sql.DB {
	t.Helper()

	db := sql.OpenDB(&errRowsConnector{err: err})
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func Test_rowIterationFailure_isReported(t *testing.T) {
	errIteration := errors.New("synthetic iteration failure")

	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("get", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, c := newErrTestHandler(mc, newErrRowsDB(t, errIteration))
				// the strict cache mock has no set expectation: a failed read must not be cached
				e.expectGetMiss(c)

				err := e.get(h, ctx, testEntityID)
				assertErrContains(t, err, "row iteration error")
				if !errors.Is(err, errIteration) {
					t.Errorf("Wrong error. expect: wrapping the driver error, got: %v", err)
				}
			})

			t.Run("list", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, _ := newErrTestHandler(mc, newErrRowsDB(t, errIteration))

				err := e.list(h, ctx)
				assertErrContains(t, err, "rows iteration error")
				if !errors.Is(err, errIteration) {
					t.Errorf("Wrong error. expect: wrapping the driver error, got: %v", err)
				}
			})
		})
	}
}

// A filter value of an unsupported type is rejected before any statement runs:
// the database is closed, so reaching it would give a different error.
func Test_list_unsupportedFilter(t *testing.T) {
	unsupported := []string{"a"}
	ctx := context.Background()

	tests := []struct {
		name string
		list func(h *handler) error
	}{
		{
			name: "widget",
			list: func(h *handler) error {
				_, err := h.WidgetList(ctx, 10, testListToken, map[widget.Field]any{widget.FieldName: unsupported})
				return err
			},
		},
		{
			name: "session",
			list: func(h *handler) error {
				_, err := h.SessionList(ctx, 10, testListToken, map[session.Field]any{session.FieldStatus: unsupported})
				return err
			},
		},
		{
			name: "message",
			list: func(h *handler) error {
				_, err := h.MessageList(ctx, 10, testListToken, map[message.Field]any{message.FieldText: unsupported})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _ := newErrTestHandler(mc, newClosedDB(t))

			assertErrContains(t, tt.list(h), "could not apply filters")
		})
	}
}

// An update value that can not be encoded is rejected while the fields are
// prepared. An empty slice encodes fine, so the slice holds one channel.
func Test_update_unencodableValue(t *testing.T) {
	unencodable := []any{make(chan int)}
	ctx := context.Background()

	t.Run("widget", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newErrTestHandler(mc, newClosedDB(t))

		err := h.WidgetUpdate(ctx, testEntityID, map[widget.Field]any{widget.FieldName: unencodable})
		assertErrContains(t, err, "prepare fields failed")
	})

	t.Run("session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newErrTestHandler(mc, newClosedDB(t))

		err := h.SessionUpdate(ctx, testEntityID, map[session.Field]any{session.FieldStatus: unencodable})
		assertErrContains(t, err, "prepare fields failed")
	})
}
