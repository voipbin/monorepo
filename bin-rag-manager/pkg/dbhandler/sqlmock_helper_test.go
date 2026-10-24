package dbhandler

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"
)

const (
	testCurTime = "2025-01-01 00:00:00.000000"

	ragCols = "id, customer_id, name, description, tm_create, tm_update, tm_delete"
	docCols = "id, customer_id, rag_id, name, doc_type, storage_file_id, source_url, status, status_message, tm_create, tm_update, tm_delete, retry_count, tm_processing"
)

var (
	testNow = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	testID1 = uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	testID2 = uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")
	testID3 = uuid.FromStringOrNil("33333333-3333-4333-8333-333333333333")
	testID4 = uuid.FromStringOrNil("44444444-4444-4444-8444-444444444444")
)

// newTestHandler returns a handler backed by sqlmock. The query matcher compares
// the full SQL text (whitespace normalized), so every test pins the exact statement.
// The mock util handler is returned so a test can set the exact number of clock reads.
func newTestHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *utilhandler.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mockUtil := utilhandler.NewMockUtilHandler(mc)

	return &handler{utilHandler: mockUtil, db: db}, mock, mockUtil
}

// expectNow makes the clock return testNow exactly n times.
func expectNow(m *utilhandler.MockUtilHandler, n int) {
	now := testNow
	m.EXPECT().TimeNow().Return(&now).Times(n)
}

func assertMet(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. unmet sqlmock expectations: %v", err)
	}
}
