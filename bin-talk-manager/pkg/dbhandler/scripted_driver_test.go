package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
)

// scriptedDriver is a database/sql driver whose Exec and Query behaviour is supplied by the
// test. It records every statement so tests can assert on the SQL text and the arguments,
// and it can fail a chosen call, which a real sqlite database cannot do on demand.
type scriptedDriver struct{}

var (
	registerScriptedOnce sync.Once
	scriptedMu           sync.Mutex
	scriptedByName       = map[string]*script{}
)

type execCall struct {
	query string
	args  []driver.NamedValue
}

type script struct {
	// exec is called for every ExecContext. n is the zero based index of the call.
	exec func(n int, query string) (driver.Result, error)
	// query is called for every QueryContext.
	query func(query string) (driver.Rows, error)

	mu    sync.Mutex
	execs []execCall
}

func (s *script) recordedExecs() []execCall {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]execCall(nil), s.execs...)
}

func (scriptedDriver) Open(name string) (driver.Conn, error) {
	scriptedMu.Lock()
	defer scriptedMu.Unlock()

	s, ok := scriptedByName[name]
	if !ok {
		return nil, errors.New("unknown scripted database")
	}

	return &scriptedConn{s: s}, nil
}

type scriptedConn struct {
	s *script
}

func (*scriptedConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (*scriptedConn) Close() error                        { return nil }
func (*scriptedConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }

func (c *scriptedConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.s.mu.Lock()
	n := len(c.s.execs)
	c.s.execs = append(c.s.execs, execCall{query: query, args: args})
	c.s.mu.Unlock()

	if c.s.exec == nil {
		return driver.RowsAffected(1), nil
	}

	return c.s.exec(n, query)
}

func (c *scriptedConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.s.query == nil {
		return nil, errors.New("no query script")
	}

	return c.s.query(query)
}

// rowsOf returns driver rows with the given columns and values.
func rowsOf(cols []string, values ...[]driver.Value) driver.Rows {
	return &valueRows{cols: cols, values: values}
}

type valueRows struct {
	cols   []string
	values [][]driver.Value
	next   int
}

func (r *valueRows) Columns() []string { return r.cols }
func (r *valueRows) Close() error      { return nil }
func (r *valueRows) Next(dest []driver.Value) error {
	if r.next >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.next])
	r.next++

	return nil
}

// newScriptedHandler returns a dbHandler on top of the scripted driver. The database name is
// unique per call, so scripts of different tests never mix and -count=N is safe.
func newScriptedHandler(t *testing.T, s *script) *dbHandler {
	t.Helper()

	registerScriptedOnce.Do(func() { sql.Register("talk_scripted", scriptedDriver{}) })

	scriptedMu.Lock()
	name := t.Name()
	scriptedByName[name] = s
	scriptedMu.Unlock()

	db, err := sql.Open("talk_scripted", name)
	if err != nil {
		t.Fatalf("could not open the scripted db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		scriptedMu.Lock()
		delete(scriptedByName, name)
		scriptedMu.Unlock()
	})

	return &dbHandler{db: db, utilHandler: commonutil.NewUtilHandler()}
}
