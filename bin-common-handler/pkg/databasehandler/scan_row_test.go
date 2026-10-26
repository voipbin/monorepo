package databasehandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// fakeRows is a minimal in-memory driver.Rows: one row of column values.
// Column values must already be in a driver.Value-allowed shape (int64,
// float64, bool, []byte, string, time.Time, nil).
type fakeRows struct {
	cols   []string
	values []driver.Value
	done   bool
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.done {
		return sql.ErrNoRows
	}
	r.done = true
	copy(dest, r.values)
	return nil
}

type fakeStmt struct {
	cols   []string
	values []driver.Value
}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return -1 }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("fakeStmt: Exec not supported")
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &fakeRows{cols: s.cols, values: s.values}, nil
}

type fakeConn struct {
	cols   []string
	values []driver.Value
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{cols: c.cols, values: c.values}, nil
}
func (c *fakeConn) Close() error { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("fakeConn: Begin not supported")
}

// fakeConnector builds a *sql.DB that always returns one fixed row with the
// given columns and values, via sql.OpenDB (not sql.Register, which panics
// on a duplicate name under -count/-shuffle).
type fakeConnector struct {
	cols   []string
	values []driver.Value
}

func (c *fakeConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return &fakeConn{cols: c.cols, values: c.values}, nil
}
func (c *fakeConnector) Driver() driver.Driver { return fakeDriver{} }

type fakeDriver struct{}

func (fakeDriver) Open(name string) (driver.Conn, error) {
	return nil, fmt.Errorf("fakeDriver: Open not supported, use OpenDB")
}

// scanOneRow runs ScanRow against a single fixed row of cols/values and
// returns the resulting scan error (if any).
func scanOneRow(t *testing.T, cols []string, values []driver.Value, dest interface{}) error {
	t.Helper()

	db := sql.OpenDB(&fakeConnector{cols: cols, values: values})
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.Query("fake")
	if err != nil {
		t.Fatalf("fake query failed: %v", err)
	}
	t.Cleanup(func() { _ = rows.Close() })

	if !rows.Next() {
		t.Fatalf("fake query returned no rows")
	}

	return ScanRow(rows, dest)
}

func Test_ScanRow_PrimitiveTypes(t *testing.T) {
	type row struct {
		S  string  `db:"s"`
		I  int     `db:"i"`
		U  uint    `db:"u"`
		F  float64 `db:"f"`
		B  bool    `db:"b"`
		NS string  `db:"ns"`
		NI int     `db:"ni"`
		NF float64 `db:"nf"`
		NB bool    `db:"nb"`
	}

	dst := row{S: "sentinel", I: -1, U: 1, F: -1.5, B: true, NS: "sentinel", NI: -1, NF: -1.5, NB: true}
	cols := []string{"s", "i", "u", "f", "b", "ns", "ni", "nf", "nb"}
	values := []driver.Value{
		"hello", int64(42), int64(7), 3.5, true,
		nil, nil, nil, nil,
	}

	if err := scanOneRow(t, cols, values, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	expect := row{S: "hello", I: 42, U: 7, F: 3.5, B: true, NS: "sentinel", NI: -1, NF: -1.5, NB: true}
	if !reflect.DeepEqual(expect, dst) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, dst)
	}
}

func Test_ScanRow_JSONTag(t *testing.T) {
	type payload struct {
		A string `json:"a"`
	}
	type row struct {
		J  payload `db:"j,json"`
		NJ payload `db:"nj,json"`
	}

	dst := row{NJ: payload{A: "sentinel"}}
	cols := []string{"j", "nj"}
	values := []driver.Value{`{"a":"x"}`, nil}

	if err := scanOneRow(t, cols, values, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	expect := row{J: payload{A: "x"}, NJ: payload{A: "sentinel"}}
	if !reflect.DeepEqual(expect, dst) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, dst)
	}
}

func Test_ScanRow_JSON_EmptyString(t *testing.T) {
	type payload struct {
		A string `json:"a"`
	}
	type row struct {
		J payload `db:"j,json"`
	}

	dst := row{J: payload{A: "sentinel"}}
	if err := scanOneRow(t, []string{"j"}, []driver.Value{""}, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	if dst.J.A != "sentinel" {
		t.Errorf("Wrong match. expect: field untouched by an empty JSON string, got: %+v", dst.J)
	}
}

func Test_ScanRow_JSON_MalformedReturnsError(t *testing.T) {
	type payload struct {
		A string `json:"a"`
	}
	type row struct {
		J payload `db:"j,json"`
	}

	var dst row
	err := scanOneRow(t, []string{"j"}, []driver.Value{"not-json"}, &dst)
	if err == nil {
		t.Errorf("Wrong match. expect: an error for malformed JSON, got: nil")
	}
}

func Test_ScanRow_UUID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	type row struct {
		ID    uuid.UUID  `db:"id,uuid"`
		NID   uuid.UUID  `db:"nid,uuid"`
		PID   *uuid.UUID `db:"pid,uuid"`
		NPID  *uuid.UUID `db:"npid,uuid"`
		NPID2 *uuid.UUID `db:"npid2,uuid"`
	}

	sentinel := uuid.Must(uuid.NewV4())
	dst := row{NID: sentinel, PID: &sentinel, NPID: &sentinel, NPID2: nil}

	cols := []string{"id", "nid", "pid", "npid", "npid2"}
	values := []driver.Value{
		string(id.Bytes()), nil, string(id.Bytes()), nil, nil,
	}

	if err := scanOneRow(t, cols, values, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	if dst.ID != id {
		t.Errorf("Wrong match. expect ID: %v, got: %v", id, dst.ID)
	}
	if dst.NID != uuid.Nil {
		t.Errorf("Wrong match. expect NID (NULL non-pointer UUID): %v, got: %v", uuid.Nil, dst.NID)
	}
	if dst.PID == nil || *dst.PID != id {
		t.Errorf("Wrong match. expect PID: %v, got: %v", id, dst.PID)
	}
	if dst.NPID != nil {
		t.Errorf("Wrong match. expect NPID (NULL nullable UUID): nil, got: %v", dst.NPID)
	}
	if dst.NPID2 != nil {
		t.Errorf("Wrong match. expect NPID2 (already-nil nullable UUID): nil, got: %v", dst.NPID2)
	}
}

func Test_ScanRow_UUID_BadByteLengthReturnsError(t *testing.T) {
	type row struct {
		ID uuid.UUID `db:"id,uuid"`
	}
	type rowPtr struct {
		ID *uuid.UUID `db:"id,uuid"`
	}

	tests := []struct {
		name string
		n    int
	}{
		{name: "15 bytes", n: 15},
		{name: "17 bytes", n: 17},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/non-pointer", func(t *testing.T) {
			var dst row
			if err := scanOneRow(t, []string{"id"}, []driver.Value{string(make([]byte, tt.n))}, &dst); err == nil {
				t.Errorf("Wrong match. expect: an error for a %d-byte uuid, got: nil", tt.n)
			}
		})
		t.Run(tt.name+"/pointer", func(t *testing.T) {
			var dst rowPtr
			if err := scanOneRow(t, []string{"id"}, []driver.Value{string(make([]byte, tt.n))}, &dst); err == nil {
				t.Errorf("Wrong match. expect: an error for a %d-byte uuid, got: nil", tt.n)
			}
		})
	}
}

func Test_ScanRow_TimePtr(t *testing.T) {
	tests := []struct {
		name  string
		input driver.Value
	}{
		{name: "MySQL format", input: "2024-01-15 01:30:45.000000"},
		{name: "ISO with Z", input: "2024-01-15T01:30:45Z"},
		{name: "offset with space", input: "2024-01-15 10:30:45+09:00"},
		{name: "offset with T", input: "2024-01-15T10:30:45+09:00"},
		{name: "driver time.Time with FixedZone", input: time.Date(2024, 1, 15, 10, 30, 45, 0, time.FixedZone("JST", 9*3600))},
	}

	expect := time.Date(2024, 1, 15, 1, 30, 45, 0, time.UTC)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type row struct {
				T *time.Time `db:"t"`
			}
			var dst row

			if err := scanOneRow(t, []string{"t"}, []driver.Value{tt.input}, &dst); err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}
			if dst.T == nil || !reflect.DeepEqual(*dst.T, expect) {
				t.Errorf("Wrong match. expect: %v (UTC), got: %v", expect, dst.T)
			}
		})
	}
}

func Test_ScanRow_TimePtr_EmptyAndUnparseable(t *testing.T) {
	type row struct {
		T *time.Time `db:"t"`
	}

	t.Run("empty string leaves the field nil", func(t *testing.T) {
		var dst row
		if err := scanOneRow(t, []string{"t"}, []driver.Value{""}, &dst); err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
		if dst.T != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", dst.T)
		}
	})

	t.Run("unparseable string is an error", func(t *testing.T) {
		var dst row
		err := scanOneRow(t, []string{"t"}, []driver.Value{"not-a-time"}, &dst)
		if err == nil {
			t.Errorf("Wrong match. expect: an error, got: nil")
		}
	})
}

func Test_ScanRow_NonNullableTime_NullReturnsError(t *testing.T) {
	type row struct {
		T time.Time `db:"t"`
	}
	var dst row

	err := scanOneRow(t, []string{"t"}, []driver.Value{nil}, &dst)
	if err == nil {
		t.Errorf("Wrong match. expect: an error for a non-nullable time.Time field with NULL, got: nil")
	}
}

func Test_ScanRow_UnconvertibleDriverValueReturnsError(t *testing.T) {
	type row struct {
		I int `db:"i"`
	}
	var dst row

	// 1e20 cannot be represented by database/sql's NullInt64 conversion.
	err := scanOneRow(t, []string{"i"}, []driver.Value{1e20}, &dst)
	if err == nil {
		t.Errorf("Wrong match. expect: a scan error for an unconvertible value, got: nil")
	}
}

func Test_ScanRow_ColumnCountMismatch(t *testing.T) {
	type row struct {
		A string `db:"a"`
	}
	var dst row

	t.Run("more driver columns than targets", func(t *testing.T) {
		err := scanOneRow(t, []string{"a", "b"}, []driver.Value{"x", "y"}, &dst)
		if err == nil {
			t.Errorf("Wrong match. expect: a scan error, got: nil")
		}
	})
}

func Test_ScanRow_EmbeddedStruct(t *testing.T) {
	type embedded struct {
		A string `db:"a"`
	}
	type row struct {
		embedded
		B string `db:"b"`
	}
	var dst row

	if err := scanOneRow(t, []string{"a", "b"}, []driver.Value{"x", "y"}, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	if dst.A != "x" || dst.B != "y" {
		t.Errorf("Wrong match. expect: A=x B=y, got: A=%s B=%s", dst.A, dst.B)
	}
}

func Test_ScanRow_EmbeddedNilPointerStructIsSkipped(t *testing.T) {
	type embedded struct {
		A string `db:"a"`
	}
	type row struct {
		*embedded
		B string `db:"b"`
	}
	var dst row

	if err := scanOneRow(t, []string{"b"}, []driver.Value{"y"}, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	if dst.B != "y" {
		t.Errorf("Wrong match. expect: B=y, got: B=%s", dst.B)
	}
}

func Test_ScanRow_NoDBTagFields(t *testing.T) {
	type row struct {
		Untagged string
	}

	t.Run("zero columns succeeds", func(t *testing.T) {
		var dst row
		if err := scanOneRow(t, []string{}, []driver.Value{}, &dst); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})
}

func Test_ScanRow_DBDashIgnored(t *testing.T) {
	type row struct {
		Ignored string `db:"-"`
		A       string `db:"a"`
	}
	dst := row{Ignored: "sentinel"}

	if err := scanOneRow(t, []string{"a"}, []driver.Value{"x"}, &dst); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	if dst.Ignored != "sentinel" || dst.A != "x" {
		t.Errorf("Wrong match. expect: Ignored unchanged, A=x, got: Ignored=%s A=%s", dst.Ignored, dst.A)
	}
}

func Test_ScanRow_DestMustBePointerToStruct(t *testing.T) {
	type row struct {
		A string `db:"a"`
	}

	t.Run("non-pointer destination", func(t *testing.T) {
		var dst row
		db := sql.OpenDB(&fakeConnector{cols: []string{"a"}, values: []driver.Value{"x"}})
		defer func() { _ = db.Close() }()
		rows, _ := db.Query("fake")
		defer func() { _ = rows.Close() }()
		rows.Next()

		if err := ScanRow(rows, dst); err == nil {
			t.Errorf("Wrong match. expect: an error for a non-pointer dest, got: nil")
		}
	})

	t.Run("pointer to non-struct destination", func(t *testing.T) {
		var dst string
		db := sql.OpenDB(&fakeConnector{cols: []string{}, values: []driver.Value{}})
		defer func() { _ = db.Close() }()
		rows, _ := db.Query("fake")
		defer func() { _ = rows.Close() }()
		rows.Next()

		if err := ScanRow(rows, &dst); err == nil {
			t.Errorf("Wrong match. expect: an error for a pointer-to-non-struct dest, got: nil")
		}
	})
}

func Test_PrepareFieldsFromMap_ErrorPaths(t *testing.T) {
	t.Run("non-map input is rejected", func(t *testing.T) {
		_, err := PrepareFields(42)
		if err == nil {
			t.Errorf("Wrong match. expect: an error for a non-struct, non-map input, got: nil")
		}
	})

	t.Run("non-string-keyed map is rejected", func(t *testing.T) {
		_, err := prepareFieldsFromMap(map[int]any{1: "x"})
		if err == nil {
			t.Errorf("Wrong match. expect: an error for a non-string-keyed map, got: nil")
		}
	})

	t.Run("string-keyed custom type map is accepted", func(t *testing.T) {
		type field string
		res, err := prepareFieldsFromMap(map[field]any{"a": "x"})
		if err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}
		if res["a"] != "x" {
			t.Errorf("Wrong match. expect: a=x, got: %v", res)
		}
	})

	t.Run("a value that fails JSON marshal is wrapped with the field name", func(t *testing.T) {
		_, err := prepareFieldsFromMap(map[string]any{"bad": struct{ C chan int }{}})
		if err == nil {
			t.Fatalf("Wrong match. expect: an error, got: nil")
		}
		if !strings.HasPrefix(err.Error(), "field bad: ") {
			t.Errorf("Wrong match. expect: the error to start with \"field bad: \", got: %v", err)
		}
	})
}

func Test_ConvertValueForDB_JSONTagChanError(t *testing.T) {
	_, err := convertValueForDB(struct{ C chan int }{}, "json")
	if err == nil {
		t.Errorf("Wrong match. expect: a JSON marshal error for a chan field, got: nil")
	}
}

func Test_ConvertValueForDB_UUIDWrongType(t *testing.T) {
	_, err := convertValueForDB("not-a-uuid", "uuid")
	if err == nil {
		t.Errorf("Wrong match. expect: an error for a non-uuid value with the uuid conversion type, got: nil")
	}
}
