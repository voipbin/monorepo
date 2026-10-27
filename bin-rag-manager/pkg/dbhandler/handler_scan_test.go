package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_NewHandler_usesTheGivenDB(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("could not create sqlmock. err: %v", err)
	}
	defer func() { _ = db.Close() }()

	h := NewHandler(db)

	mock.ExpectExec("DELETE FROM rag_chunks WHERE rag_id = $1").
		WithArgs(testID1).
		WillReturnResult(sqlmock.NewResult(0, 3))

	if errDelete := h.ChunkDeleteByRagID(context.Background(), testID1); errDelete != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errDelete)
	}
	assertMet(t, mock)
}

func Test_handler_DocumentGetsByRagID_scanError(t *testing.T) {
	const sqlStr = "SELECT " + docCols + " FROM rag_documents WHERE rag_id = $1 AND tm_delete IS NULL"

	h, mock, _ := newTestHandler(t)
	// the id column is not a UUID, so the row cannot be scanned
	mock.ExpectQuery(sqlStr).WithArgs(testID3).WillReturnRows(
		docRows().AddRow("not-a-uuid", testID2.String(), testID3.String(), "a", "url", nil, "u1", "ready", "", testNow, testNow, nil, int64(0), nil),
	)

	got, err := h.DocumentGetsByRagID(context.Background(), testID3)
	if err == nil || got != nil {
		t.Fatalf("Wrong match. expect an error and nil, got: %v, %v", got, err)
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Wrong match. a scan failure must not look like a missing row, got: %v", err)
	}
	if !strings.Contains(err.Error(), `column index 0, name "id"`) {
		t.Errorf("Wrong match. expect the scan error of the id column, got: %v", err)
	}
	assertMet(t, mock)
}
