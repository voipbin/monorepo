package analysisdbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/analysis"
)

func newTestHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *gomock.Controller) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{
		utilHandler: mockUtil,
		db:          db,
	}
	return h, mock, mc
}

func analysisCols() []string {
	// must mirror commondatabasehandler.GetDBFields(analysis.Analysis{}) order.
	return []string{"id", "customer_id", "activeflow_id", "status", "result", "model", "error", "tm_create", "tm_update"}
}

func identityOf(id, customerID string) commonidentity.Identity {
	return commonidentity.Identity{
		ID:         uuid.FromStringOrNil(id),
		CustomerID: uuid.FromStringOrNil(customerID),
	}
}

func Test_AnalysisCreate(t *testing.T) {
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

	mock.ExpectExec("INSERT INTO timeline_analyses").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := h.AnalysisCreate(context.Background(), a); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisCreate_duplicate(t *testing.T) {
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

	mock.ExpectExec("INSERT INTO timeline_analyses").
		WillReturnError(&mysql.MySQLError{Number: mysqlErrDupEntry, Message: "Duplicate entry"})

	err := h.AnalysisCreate(context.Background(), a)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got: %v", err)
	}
}

func Test_AnalysisGet_ownership_mask(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")

	rows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "", "", time.Now(), nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)

	// caller is NOT the owner -> masked not-found.
	_, err := h.AnalysisGet(context.Background(), other, id)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound (masked), got: %v", err)
	}
}

func Test_AnalysisReset_won(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&ts)

	// in-place reset CAS (won: 1 row). No transaction, no archive.
	mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 1))

	n, err := h.AnalysisReset(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected rowsAffected=1, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisReset_lost(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&ts)

	// reset update affects 0 rows (already progressing / another reanalyze won).
	mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 0))

	n, err := h.AnalysisReset(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected rowsAffected=0 (lost race), got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisUpdateResult_guard(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	mock.ExpectExec("UPDATE timeline_analyses").
		WillReturnResult(sqlmock.NewResult(0, 0)) // hard-deleted while running -> 0 rows

	n, err := h.AnalysisUpdateResult(context.Background(), id, analysis.StatusCompleted, []byte(`{"version":1}`), "gpt", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected rowsAffected=0 (resurrection guard), got %d", n)
	}
}

func Test_AnalysisDelete(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	// AnalysisGet (ownership) -> select.
	getRows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(getRows)

	// hard delete (1 row).
	mock.ExpectExec("DELETE FROM timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 1))

	res, err := h.AnalysisDelete(context.Background(), owner, id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.ID != id {
		t.Fatalf("expected deleted record id %s, got %s", id, res.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisDelete_lost_race(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	getRows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(getRows)

	// concurrent delete already removed it -> 0 rows -> masked not-found.
	mock.ExpectExec("DELETE FROM timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := h.AnalysisDelete(context.Background(), owner, id)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound (lost delete race), got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisCountProgressing(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()

	cust := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(3))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM timeline_analyses").WillReturnRows(rows)

	n, err := h.AnalysisCountProgressing(context.Background(), cust)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected count=3, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
