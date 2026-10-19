package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

var errTXRowsBroken = errors.New("simulated failure while reading rows")

func newTXTestHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockCache
}

func Test_CallTXStart_rowsErr(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	h, mock, _ := newTXTestHandler(t)

	// the first row fails when it is read, so Next returns false and the
	// cause is only reported by rows.Err().
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FOR UPDATE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()).RowError(0, errTXRowsBroken))
	mock.ExpectRollback()

	tx, c, err := h.CallTXStart(context.Background(), id)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
	}
	if !errors.Is(err, errTXRowsBroken) {
		t.Errorf("expected the iteration failure in the error, got: %v", err)
	}
	if tx != nil || c != nil {
		t.Errorf("expected no transaction and no call. tx: %v, call: %v", tx, c)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the transaction must be rolled back. err: %v", err)
	}
}

func Test_CallTXStart_notFound(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	h, mock, _ := newTXTestHandler(t)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	if _, _, err := h.CallTXStart(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the transaction must be rolled back. err: %v", err)
	}
}

func Test_CallTXStart_found(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	h, mock, _ := newTXTestHandler(t)

	// a row of all NULL values has the same shape as a real row
	cols := commondatabasehandler.GetDBFields(&call.Call{})
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FOR UPDATE").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(make([]driver.Value, len(cols))...))

	tx, c, err := h.CallTXStart(context.Background(), id)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if tx == nil || c == nil {
		t.Fatalf("expected a transaction and a call. tx: %v, call: %v", tx, c)
	}
	mock.ExpectRollback()
	h.CallTXFinish(tx, false)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test_CallTXStart_cancelBeforeBegin verifies that the context reaches
// BeginTx: a cancelled context never reaches the driver.
func Test_CallTXStart_cancelBeforeBegin(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	h, _, _ := newTXTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// no ExpectBegin: the call must fail before the driver is reached.
	if _, _, err := h.CallTXStart(ctx, id); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// Test_CallTXStart_cancelDuringQuery verifies that the context reaches the
// FOR UPDATE query: cancelling while the query waits aborts it instead of
// waiting for it to finish.
func Test_CallTXStart_cancelDuringQuery(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	delay := 2 * time.Second

	h, mock, _ := newTXTestHandler(t)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FOR UPDATE").WillDelayFor(delay).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	start := time.Now()
	_, _, err := h.CallTXStart(ctx, id)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("a cancelled query must not be reported as ErrNotFound. err: %v", err)
	}
	if elapsed >= delay/2 {
		t.Errorf("the query was not aborted by the cancel. elapsed: %v", elapsed)
	}
}

// Test_CallTXChainedCallID_cancelled verifies that the context reaches the
// statements of a running transaction and that nothing is written to the
// cache when the statement is aborted.
func Test_CallTXChainedCallID_cancelled(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	chainedCallID := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")

	tests := []struct {
		name string
		fn   func(h *handler, ctx context.Context, tx *sql.Tx) error
	}{
		{"add", func(h *handler, ctx context.Context, tx *sql.Tx) error {
			return h.CallTXAddChainedCallID(ctx, tx, id, chainedCallID)
		}},
		{"remove", func(h *handler, ctx context.Context, tx *sql.Tx) error {
			return h.CallTXRemoveChainedCallID(ctx, tx, id, chainedCallID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newTXTestHandler(t) // the cache mock has no expectation

			mock.ExpectBegin()
			tx, err := h.db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatalf("could not begin: %v", err)
			}
			defer func() { _ = tx.Rollback() }()

			// the expectation only detects a statement that ignores the
			// cancelled context. It is not fulfilled by a correct call.
			mock.ExpectExec("UPDATE call_calls").WillReturnResult(sqlmock.NewResult(0, 1))

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tt.fn(h, ctx, tx); !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got: %v", err)
			}
		})
	}
}

// Test_CallTXChainedCallID_cacheRefreshCancelled verifies that the cache
// refresh before the commit uses the passed context: cancelling while the
// refresh reads the call aborts it instead of waiting for it.
func Test_CallTXChainedCallID_cacheRefreshCancelled(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	chainedCallID := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")
	delay := 2 * time.Second

	tests := []struct {
		name string
		fn   func(h *handler, ctx context.Context, tx *sql.Tx) error
	}{
		{"add", func(h *handler, ctx context.Context, tx *sql.Tx) error {
			return h.CallTXAddChainedCallID(ctx, tx, id, chainedCallID)
		}},
		{"remove", func(h *handler, ctx context.Context, tx *sql.Tx) error {
			return h.CallTXRemoveChainedCallID(ctx, tx, id, chainedCallID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newTXTestHandler(t) // the cache mock has no expectation

			mock.ExpectBegin()
			tx, err := h.db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatalf("could not begin: %v", err)
			}
			defer func() { _ = tx.Rollback() }()

			mock.ExpectExec("UPDATE call_calls").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT .* FROM call_calls").WillDelayFor(delay).WillReturnRows(sqlmock.NewRows([]string{"id"}))

			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(20*time.Millisecond, cancel)

			start := time.Now()
			if err := tt.fn(h, ctx, tx); err != nil {
				t.Errorf("the cache refresh failure must not fail the statement. err: %v", err)
			}
			if elapsed := time.Since(start); elapsed >= delay/2 {
				t.Errorf("the cache refresh was not aborted by the cancel. elapsed: %v", elapsed)
			}
		})
	}
}

func Test_CallTXFinish_commitError(t *testing.T) {
	h, mock, _ := newTXTestHandler(t)

	mock.ExpectBegin()
	tx, err := h.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("could not begin: %v", err)
	}
	errCommit := errors.New("simulated commit failure")
	mock.ExpectCommit().WillReturnError(errCommit)

	hook := logrustest.NewGlobal()
	defer logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(prev)

	h.CallTXFinish(tx, true)

	entry := hook.LastEntry()
	if entry == nil {
		t.Fatalf("expected the commit failure to be logged")
	}
	if entry.Level != logrus.ErrorLevel {
		t.Errorf("expected an error level log, got: %v", entry.Level)
	}
	if got, ok := entry.Data["error"].(error); !ok || !errors.Is(got, errCommit) {
		t.Errorf("expected the commit failure in the log, got: %v", entry.Data)
	}
}

func Test_CallTXFinish_noLogOnSuccessOrRollback(t *testing.T) {
	h, mock, _ := newTXTestHandler(t)

	hook := logrustest.NewGlobal()
	defer logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))

	mock.ExpectBegin()
	tx1, _ := h.db.BeginTx(context.Background(), nil)
	mock.ExpectCommit()
	h.CallTXFinish(tx1, true)

	mock.ExpectBegin()
	tx2, _ := h.db.BeginTx(context.Background(), nil)
	mock.ExpectRollback()
	h.CallTXFinish(tx2, false)

	// a rollback of an already finished transaction is expected and silent
	h.CallTXFinish(tx2, false)

	if len(hook.AllEntries()) != 0 {
		t.Errorf("expected no log entries, got: %v", hook.AllEntries())
	}
}
