package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/chunk"
)

const (
	chunkInsertSQL = "INSERT INTO rag_chunks (id,document_id,rag_id,customer_id,chunk_index,text,section_title,embedding,token_count,tm_create) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::vector,$9,$10)"
	chunkSearchSQL = `SELECT id, document_id, rag_id, customer_id, chunk_index, text, section_title, token_count, tm_create,
                     embedding <=> $1::vector AS distance
              FROM rag_chunks
              WHERE rag_id = $2 AND tm_delete IS NULL
              ORDER BY embedding <=> $1::vector
              LIMIT $3`
)

func newTestChunk(id uuid.UUID, index int, text string) *chunk.Chunk {
	return &chunk.Chunk{
		ID: id, DocumentID: testID2, RagID: testID3, CustomerID: testID4,
		ChunkIndex: index, Text: text, SectionTitle: "sec", TokenCount: 7,
	}
}

func Test_handler_ChunkCreate(t *testing.T) {
	tests := []struct {
		name    string
		execErr error
	}{
		{name: "success"},
		{name: "exec error", execErr: errors.New("dimension mismatch")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(chunkInsertSQL).WithArgs(testID1, testID2, testID3, testID4, 3, "hello", "sec", "[0.5,-1,0.25]", 7, testNow)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			c := newTestChunk(testID1, 3, "hello")
			err := h.ChunkCreate(context.Background(), c, []float32{0.5, -1, 0.25})
			if tt.execErr == nil && (err != nil || c.TMCreate == nil || !c.TMCreate.Equal(testNow)) {
				t.Errorf("Wrong match. expect ok and TMCreate populated, got: %v, %v", err, c.TMCreate)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_ChunkCreateBatch(t *testing.T) {
	c1 := func() *chunk.Chunk { return newTestChunk(testID1, 0, "first") }
	c2 := func() *chunk.Chunk { return newTestChunk(testID4, 1, "second") }
	embs := [][]float32{{1, 2}, {3, 4}}

	t.Run("length mismatch is rejected before touching the database", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)

		err := h.ChunkCreateBatch(context.Background(), []*chunk.Chunk{c1(), c2()}, embs[:1])
		if err == nil || !strings.Contains(err.Error(), "length mismatch") {
			t.Errorf("Wrong match. expect length mismatch error, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("empty input is a no-op without a transaction", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)

		if err := h.ChunkCreateBatch(context.Background(), nil, nil); err != nil {
			t.Errorf("Wrong match. expect nil, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("begin failure", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("no connection")
		mock.ExpectBegin().WillReturnError(dbErr)

		err := h.ChunkCreateBatch(context.Background(), []*chunk.Chunk{c1()}, embs[:1])
		if !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped begin error, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("inserts every chunk in order and commits", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 2)
		mock.ExpectBegin()
		mock.ExpectExec(chunkInsertSQL).WithArgs(testID1, testID2, testID3, testID4, 0, "first", "sec", "[1,2]", 7, testNow).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(chunkInsertSQL).WithArgs(testID4, testID2, testID3, testID4, 1, "second", "sec", "[3,4]", 7, testNow).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		chunks := []*chunk.Chunk{c1(), c2()}
		if err := h.ChunkCreateBatch(context.Background(), chunks, embs); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		for i, c := range chunks {
			if c.TMCreate == nil || !c.TMCreate.Equal(testNow) {
				t.Errorf("Wrong match. chunk %d TMCreate not populated: %v", i, c.TMCreate)
			}
		}
		assertMet(t, mock)
	})

	t.Run("insert failure rolls back and names the index", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 2)
		dbErr := errors.New("constraint")
		mock.ExpectBegin()
		mock.ExpectExec(chunkInsertSQL).WithArgs(testID1, testID2, testID3, testID4, 0, "first", "sec", "[1,2]", 7, testNow).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(chunkInsertSQL).WithArgs(testID4, testID2, testID3, testID4, 1, "second", "sec", "[3,4]", 7, testNow).WillReturnError(dbErr)
		mock.ExpectRollback()

		err := h.ChunkCreateBatch(context.Background(), []*chunk.Chunk{c1(), c2()}, embs)
		if !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "index 1") {
			t.Errorf("Wrong match. expect wrapped error naming index 1, got: %v", err)
		}
		assertMet(t, mock)
	})

	t.Run("commit failure is reported", func(t *testing.T) {
		h, mock, util := newTestHandler(t)
		expectNow(util, 1)
		dbErr := errors.New("commit failed")
		mock.ExpectBegin()
		mock.ExpectExec(chunkInsertSQL).WithArgs(testID1, testID2, testID3, testID4, 0, "first", "sec", "[1,2]", 7, testNow).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(dbErr)

		err := h.ChunkCreateBatch(context.Background(), []*chunk.Chunk{c1()}, embs[:1])
		if !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped commit error, got: %v", err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_ChunkSearchByRagID(t *testing.T) {
	cols := []string{"id", "document_id", "rag_id", "customer_id", "chunk_index", "text", "section_title", "token_count", "tm_create", "distance"}

	t.Run("score is one minus the cosine distance, best match first", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(chunkSearchSQL).WithArgs("[0.5,0.5]", testID3, 2).WillReturnRows(
			sqlmock.NewRows(cols).
				AddRow(testID1.String(), testID2.String(), testID3.String(), testID4.String(), 0, "near", "s1", 5, testNow, 0.2).
				AddRow(testID4.String(), testID2.String(), testID3.String(), testID4.String(), 1, "far", "s2", 6, testNow, 0.75),
		)

		chunks, scores, err := h.ChunkSearchByRagID(context.Background(), testID3, []float32{0.5, 0.5}, 2)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if len(chunks) != 2 || chunks[0].Text != "near" || chunks[1].Text != "far" || chunks[0].SectionTitle != "s1" || chunks[1].TokenCount != 6 {
			t.Errorf("Wrong match. unexpected chunks: %+v", chunks)
		}
		if len(scores) != 2 || scores[0] != 0.8 || scores[1] != 0.25 {
			t.Errorf("Wrong match. expect scores [0.8 0.25], got: %v", scores)
		}
		assertMet(t, mock)
	})

	t.Run("no match returns empty non-nil slices", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(chunkSearchSQL).WithArgs("[1]", testID3, 5).WillReturnRows(sqlmock.NewRows(cols))

		chunks, scores, err := h.ChunkSearchByRagID(context.Background(), testID3, []float32{1}, 5)
		if err != nil || chunks == nil || scores == nil || len(chunks) != 0 || len(scores) != 0 {
			t.Errorf("Wrong match. expect empty non-nil slices, got: %v, %v, %v", chunks, scores, err)
		}
		assertMet(t, mock)
	})

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(chunkSearchSQL).WithArgs("[1]", testID3, 5).WillReturnError(dbErr)

		chunks, scores, err := h.ChunkSearchByRagID(context.Background(), testID3, []float32{1}, 5)
		if !errors.Is(err, dbErr) || chunks != nil || scores != nil {
			t.Errorf("Wrong match. expect wrapped query error and nils, got: %v, %v, %v", chunks, scores, err)
		}
		assertMet(t, mock)
	})

	t.Run("scan error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(chunkSearchSQL).WithArgs("[1]", testID3, 5).WillReturnRows(
			sqlmock.NewRows(cols).AddRow("bad", testID2.String(), testID3.String(), testID4.String(), 0, "t", "s", 1, testNow, 0.1),
		)

		chunks, scores, err := h.ChunkSearchByRagID(context.Background(), testID3, []float32{1}, 5)
		if err == nil || chunks != nil || scores != nil {
			t.Errorf("Wrong match. expect scan error and nils, got: %v, %v, %v", chunks, scores, err)
		}
		assertMet(t, mock)
	})

	t.Run("row iteration error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		rowErr := errors.New("stream broken")
		mock.ExpectQuery(chunkSearchSQL).WithArgs("[1]", testID3, 5).WillReturnRows(
			sqlmock.NewRows(cols).AddRow(testID1.String(), testID2.String(), testID3.String(), testID4.String(), 0, "t", "s", 1, testNow, 0.1).RowError(0, rowErr),
		)

		chunks, scores, err := h.ChunkSearchByRagID(context.Background(), testID3, []float32{1}, 5)
		if !errors.Is(err, rowErr) || chunks != nil || scores != nil {
			t.Errorf("Wrong match. expect row error and nils, got: %v, %v, %v", chunks, scores, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_Chunk_delete_variants(t *testing.T) {
	tests := []struct {
		name     string
		sqlStr   string
		args     []driver.Value
		clockUse int
		call     func(h *handler) error
	}{
		{
			name:   "hard delete by document",
			sqlStr: "DELETE FROM rag_chunks WHERE document_id = $1",
			args:   []driver.Value{testID1},
			call:   func(h *handler) error { return h.ChunkDeleteByDocumentID(context.Background(), testID1) },
		},
		{
			name:   "hard delete by rag",
			sqlStr: "DELETE FROM rag_chunks WHERE rag_id = $1",
			args:   []driver.Value{testID1},
			call:   func(h *handler) error { return h.ChunkDeleteByRagID(context.Background(), testID1) },
		},
		{
			name:     "soft delete by document",
			sqlStr:   "UPDATE rag_chunks SET tm_delete = $1 WHERE document_id = $2 AND tm_delete IS NULL",
			args:     []driver.Value{testNow, testID1},
			clockUse: 1,
			call:     func(h *handler) error { return h.ChunkSoftDeleteByDocumentID(context.Background(), testID1) },
		},
		{
			name:     "soft delete by rag",
			sqlStr:   "UPDATE rag_chunks SET tm_delete = $1 WHERE rag_id = $2 AND tm_delete IS NULL",
			args:     []driver.Value{testNow, testID1},
			clockUse: 1,
			call:     func(h *handler) error { return h.ChunkSoftDeleteByRagID(context.Background(), testID1) },
		},
	}

	for _, tt := range tests {
		for _, failing := range []bool{false, true} {
			name := tt.name
			if failing {
				name += " exec error"
			}
			t.Run(name, func(t *testing.T) {
				h, mock, util := newTestHandler(t)
				if tt.clockUse > 0 {
					expectNow(util, tt.clockUse)
				}

				exp := mock.ExpectExec(tt.sqlStr).WithArgs(tt.args...)
				dbErr := errors.New("deadlock")
				if failing {
					exp.WillReturnError(dbErr)
				} else {
					exp.WillReturnResult(sqlmock.NewResult(0, 3))
				}

				err := tt.call(h)
				if !failing && err != nil {
					t.Errorf("Wrong match. expect ok, got: %v", err)
				}
				if failing && !errors.Is(err, dbErr) {
					t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
				}
				assertMet(t, mock)
			})
		}
	}
}
