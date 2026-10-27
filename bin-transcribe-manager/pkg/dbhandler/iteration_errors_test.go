package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

var (
	errTestIteration = errors.New("test row iteration failure")
	errTestValue     = errors.New("test filter value failure")
)

// failingRowsConnector is a database/sql connector whose queries succeed but whose rows fail on
// the first step, like a connection that breaks while the result is being read. It is used with
// sql.OpenDB, so nothing is registered globally.
type failingRowsConnector struct{}

func (failingRowsConnector) Connect(context.Context) (driver.Conn, error) {
	return failingRowsConn{}, nil
}
func (failingRowsConnector) Driver() driver.Driver { return failingRowsDriver{} }

type failingRowsDriver struct{}

func (failingRowsDriver) Open(string) (driver.Conn, error) { return failingRowsConn{}, nil }

type failingRowsConn struct{}

func (failingRowsConn) Prepare(string) (driver.Stmt, error) { return failingRowsStmt{}, nil }
func (failingRowsConn) Close() error                        { return nil }
func (failingRowsConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }

type failingRowsStmt struct{}

func (failingRowsStmt) Close() error  { return nil }
func (failingRowsStmt) NumInput() int { return -1 }
func (failingRowsStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("not supported")
}
func (failingRowsStmt) Query([]driver.Value) (driver.Rows, error) { return failingRows{}, nil }

type failingRows struct{}

func (failingRows) Columns() []string         { return []string{"id"} }
func (failingRows) Close() error              { return nil }
func (failingRows) Next([]driver.Value) error { return errTestIteration }

// failingValuer is a filter value that cannot be converted for the query.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) { return nil, errTestValue }

func newFailingRowsHandler() *handler {
	return &handler{db: sql.OpenDB(failingRowsConnector{})}
}

func Test_getFromDB_rowIterationError(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000201")

	tests := []struct {
		name string

		call func(t *testing.T, h *handler) error

		expectErr string
	}{
		{
			name: "transcribe",
			call: func(t *testing.T, h *handler) error {
				_, err := h.transcribeGetFromDB(context.Background(), id)
				return err
			},
			expectErr: "row iteration error. transcribeGetFromDB",
		},
		{
			name: "transcript",
			call: func(t *testing.T, h *handler) error {
				_, err := h.transcriptGetFromDB(context.Background(), id)
				return err
			},
			expectErr: "row iteration error. transcriptGetFromDB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFailingRowsHandler()

			err := tt.call(t, h)
			if !errors.Is(err, errTestIteration) || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %s wrapping the iteration error, got: %v", tt.expectErr, err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: an iteration failure is not reported as not found, got: %v", err)
			}
		})
	}
}

func Test_list_rowIterationError(t *testing.T) {
	tests := []struct {
		name string

		call func(t *testing.T, h *handler) error

		expectErr string
	}{
		{
			name: "transcribe",
			call: func(t *testing.T, h *handler) error {
				res, err := h.TranscribeList(context.Background(), 10, "2024-01-01 00:00:00.000000", nil)
				if res != nil {
					t.Errorf("Wrong match. expect: no result on error, got: %v", res)
				}
				return err
			},
			expectErr: "rows iteration error. TranscribeGets",
		},
		{
			name: "transcript",
			call: func(t *testing.T, h *handler) error {
				res, err := h.TranscriptList(context.Background(), 10, "2024-01-01 00:00:00.000000", nil)
				if res != nil {
					t.Errorf("Wrong match. expect: no result on error, got: %v", res)
				}
				return err
			},
			expectErr: "rows iteration error. TranscriptGets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFailingRowsHandler()

			err := tt.call(t, h)
			if !errors.Is(err, errTestIteration) || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %s wrapping the iteration error, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_list_filterValueError(t *testing.T) {
	tests := []struct {
		name string

		call func(t *testing.T, h *handler) error

		expectErr string
	}{
		{
			name: "transcribe",
			call: func(t *testing.T, h *handler) error {
				_, err := h.TranscribeList(context.Background(), 10, "2024-01-01 00:00:00.000000", map[transcribe.Field]any{
					transcribe.FieldStatus: commondatabasehandler.NotEq{Value: failingValuer{}},
				})
				return err
			},
			expectErr: "could not build query. TranscribeGets",
		},
		{
			name: "transcript",
			call: func(t *testing.T, h *handler) error {
				_, err := h.TranscriptList(context.Background(), 10, "2024-01-01 00:00:00.000000", map[transcript.Field]any{
					transcript.FieldDirection: commondatabasehandler.NotEq{Value: failingValuer{}},
				})
				return err
			},
			expectErr: "could not build query. TranscriptGets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFailingRowsHandler()

			err := tt.call(t, h)
			if !errors.Is(err, errTestValue) || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %s wrapping the value error, got: %v", tt.expectErr, err)
			}
		})
	}
}
