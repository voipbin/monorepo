package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// fakeConn is a minimal ClickHouse connection double. It embeds the driver
// interface so only Query and PrepareBatch need an implementation; any other
// method would panic on the nil embedded value, which keeps accidental use loud.
type fakeConn struct {
	driver.Conn

	rows     driver.Rows
	queryErr error

	batch      driver.Batch
	prepareErr error

	queryCalls   int
	queryCtx     context.Context
	querySQL     string
	queryArgs    []any
	prepareCalls int
	prepareSQL   string
}

func (f *fakeConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	f.queryCalls++
	f.queryCtx = ctx
	f.querySQL = query
	f.queryArgs = args
	if f.queryErr != nil {
		// Return an untyped nil so callers never hold a typed-nil Rows.
		return nil, f.queryErr
	}
	return f.rows, nil
}

func (f *fakeConn) PrepareBatch(_ context.Context, query string, _ ...driver.PrepareBatchOption) (driver.Batch, error) {
	f.prepareCalls++
	f.prepareSQL = query
	if f.prepareErr != nil {
		return nil, f.prepareErr
	}
	return f.batch, nil
}

// fakeRows serves positional rows. Scan checks the destination count and the
// exact destination type of each column, so swapped columns fail the test.
type fakeRows struct {
	driver.Rows

	t         *testing.T
	data      [][]any
	next      int
	scanErr   error
	scanErrAt int // 1-based index of the Scan call that fails; 0 = never
	scanCalls int
	err       error // reported by Err only after Next returned false
	exhausted bool
	closed    int
}

func (r *fakeRows) Next() bool {
	if r.next >= len(r.data) {
		r.exhausted = true
		return false
	}
	r.next++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	r.t.Helper()
	r.scanCalls++
	if r.scanErrAt != 0 && r.scanCalls == r.scanErrAt {
		return r.scanErr
	}

	row := r.data[r.next-1]
	if len(dest) != len(row) {
		r.t.Fatalf("Scan destination count = %d, want %d", len(dest), len(row))
	}

	for i, d := range dest {
		switch p := d.(type) {
		case *string:
			v, ok := row[i].(string)
			if !ok {
				r.t.Fatalf("Scan column %d: value %T is not a string", i, row[i])
			}
			*p = v
		case *time.Time:
			v, ok := row[i].(time.Time)
			if !ok {
				r.t.Fatalf("Scan column %d: value %T is not a time.Time", i, row[i])
			}
			*p = v
		case *[]string:
			v, ok := row[i].([]string)
			if !ok {
				r.t.Fatalf("Scan column %d: value %T is not a []string", i, row[i])
			}
			*p = v
		default:
			r.t.Fatalf("Scan column %d: unsupported destination %T", i, d)
		}
	}
	return nil
}

func (r *fakeRows) Err() error {
	if !r.exhausted {
		return nil
	}
	return r.err
}

func (r *fakeRows) Close() error {
	r.closed++
	return nil
}

// fakeBatch records Append argument lists and Send calls.
type fakeBatch struct {
	driver.Batch

	appended    [][]any
	appendErr   error
	appendErrAt int // 1-based index of the Append call that fails; 0 = never
	sendErr     error
	sendCalls   int
}

func (b *fakeBatch) Append(v ...any) error {
	b.appended = append(b.appended, append([]any(nil), v...))
	if b.appendErrAt != 0 && len(b.appended) == b.appendErrAt {
		return b.appendErr
	}
	return nil
}

func (b *fakeBatch) Send() error {
	b.sendCalls++
	return b.sendErr
}

// normSQL collapses all whitespace so assertions do not depend on the
// indentation of the raw query strings.
func normSQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// assertWrapped checks the exact message "<phrase>: <cause>" and that the
// cause is reachable through errors.Is.
func assertWrapped(t *testing.T, err, cause error, phrase string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want %q", phrase)
	}
	if !errors.Is(err, cause) {
		t.Errorf("error %q does not wrap cause %q", err, cause)
	}
	if want := phrase + ": " + cause.Error(); err.Error() != want {
		t.Errorf("error message = %q, want %q", err.Error(), want)
	}
}
