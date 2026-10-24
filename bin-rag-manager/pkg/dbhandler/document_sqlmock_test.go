package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
)

func docRows() *sqlmock.Rows {
	return sqlmock.NewRows(strings.Split(docCols, ", "))
}

func Test_handler_DocumentCreate(t *testing.T) {
	const insertSQL = "INSERT INTO rag_documents (id,customer_id,rag_id,name,doc_type,storage_file_id,source_url,status,status_message,tm_create,tm_update) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)"

	tests := []struct {
		name          string
		storageFileID uuid.UUID
		wantStorage   driver.Value
		execErr       error
	}{
		{name: "nil storage file id is stored as NULL", wantStorage: nil},
		{name: "storage file id is passed through", storageFileID: testID4, wantStorage: testID4},
		{name: "exec error", execErr: errors.New("unique violation"), wantStorage: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 2)

			exp := mock.ExpectExec(insertSQL).WithArgs(
				testID1, testID2, testID3, "doc", document.DocTypeURL, tt.wantStorage, "https://example.invalid/a",
				document.StatusPending, "", testNow, testNow,
			)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			d := &document.Document{
				ID: testID1, CustomerID: testID2, RagID: testID3, Name: "doc", DocType: document.DocTypeURL,
				StorageFileID: tt.storageFileID, SourceURL: "https://example.invalid/a", Status: document.StatusPending,
			}
			err := h.DocumentCreate(context.Background(), d)
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_DocumentGet(t *testing.T) {
	const getSQL = "SELECT " + docCols + " FROM rag_documents WHERE id = $1 AND tm_delete IS NULL"

	create := testNow
	proc := testNow.Add(time.Second)

	tests := []struct {
		name string
		row  []driver.Value
		want *document.Document
	}{
		{
			name: "uploaded document with storage file and no url",
			row:  []driver.Value{testID1.String(), testID2.String(), testID3.String(), "f", "uploaded", testID4.String(), nil, "ready", "", create, create, nil, int64(2), proc},
			want: &document.Document{
				ID: testID1, CustomerID: testID2, RagID: testID3, Name: "f", DocType: document.DocTypeUploaded,
				StorageFileID: testID4, Status: document.StatusReady, TMCreate: &create, TMUpdate: &create, RetryCount: 2, TMProcessing: &proc,
			},
		},
		{
			name: "url document with null storage file",
			row:  []driver.Value{testID1.String(), testID2.String(), testID3.String(), "u", "url", nil, "https://example.invalid/x", "pending", "msg", create, create, nil, int64(0), nil},
			want: &document.Document{
				ID: testID1, CustomerID: testID2, RagID: testID3, Name: "u", DocType: document.DocTypeURL,
				SourceURL: "https://example.invalid/x", Status: document.StatusPending, StatusMessage: "msg", TMCreate: &create, TMUpdate: &create,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newTestHandler(t)
			mock.ExpectQuery(getSQL).WithArgs(testID1).WillReturnRows(docRows().AddRow(tt.row...))

			got, err := h.DocumentGet(context.Background(), testID1)
			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.want, got)
			}
			assertMet(t, mock)
		})
	}

	t.Run("no rows becomes ErrNotFound", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(getSQL).WithArgs(testID1).WillReturnError(sql.ErrNoRows)

		got, err := h.DocumentGet(context.Background(), testID1)
		if !errors.Is(err, ErrNotFound) || got != nil {
			t.Errorf("Wrong match. expect ErrNotFound and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("other error is not ErrNotFound", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("connection reset")
		mock.ExpectQuery(getSQL).WithArgs(testID1).WillReturnError(dbErr)

		_, err := h.DocumentGet(context.Background(), testID1)
		if err == nil || errors.Is(err, ErrNotFound) || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped driver error without ErrNotFound, got: %v", err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentList(t *testing.T) {
	const base = "SELECT " + docCols + " FROM rag_documents WHERE tm_create < $1"
	const tail = " ORDER BY tm_create DESC LIMIT "

	tests := []struct {
		name     string
		size     uint64
		token    string
		filters  map[document.Field]any
		wantSQL  string
		wantArgs []driver.Value
		curTime  bool
	}{
		{
			name:     "default token and size hide deleted rows",
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{testCurTime},
			curTime:  true,
		},
		{
			name:     "explicit size",
			size:     3,
			token:    "t",
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "3",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "deleted true",
			token:    "t",
			filters:  map[document.Field]any{"deleted": true},
			wantSQL:  base + " AND tm_delete IS NOT NULL" + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "deleted false",
			token:    "t",
			filters:  map[document.Field]any{"deleted": false},
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "non bool deleted adds no tm_delete condition",
			token:    "t",
			filters:  map[document.Field]any{"deleted": "yes"},
			wantSQL:  base + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "status filter",
			token:    "t",
			filters:  map[document.Field]any{document.FieldStatus: document.StatusReady},
			wantSQL:  base + " AND status = $2 AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{"t", document.StatusReady},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			if tt.curTime {
				util.EXPECT().TimeGetCurTime().Return(testCurTime)
			}

			mock.ExpectQuery(tt.wantSQL).WithArgs(tt.wantArgs...).WillReturnRows(
				docRows().AddRow(testID1.String(), testID2.String(), testID3.String(), "a", "url", nil, "u", "ready", "", testNow, testNow, nil, int64(0), nil),
			)

			got, err := h.DocumentList(context.Background(), tt.size, tt.token, tt.filters)
			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}
			if len(got) != 1 || got[0].ID != testID1 || got[0].SourceURL != "u" {
				t.Errorf("Wrong match. unexpected rows: %+v", got)
			}
			assertMet(t, mock)
		})
	}

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(base + " AND tm_delete IS NULL" + tail + "100").WithArgs("t").WillReturnError(dbErr)

		got, err := h.DocumentList(context.Background(), 0, "t", nil)
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_scanDocumentRows_errors(t *testing.T) {
	const sqlStr = "SELECT " + docCols + " FROM rag_documents WHERE rag_id = $1 AND tm_delete IS NULL"

	t.Run("scan error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(sqlStr).WithArgs(testID1).WillReturnRows(
			docRows().AddRow("not-a-uuid", testID2.String(), testID3.String(), "a", "url", nil, nil, "ready", "", testNow, testNow, nil, int64(0), nil),
		)

		got, err := h.DocumentGetsByRagID(context.Background(), testID1)
		if err == nil || got != nil {
			t.Errorf("Wrong match. expect scan error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("row iteration error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		rowErr := errors.New("stream broken")
		mock.ExpectQuery(sqlStr).WithArgs(testID1).WillReturnRows(
			docRows().AddRow(testID1.String(), testID2.String(), testID3.String(), "a", "url", nil, nil, "ready", "", testNow, testNow, nil, int64(0), nil).RowError(0, rowErr),
		)

		got, err := h.DocumentGetsByRagID(context.Background(), testID1)
		if !errors.Is(err, rowErr) || got != nil {
			t.Errorf("Wrong match. expect row error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentUpdate(t *testing.T) {
	const updateSQL = "UPDATE rag_documents SET status = $1, tm_update = $2 WHERE id = $3 AND tm_delete IS NULL"

	tests := []struct {
		name    string
		execErr error
	}{
		{name: "success"},
		{name: "exec error", execErr: errors.New("deadlock")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(updateSQL).WithArgs(document.StatusReady, testNow, testID1)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := h.DocumentUpdate(context.Background(), testID1, map[document.Field]any{document.FieldStatus: document.StatusReady})
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_DocumentDelete_and_DeleteByRagID(t *testing.T) {
	tests := []struct {
		name    string
		sqlStr  string
		call    func(h *handler) error
		execErr error
	}{
		{
			name:   "delete by id",
			sqlStr: "UPDATE rag_documents SET tm_delete = $1, tm_update = $2 WHERE id = $3 AND tm_delete IS NULL",
			call:   func(h *handler) error { return h.DocumentDelete(context.Background(), testID1) },
		},
		{
			name:    "delete by id exec error",
			sqlStr:  "UPDATE rag_documents SET tm_delete = $1, tm_update = $2 WHERE id = $3 AND tm_delete IS NULL",
			call:    func(h *handler) error { return h.DocumentDelete(context.Background(), testID1) },
			execErr: errors.New("deadlock"),
		},
		{
			name:   "delete by rag id",
			sqlStr: "UPDATE rag_documents SET tm_delete = $1, tm_update = $2 WHERE rag_id = $3 AND tm_delete IS NULL",
			call:   func(h *handler) error { return h.DocumentDeleteByRagID(context.Background(), testID1) },
		},
		{
			name:    "delete by rag id exec error",
			sqlStr:  "UPDATE rag_documents SET tm_delete = $1, tm_update = $2 WHERE rag_id = $3 AND tm_delete IS NULL",
			call:    func(h *handler) error { return h.DocumentDeleteByRagID(context.Background(), testID1) },
			execErr: errors.New("deadlock"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(tt.sqlStr).WithArgs(testNow, testNow, testID1)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := tt.call(h)
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_DocumentClaimForProcessing(t *testing.T) {
	const claimSQL = "UPDATE rag_documents SET retry_count = retry_count + 1, status = $1, tm_processing = $2, tm_update = $3 WHERE id = $4 AND status = $5 AND tm_delete IS NULL RETURNING " + docCols

	t.Run("claims a pending document", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		mock.ExpectQuery(claimSQL).WithArgs(document.StatusProcessing, testNow, testNow, testID1, document.StatusPending).
			WillReturnRows(docRows().AddRow(testID1.String(), testID2.String(), testID3.String(), "n", "url", nil, "u", "processing", "", testNow, testNow, nil, int64(1), testNow))

		got, err := h.DocumentClaimForProcessing(context.Background(), testID1)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if got.Status != document.StatusProcessing || got.RetryCount != 1 || got.TMProcessing == nil {
			t.Errorf("Wrong match. unexpected claimed document: %+v", got)
		}
		assertMet(t, mock)
	})

	t.Run("not pending returns sql.ErrNoRows unchanged", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		mock.ExpectQuery(claimSQL).WithArgs(document.StatusProcessing, testNow, testNow, testID1, document.StatusPending).
			WillReturnRows(docRows())

		got, err := h.DocumentClaimForProcessing(context.Background(), testID1)
		if err != sql.ErrNoRows || got != nil {
			t.Errorf("Wrong match. expect sql.ErrNoRows itself and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentUpdateHeartbeat(t *testing.T) {
	const sqlStr = "UPDATE rag_documents SET tm_processing = $1 WHERE id = $2 AND tm_delete IS NULL"

	tests := []struct {
		name    string
		execErr error
	}{
		{name: "success"},
		{name: "exec error", execErr: errors.New("deadlock")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(sqlStr).WithArgs(testNow, testID1)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := h.DocumentUpdateHeartbeat(context.Background(), testID1)
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_DocumentGetStale_and_GetPending(t *testing.T) {
	const staleSQL = "SELECT " + docCols + " FROM rag_documents WHERE status = $1 AND tm_processing < $2 AND tm_delete IS NULL"
	const pendingSQL = "SELECT " + docCols + " FROM rag_documents WHERE status = $1 AND retry_count < $2 AND tm_delete IS NULL"

	oneRow := func() *sqlmock.Rows {
		return docRows().AddRow(testID1.String(), testID2.String(), testID3.String(), "n", "url", nil, "u", "processing", "", testNow, testNow, nil, int64(1), testNow)
	}

	t.Run("stale uses now minus threshold as cutoff", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		mock.ExpectQuery(staleSQL).WithArgs(document.StatusProcessing, testNow.Add(-5*time.Minute)).WillReturnRows(oneRow())

		got, err := h.DocumentGetStale(context.Background(), 5*time.Minute)
		if err != nil || len(got) != 1 || got[0].ID != testID1 {
			t.Errorf("Wrong match. expect one stale document, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("stale query error", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		dbErr := errors.New("boom")
		mock.ExpectQuery(staleSQL).WithArgs(document.StatusProcessing, testNow.Add(-time.Minute)).WillReturnError(dbErr)

		got, err := h.DocumentGetStale(context.Background(), time.Minute)
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("pending only below the retry limit", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(pendingSQL).WithArgs(document.StatusPending, 3).WillReturnRows(oneRow())

		got, err := h.DocumentGetPending(context.Background())
		if err != nil || len(got) != 1 {
			t.Errorf("Wrong match. expect one pending document, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("pending query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(pendingSQL).WithArgs(document.StatusPending, 3).WillReturnError(dbErr)

		got, err := h.DocumentGetPending(context.Background())
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentResetStaleToPending(t *testing.T) {
	const retrySQL = "UPDATE rag_documents SET status = $1, tm_update = $2 WHERE status = $3 AND tm_processing < $4 AND retry_count < $5 AND tm_delete IS NULL"
	const exhaustedSQL = "UPDATE rag_documents SET status = $1, status_message = $2, tm_update = $3 WHERE status = $4 AND tm_processing < $5 AND retry_count >= $6 AND tm_delete IS NULL"

	cutoff := testNow.Add(-2 * time.Minute)

	t.Run("resets retryable then fails exhausted", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		mock.ExpectExec(retrySQL).WithArgs(document.StatusPending, testNow, document.StatusProcessing, cutoff, 3).WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(exhaustedSQL).WithArgs(document.StatusError, "max retries exceeded (stale processing)", testNow, document.StatusProcessing, cutoff, 3).WillReturnResult(sqlmock.NewResult(0, 1))

		if err := h.DocumentResetStaleToPending(context.Background(), 2*time.Minute); err != nil {
			t.Errorf("Wrong match. expect ok, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("first update failure skips the exhausted update", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		dbErr := errors.New("deadlock")
		mock.ExpectExec(retrySQL).WithArgs(document.StatusPending, testNow, document.StatusProcessing, cutoff, 3).WillReturnError(dbErr)

		err := h.DocumentResetStaleToPending(context.Background(), 2*time.Minute)
		if !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped first update error, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("second update failure is reported", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		dbErr := errors.New("deadlock")
		mock.ExpectExec(retrySQL).WithArgs(document.StatusPending, testNow, document.StatusProcessing, cutoff, 3).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(exhaustedSQL).WithArgs(document.StatusError, "max retries exceeded (stale processing)", testNow, document.StatusProcessing, cutoff, 3).WillReturnError(dbErr)

		err := h.DocumentResetStaleToPending(context.Background(), 2*time.Minute)
		if !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped second update error, got: %v", err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentGetsByRagID(t *testing.T) {
	const sqlStr = "SELECT " + docCols + " FROM rag_documents WHERE rag_id = $1 AND tm_delete IS NULL"

	t.Run("success", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(sqlStr).WithArgs(testID3).WillReturnRows(
			docRows().
				AddRow(testID1.String(), testID2.String(), testID3.String(), "a", "url", nil, "u1", "ready", "", testNow, testNow, nil, int64(0), nil).
				AddRow(testID4.String(), testID2.String(), testID3.String(), "b", "url", nil, "u2", "ready", "", testNow, testNow, nil, int64(0), nil),
		)

		got, err := h.DocumentGetsByRagID(context.Background(), testID3)
		if err != nil || len(got) != 2 || got[0].SourceURL != "u1" || got[1].SourceURL != "u2" {
			t.Errorf("Wrong match. expect two documents in order, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(sqlStr).WithArgs(testID3).WillReturnError(dbErr)

		got, err := h.DocumentGetsByRagID(context.Background(), testID3)
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentGetsByRagIDs(t *testing.T) {
	const sqlStr = "SELECT " + docCols + " FROM rag_documents WHERE rag_id IN ($1,$2) AND tm_delete IS NULL"

	t.Run("empty input issues no query", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)

		got, err := h.DocumentGetsByRagIDs(context.Background(), nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("Wrong match. expect empty non-nil map, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("groups documents by rag id", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(sqlStr).WithArgs(testID3, testID4).WillReturnRows(
			docRows().
				AddRow(testID1.String(), testID2.String(), testID3.String(), "a", "url", nil, "u1", "ready", "", testNow, testNow, nil, int64(0), nil).
				AddRow(testID2.String(), testID2.String(), testID4.String(), "b", "url", nil, "u2", "ready", "", testNow, testNow, nil, int64(0), nil).
				AddRow(testID3.String(), testID2.String(), testID3.String(), "c", "url", nil, "u3", "ready", "", testNow, testNow, nil, int64(0), nil),
		)

		got, err := h.DocumentGetsByRagIDs(context.Background(), []uuid.UUID{testID3, testID4})
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if len(got) != 2 || len(got[testID3]) != 2 || len(got[testID4]) != 1 || got[testID4][0].SourceURL != "u2" {
			t.Errorf("Wrong match. unexpected grouping: %v", got)
		}
		assertMet(t, mock)
	})

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(sqlStr).WithArgs(testID3, testID4).WillReturnError(dbErr)

		got, err := h.DocumentGetsByRagIDs(context.Background(), []uuid.UUID{testID3, testID4})
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("scan error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(sqlStr).WithArgs(testID3, testID4).WillReturnRows(
			docRows().AddRow("bad", testID2.String(), testID3.String(), "a", "url", nil, nil, "ready", "", testNow, testNow, nil, int64(0), nil),
		)

		got, err := h.DocumentGetsByRagIDs(context.Background(), []uuid.UUID{testID3, testID4})
		if err == nil || got != nil {
			t.Errorf("Wrong match. expect scan error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_DocumentGetsByRagIDAndSources(t *testing.T) {
	const base = "SELECT " + docCols + " FROM rag_documents WHERE rag_id = $1 AND tm_delete IS NULL AND ("

	tests := []struct {
		name     string
		fileIDs  []uuid.UUID
		urls     []string
		wantSQL  string
		wantArgs []driver.Value
	}{
		{
			name:     "storage files only",
			fileIDs:  []uuid.UUID{testID2},
			wantSQL:  base + "storage_file_id IN ($2))",
			wantArgs: []driver.Value{testID1, testID2},
		},
		{
			name:     "urls only",
			urls:     []string{"https://example.invalid/a", "https://example.invalid/b"},
			wantSQL:  base + "source_url IN ($2,$3))",
			wantArgs: []driver.Value{testID1, "https://example.invalid/a", "https://example.invalid/b"},
		},
		{
			name:     "both are OR-ed",
			fileIDs:  []uuid.UUID{testID2, testID3},
			urls:     []string{"https://example.invalid/a"},
			wantSQL:  base + "storage_file_id IN ($2,$3) OR source_url IN ($4))",
			wantArgs: []driver.Value{testID1, testID2, testID3, "https://example.invalid/a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newTestHandler(t)
			mock.ExpectQuery(tt.wantSQL).WithArgs(tt.wantArgs...).WillReturnRows(
				docRows().AddRow(testID4.String(), testID2.String(), testID1.String(), "a", "url", nil, "u", "ready", "", testNow, testNow, nil, int64(0), nil),
			)

			got, err := h.DocumentGetsByRagIDAndSources(context.Background(), testID1, tt.fileIDs, tt.urls)
			if err != nil || len(got) != 1 || got[0].ID != testID4 {
				t.Errorf("Wrong match. expect one duplicate, got: %v, %v", got, err)
			}
			assertMet(t, mock)
		})
	}

	t.Run("no sources issues no query", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)

		got, err := h.DocumentGetsByRagIDAndSources(context.Background(), testID1, nil, nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("Wrong match. expect empty non-nil slice, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(base+"source_url IN ($2))").WithArgs(testID1, "u").WillReturnError(dbErr)

		got, err := h.DocumentGetsByRagIDAndSources(context.Background(), testID1, nil, []string{"u"})
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}
