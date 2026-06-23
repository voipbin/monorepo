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
	return []string{"id", "customer_id", "activeflow_id", "status", "result", "model", "error", "tm_create", "tm_update", "tm_delete"}
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
		Identity: identityOf("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"),
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
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "", "", time.Now(), nil, nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)

	// caller is NOT the owner -> masked not-found.
	_, err := h.AnalysisGet(context.Background(), other, id)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound (masked), got: %v", err)
	}
}

func Test_AnalysisArchiveAndReset_won(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	histID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	mockUtil.EXPECT().UUIDCreate().Return(histID)
	mockUtil.EXPECT().TimeNow().Return(&ts).Times(2) // archive row + reset tm_update

	mock.ExpectBegin()
	// archive select
	rows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts, nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)
	// archive insert
	mock.ExpectExec("INSERT INTO timeline_analysis_histories").WillReturnResult(sqlmock.NewResult(1, 1))
	// reset update (won: 1 row)
	mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	n, err := h.AnalysisArchiveAndReset(context.Background(), id)
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

func Test_AnalysisArchiveAndReset_lost_rollsback(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	histID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	mockUtil.EXPECT().UUIDCreate().Return(histID)
	mockUtil.EXPECT().TimeNow().Return(&ts).Times(2)

	mock.ExpectBegin()
	rows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts, nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(rows)
	mock.ExpectExec("INSERT INTO timeline_analysis_histories").WillReturnResult(sqlmock.NewResult(1, 1))
	// reset update affects 0 rows (another reanalyze already won)
	mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	n, err := h.AnalysisArchiveAndReset(context.Background(), id)
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
		WillReturnResult(sqlmock.NewResult(0, 0)) // soft-deleted while running -> 0 rows

	n, err := h.AnalysisUpdateResult(context.Background(), id, analysis.StatusCompleted, []byte(`{"version":1}`), "gpt", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected rowsAffected=0 (resurrection guard), got %d", n)
	}
}

func Test_AnalysisArchiveAndDelete(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	histID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	// AnalysisGet (ownership) -> select
	getRows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts, nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(getRows)

	mockUtil.EXPECT().UUIDCreate().Return(histID)
	mockUtil.EXPECT().TimeNow().Return(&ts).Times(2) // archive tm_create + delete ts

	mock.ExpectBegin()
	archRows := sqlmock.NewRows(analysisCols()).
		AddRow(id.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), "completed", nil, "gpt", "", ts, &ts, nil)
	mock.ExpectQuery("SELECT .* FROM timeline_analyses").WillReturnRows(archRows)
	mock.ExpectExec("INSERT INTO timeline_analysis_histories").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE timeline_analyses").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	res, err := h.AnalysisArchiveAndDelete(context.Background(), owner, id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.TMDelete == nil {
		t.Fatalf("expected tm_delete set on returned record")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func Test_AnalysisHistoryList(t *testing.T) {
	h, mock, mc := newTestHandler(t)
	defer mc.Finish()
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)

	owner := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	af := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	ts := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-06-23 00:00:00.000000")

	cols := []string{"id", "customer_id", "analysis_id", "activeflow_id", "status", "result", "model", "error", "reason", "tm_create"}
	rows := sqlmock.NewRows(cols).
		AddRow(uuid.Nil.Bytes(), owner.Bytes(), uuid.Nil.Bytes(), af.Bytes(), "completed", nil, "gpt", "", "reanalyze", ts)
	mock.ExpectQuery("SELECT .* FROM timeline_analysis_histories").WillReturnRows(rows)

	res, err := h.AnalysisHistoryList(context.Background(), owner, af, 10, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(res))
	}
	if res[0].Reason != "reanalyze" {
		t.Errorf("expected reason=reanalyze, got %s", res[0].Reason)
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
