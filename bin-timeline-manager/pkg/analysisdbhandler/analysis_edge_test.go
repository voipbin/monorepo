package analysisdbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/analysis"
)

func Test_AnalysisCreate_with_result(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	a := &analysis.Analysis{
		Identity:     identityOf("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"),
		ActiveflowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		Status:       analysis.StatusCompleted,
		Result:       []byte(`{"version":1}`),
		Model:        "test-model",
	}

	// the payload must be bound as a valid string (not NULL) and in column order.
	mock.ExpectExec("INSERT INTO timeline_analyses").
		WithArgs(
			a.ID.Bytes(), a.CustomerID.Bytes(), a.ActiveflowID.Bytes(),
			"completed", `{"version":1}`, "test-model", "", ts,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := h.AnalysisCreate(context.Background(), a); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. expect: expectations met, got: %v", err)
	}
	// a new row is stamped with the creation time and has no update time.
	if a.TMCreate == nil || !a.TMCreate.Equal(ts) || a.TMUpdate != nil {
		t.Errorf("Wrong match. expect: tm_create %v and nil tm_update, got: %v and %v", ts, a.TMCreate, a.TMUpdate)
	}
}

func Test_AnalysisCreate_without_result(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	a := &analysis.Analysis{
		Identity:     identityOf("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"),
		ActiveflowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		Status:       analysis.StatusProgressing,
	}

	// an empty result must be stored as NULL, not as an empty string.
	mock.ExpectExec("INSERT INTO timeline_analyses").
		WithArgs(
			a.ID.Bytes(), a.CustomerID.Bytes(), a.ActiveflowID.Bytes(),
			"progressing", nil, "", "", ts,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := h.AnalysisCreate(context.Background(), a); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. expect: expectations met, got: %v", err)
	}
}

func Test_AnalysisCreate_exec_error(t *testing.T) {
	execErr := errors.New("connection reset")

	tests := []struct {
		name string
		err  error
	}{
		{"plain error", execErr},
		// a MySQL error that is not the duplicate-key number must not be
		// reported as a duplicate.
		{"other mysql error number", &mysql.MySQLError{Number: 1213, Message: "deadlock"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mc := newTestHandler(t)
			defer mc.Finish()
			mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

			ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
			mockUtil.EXPECT().TimeNow().Return(&ts)
			mock.ExpectExec("INSERT INTO timeline_analyses").WillReturnError(tt.err)

			a := &analysis.Analysis{Identity: identityOf("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")}
			err := h.AnalysisCreate(context.Background(), a)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if errors.Is(err, ErrDuplicate) {
				t.Errorf("Wrong match. expect: not ErrDuplicate, got: %v", err)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", tt.err, err)
			}
		})
	}
}

func Test_AnalysisGet_not_found_and_scan_error(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	t.Run("no row", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(sqlmock.NewRows(analysisCols()))

		_, err := h.AnalysisGet(context.Background(), owner, id)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
		}
	})

	t.Run("column count mismatch", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		rows := sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes())
		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)

		_, err := h.AnalysisGet(context.Background(), owner, id)
		if err == nil {
			t.Fatalf("Wrong match. expect: scan error, got: nil")
		}
		if errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: not ErrNotFound, got: %v", err)
		}
	})
}

func Test_AnalysisGetByActiveflowID_outcomes(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	af := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	t.Run("found", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		rows := sqlmock.NewRows(analysisCols()).
			AddRow(id.Bytes(), owner.Bytes(), af.Bytes(), "completed", nil, "test-model", "", ts, nil)
		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)

		res, err := h.AnalysisGetByActiveflowID(context.Background(), owner, af)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.ID != id || res.CustomerID != owner || res.ActiveflowID != af || res.Status != analysis.StatusCompleted || res.Model != "test-model" {
			t.Errorf("Wrong match. expect: row for %s, got: %+v", id, res)
		}
	})

	t.Run("no row", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(sqlmock.NewRows(analysisCols()))

		_, err := h.AnalysisGetByActiveflowID(context.Background(), owner, af)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
		}
	})

	t.Run("column count mismatch", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()))

		_, err := h.AnalysisGetByActiveflowID(context.Background(), owner, af)
		if err == nil {
			t.Fatalf("Wrong match. expect: scan error, got: nil")
		}
		if errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: not ErrNotFound, got: %v", err)
		}
	})
}

func Test_AnalysisList_outcomes(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	af := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	t.Run("rows returned with explicit token", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		rows := sqlmock.NewRows(analysisCols()).
			AddRow(id.Bytes(), owner.Bytes(), af.Bytes(), "completed", nil, "", "", ts, nil)
		mock.ExpectQuery("SELECT .* FROM timeline_analyses").
			WithArgs("2026-06-24T00:00:00.000000Z", owner.Bytes(), "completed").
			WillReturnRows(rows)

		filters := map[analysis.Field]any{analysis.FieldStatus: "completed"}
		res, err := h.AnalysisList(context.Background(), owner, 10, "2026-06-24T00:00:00.000000Z", filters)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) != 1 || res[0].ID != id || res[0].ActiveflowID != af {
			t.Errorf("Wrong match. expect: 1 row for %s, got: %+v", id, res)
		}
	})

	t.Run("column count mismatch", func(t *testing.T) {
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()))

		res, err := h.AnalysisList(context.Background(), owner, 10, "2026-06-24T00:00:00.000000Z", nil)
		if err == nil {
			t.Fatalf("Wrong match. expect: scan error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %d rows", len(res))
		}
	})

	t.Run("unsupported filter value", func(t *testing.T) {
		// no expectation on the mock: the query must not be executed.
		h, mock, mc := newTestHandler(t)
		defer mc.Finish()

		filters := map[analysis.Field]any{analysis.FieldStatus: []string{"completed"}}
		res, err := h.AnalysisList(context.Background(), owner, 10, "2026-06-24T00:00:00.000000Z", filters)
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %d rows", len(res))
		}
		if !strings.Contains(err.Error(), "could not apply filters") {
			t.Errorf("Wrong match. expect: filter error, got: %v", err)
		}
		if errMock := mock.ExpectationsWereMet(); errMock != nil {
			t.Errorf("Wrong match. expect: no query executed, got: %v", errMock)
		}
	})
}

func Test_AnalysisReset_errors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
	}{
		{"exec error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("UPDATE timeline_analyses").WillReturnError(injected)
		}},
		{"rows affected error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewErrorResult(injected))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mc := newTestHandler(t)
			defer mc.Finish()
			mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)
			mockUtil.EXPECT().TimeNow().Return(&ts)
			tt.expect(mock)

			n, err := h.AnalysisReset(context.Background(), id)
			if !errors.Is(err, injected) {
				t.Errorf("Wrong match. expect: %v, got: %v", injected, err)
			}
			if n != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", n)
			}
		})
	}
}

func Test_AnalysisUpdateResult_errors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
	}{
		{"exec error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("UPDATE timeline_analyses").WillReturnError(injected)
		}},
		{"rows affected error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewErrorResult(injected))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mc := newTestHandler(t)
			defer mc.Finish()
			mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)
			mockUtil.EXPECT().TimeNow().Return(&ts)
			tt.expect(mock)

			n, err := h.AnalysisUpdateResult(context.Background(), id, analysis.StatusFailed, nil, "", "test failure")
			if !errors.Is(err, injected) {
				t.Errorf("Wrong match. expect: %v, got: %v", injected, err)
			}
			if n != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", n)
			}
		})
	}
}

func Test_AnalysisDelete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
	}{
		{"exec error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("DELETE FROM timeline_analyses").WillReturnError(injected)
		}},
		{"rows affected error", func(mock sqlmock.Sqlmock) {
			mock.ExpectExec("DELETE FROM timeline_analyses").WillReturnResult(sqlmock.NewErrorResult(injected))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mc := newTestHandler(t)
			defer mc.Finish()

			getRows := sqlmock.NewRows(analysisCols()).
				AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "", "", ts, nil)
			mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(getRows)
			tt.expect(mock)

			res, err := h.AnalysisDelete(context.Background(), owner, id)
			if !errors.Is(err, injected) {
				t.Errorf("Wrong match. expect: %v, got: %v", injected, err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: not ErrNotFound, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %+v", res)
			}
		})
	}
}

func Test_AnalysisCountProgressing_scan_error(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	// a non-numeric value cannot be scanned into the int64 count.
	rows := sqlmock.NewRows([]string{"count"}).AddRow("not-a-number")
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM timeline_analyses").WillReturnRows(rows)

	n, err := h.AnalysisCountProgressing(context.Background(), cust)
	if err == nil {
		t.Fatalf("Wrong match. expect: scan error, got: nil")
	}
	if n != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", n)
	}
}
