package raghandler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"monorepo/bin-rag-manager/models/chunk"
	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/bucketreader"
	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/dbhandler"
)

var (
	covRagID      = uuid.FromStringOrNil("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	covCustomerID = uuid.FromStringOrNil("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	covDocID      = uuid.FromStringOrNil("cccccccc-cccc-4ccc-8ccc-cccccccccccc")
	covFileID     = uuid.FromStringOrNil("dddddddd-dddd-4ddd-8ddd-dddddddddddd")
)

// fakeEmbedder is a test double for embedder.Embedder. It records every call and can
// fail on a chosen (1-based) EmbedTexts call or on EmbedText.
type fakeEmbedder struct {
	textBatches [][]string
	failOnCall  int
	textErr     error
	queries     []string
	queryErr    error
}

func (f *fakeEmbedder) EmbedTexts(_ context.Context, texts []string) ([][]float32, error) {
	f.textBatches = append(f.textBatches, append([]string(nil), texts...))
	if f.failOnCall != 0 && len(f.textBatches) == f.failOnCall {
		return nil, f.textErr
	}
	res := make([][]float32, len(texts))
	for i := range texts {
		res[i] = []float32{float32(len(f.textBatches)), float32(i)}
	}
	return res, nil
}

func (f *fakeEmbedder) EmbedText(_ context.Context, text string) ([]float32, error) {
	f.queries = append(f.queries, text)
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return []float32{0.1, 0.2}, nil
}

func Test_RagCreate_covers(t *testing.T) {
	t.Run("database create failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		dbErr := errors.New("insert failed")
		mockDB.EXPECT().RagCreate(gomock.Any(), gomock.Any()).Return(dbErr)

		got, err := h.RagCreate(context.Background(), covCustomerID, "n", "d", nil, nil)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped db error, got: %v, %v", got, err)
		}
	})

	t.Run("no sources returns the enriched rag", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		var createdID uuid.UUID
		mockDB.EXPECT().RagCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *rag.Rag) error {
			createdID = r.ID
			if r.CustomerID != covCustomerID || r.Name != "n" || r.Description != "d" || r.ID == uuid.Nil {
				t.Errorf("Wrong match. unexpected rag passed to the db: %+v", r)
			}
			return nil
		})
		mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*document.Document{}, nil)
		mockDB.EXPECT().RagGet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*rag.Rag, error) {
			if id != createdID {
				t.Errorf("Wrong match. RagGet must use the created id, expect: %s, got: %s", createdID, id)
			}
			return &rag.Rag{ID: id, CustomerID: covCustomerID}, nil
		})
		mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), gomock.Any()).Return([]*document.Document{}, nil)

		got, err := h.RagCreate(context.Background(), covCustomerID, "n", "d", nil, nil)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if got.ID != createdID || got.Status != document.StatusPending {
			t.Errorf("Wrong match. expect created id and pending status, got: %+v", got)
		}
	})

	tests := []struct {
		name     string
		fileIDs  []uuid.UUID
		urls     []string
		dupDocs  []*document.Document
		dupErr   error
		deleteEr error
		wantMsg  string
		expectDB bool
	}{
		{name: "duplicate file ids in one request", fileIDs: []uuid.UUID{covFileID, covFileID}, wantMsg: "duplicate source(s) in request: storage_file_id " + covFileID.String()},
		{name: "duplicate urls in one request", urls: []string{"https://example.invalid/a", "https://example.invalid/a"}, wantMsg: "duplicate source(s) in request: source_url https://example.invalid/a"},
		{
			name: "source already exists in the rag", fileIDs: []uuid.UUID{covFileID}, urls: []string{"https://example.invalid/a"}, expectDB: true,
			dupDocs: []*document.Document{{StorageFileID: covFileID}, {SourceURL: "https://example.invalid/a"}},
			wantMsg: "source(s) already exist in this rag: storage_file_id " + covFileID.String() + ", source_url https://example.invalid/a",
		},
		{name: "duplicate lookup failure", urls: []string{"https://example.invalid/a"}, expectDB: true, dupErr: errors.New("lookup failed"), wantMsg: "could not check for duplicate sources: lookup failed"},
		{name: "rollback failure keeps the original error", fileIDs: []uuid.UUID{covFileID, covFileID}, deleteEr: errors.New("rollback failed"), wantMsg: "duplicate source(s) in request"},
	}

	for _, tt := range tests {
		t.Run("source failure rolls back: "+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB}

			var createdID uuid.UUID
			mockDB.EXPECT().RagCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r *rag.Rag) error {
				createdID = r.ID
				return nil
			})
			if tt.expectDB {
				mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), gomock.Any(), tt.fileIDs, tt.urls).Return(tt.dupDocs, tt.dupErr)
			}
			mockDB.EXPECT().RagDelete(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) error {
				if id != createdID {
					t.Errorf("Wrong match. rollback must delete the created rag, expect: %s, got: %s", createdID, id)
				}
				return tt.deleteEr
			})

			got, err := h.RagCreate(context.Background(), covCustomerID, "n", "d", tt.fileIDs, tt.urls)
			if got != nil || err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("Wrong match. expect nil and error containing %q, got: %v, %v", tt.wantMsg, got, err)
			}
		})
	}

	t.Run("document creation failures are skipped without ingestion", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().RagCreate(gomock.Any(), gomock.Any()).Return(nil)
		mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		// the file document fails on insert; the url document is inserted but cannot be read back
		mockDB.EXPECT().DocumentCreate(gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))
		mockDB.EXPECT().DocumentCreate(gomock.Any(), gomock.Any()).Return(nil)
		mockDB.EXPECT().DocumentGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("read back failed"))
		mockDB.EXPECT().RagGet(gomock.Any(), gomock.Any()).Return(&rag.Rag{}, nil)
		mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), gomock.Any()).Return(nil, nil)

		if _, err := h.RagCreate(context.Background(), covCustomerID, "n", "d", []uuid.UUID{covFileID}, []string{"https://example.invalid/a"}); err != nil {
			t.Errorf("Wrong match. per-document failures must not fail the request, got: %v", err)
		}
	})
}

func Test_RagList_covers(t *testing.T) {
	r1 := uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	r2 := uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")

	t.Run("database failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		dbErr := errors.New("boom")
		mockDB.EXPECT().RagList(gomock.Any(), uint64(10), "tok", gomock.Any()).Return(nil, dbErr)

		got, err := h.RagList(context.Background(), 10, "tok", nil)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped error, got: %v, %v", got, err)
		}
	})

	t.Run("empty result skips the document lookup", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().RagList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*rag.Rag{}, nil)

		got, err := h.RagList(context.Background(), 10, "", nil)
		if err != nil || len(got) != 0 {
			t.Errorf("Wrong match. expect empty ok, got: %v, %v", got, err)
		}
	})

	t.Run("document lookup failure still returns the rags", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		rags := []*rag.Rag{{ID: r1, Name: "a"}}
		mockDB.EXPECT().RagList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(rags, nil)
		mockDB.EXPECT().DocumentGetsByRagIDs(gomock.Any(), []uuid.UUID{r1}).Return(nil, errors.New("boom"))

		got, err := h.RagList(context.Background(), 10, "", nil)
		if err != nil || len(got) != 1 || got[0].Name != "a" || got[0].Status != "" || got[0].Sources != nil {
			t.Errorf("Wrong match. expect the unenriched rag and no error, got: %+v, %v", got, err)
		}
	})

	t.Run("status and sources are filled per rag", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		rags := []*rag.Rag{{ID: r1}, {ID: r2}}
		mockDB.EXPECT().RagList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(rags, nil)
		mockDB.EXPECT().DocumentGetsByRagIDs(gomock.Any(), []uuid.UUID{r1, r2}).Return(map[uuid.UUID][]*document.Document{
			r1: {{ID: covDocID, StorageFileID: covFileID, Status: document.StatusReady}, {SourceURL: "https://example.invalid/a", Status: document.StatusPending}},
		}, nil)

		got, err := h.RagList(context.Background(), 10, "", nil)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if got[0].Status != document.StatusProcessing || len(got[0].Sources) != 2 || got[0].Sources[0].StorageFileID == nil || *got[0].Sources[0].StorageFileID != covFileID || got[0].Sources[1].SourceURL != "https://example.invalid/a" {
			t.Errorf("Wrong match. unexpected enrichment for the first rag: %+v", got[0])
		}
		if got[1].Status != document.StatusPending || len(got[1].Sources) != 0 {
			t.Errorf("Wrong match. a rag without documents must be pending with no sources, got: %+v", got[1])
		}
	})
}

func Test_RagUpdate_covers(t *testing.T) {
	fields := map[rag.Field]any{rag.FieldName: "new"}

	tests := []struct {
		name    string
		updErr  error
		getErr  error
		wantErr error
	}{
		{name: "success"},
		{name: "update failure", updErr: errors.New("update failed"), wantErr: errors.New("update failed")},
		{name: "re-read failure", getErr: errors.New("get failed"), wantErr: errors.New("get failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB}

			mockDB.EXPECT().RagUpdate(gomock.Any(), covRagID, fields).Return(tt.updErr)
			if tt.updErr == nil {
				if tt.getErr != nil {
					mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(nil, tt.getErr)
				} else {
					mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(&rag.Rag{ID: covRagID, Name: "new"}, nil)
				}
			}

			got, err := h.RagUpdate(context.Background(), covRagID, fields)
			switch {
			case tt.wantErr != nil:
				if got != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Wrong match. expect nil and error containing %q, got: %v, %v", tt.wantErr, got, err)
				}
			default:
				if err != nil || got.Name != "new" {
					t.Errorf("Wrong match. expect updated rag, got: %v, %v", got, err)
				}
			}
		})
	}
}

func Test_RagAddSources_covers(t *testing.T) {
	t.Run("rag lookup failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		dbErr := errors.New("boom")
		mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(nil, dbErr)

		got, err := h.RagAddSources(context.Background(), covRagID, nil, nil)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped error, got: %v, %v", got, err)
		}
	})

	t.Run("nothing to add returns the rag", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(&rag.Rag{ID: covRagID, CustomerID: covCustomerID}, nil).Times(2)
		mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), covRagID, gomock.Any(), gomock.Any()).Return(nil, nil)
		mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), covRagID).Return([]*document.Document{{Status: document.StatusError}}, nil)

		got, err := h.RagAddSources(context.Background(), covRagID, nil, nil)
		if err != nil || got.Status != document.StatusError {
			t.Errorf("Wrong match. expect enriched rag with error status, got: %+v, %v", got, err)
		}
	})
}

func Test_RagGet_covers(t *testing.T) {
	t.Run("missing rag maps to a not found error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(nil, dbhandler.ErrNotFound)

		_, err := h.RagGet(context.Background(), covRagID)
		var verr *cerrors.VoipbinError
		if !errors.As(err, &verr) || verr.Reason != "RAG_NOT_FOUND" || !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect RAG_NOT_FOUND wrapping ErrNotFound, got: %v", err)
		}
	})

	t.Run("other database failure is not a not found error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(nil, errors.New("boom"))

		_, err := h.RagGet(context.Background(), covRagID)
		var verr *cerrors.VoipbinError
		if err == nil || errors.As(err, &verr) {
			t.Errorf("Wrong match. expect a plain wrapped error, got: %v", err)
		}
	})

	t.Run("document lookup failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		dbErr := errors.New("boom")
		mockDB.EXPECT().RagGet(gomock.Any(), covRagID).Return(&rag.Rag{ID: covRagID}, nil)
		mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), covRagID).Return(nil, dbErr)

		got, err := h.RagGet(context.Background(), covRagID)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped error, got: %v, %v", got, err)
		}
	})
}

func Test_computeRagStatus_covers(t *testing.T) {
	doc := func(s document.Status) *document.Document { return &document.Document{Status: s} }

	tests := []struct {
		name string
		docs []*document.Document
		want document.Status
	}{
		{name: "no documents", want: document.StatusPending},
		{name: "any pending", docs: []*document.Document{doc(document.StatusReady), doc(document.StatusPending)}, want: document.StatusProcessing},
		{name: "any processing beats error", docs: []*document.Document{doc(document.StatusError), doc(document.StatusProcessing)}, want: document.StatusProcessing},
		{name: "ready with an error stays ready", docs: []*document.Document{doc(document.StatusError), doc(document.StatusReady)}, want: document.StatusReady},
		{name: "all error", docs: []*document.Document{doc(document.StatusError), doc(document.StatusError)}, want: document.StatusError},
		{name: "unknown status only", docs: []*document.Document{doc("weird")}, want: document.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeRagStatus(tt.docs); got != tt.want {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.want, got)
			}
		})
	}
}

func Test_QueryRag_covers(t *testing.T) {
	t.Run("non positive top_k falls back to 5", func(t *testing.T) {
		for _, topK := range []int{0, -3} {
			mc := gomock.NewController(t)
			mockDB := dbhandler.NewMockDBHandler(mc)
			emb := &fakeEmbedder{}
			h := &ragHandler{dbHandler: mockDB, embedder: emb}

			mockDB.EXPECT().ChunkSearchByRagID(gomock.Any(), covRagID, []float32{0.1, 0.2}, 5).Return([]*chunk.Chunk{}, []float64{}, nil)

			got, err := h.QueryRag(context.Background(), covRagID, "hello", topK)
			if err != nil || len(got.Sources) != 0 {
				t.Errorf("Wrong match. top_k %d: expect empty sources, got: %+v, %v", topK, got, err)
			}
			if len(emb.queries) != 1 || emb.queries[0] != "hello" {
				t.Errorf("Wrong match. expect the query text to be embedded once, got: %v", emb.queries)
			}
			mc.Finish()
		}
	})

	t.Run("explicit top_k is passed through", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB, embedder: &fakeEmbedder{}}

		mockDB.EXPECT().ChunkSearchByRagID(gomock.Any(), covRagID, gomock.Any(), 9).Return(nil, nil, nil)

		if _, err := h.QueryRag(context.Background(), covRagID, "q", 9); err != nil {
			t.Errorf("Wrong match. expect ok, got: %v", err)
		}
	})

	t.Run("embedding failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		embErr := errors.New("quota")
		h := &ragHandler{dbHandler: dbhandler.NewMockDBHandler(mc), embedder: &fakeEmbedder{queryErr: embErr}}

		got, err := h.QueryRag(context.Background(), covRagID, "q", 3)
		if got != nil || !errors.Is(err, embErr) {
			t.Errorf("Wrong match. expect nil and wrapped embedding error, got: %v, %v", got, err)
		}
	})

	t.Run("search failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB, embedder: &fakeEmbedder{}}

		dbErr := errors.New("boom")
		mockDB.EXPECT().ChunkSearchByRagID(gomock.Any(), covRagID, gomock.Any(), 3).Return(nil, nil, dbErr)

		got, err := h.QueryRag(context.Background(), covRagID, "q", 3)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped search error, got: %v, %v", got, err)
		}
	})

	t.Run("sources keep chunk order and scores, a missing document only blanks its name", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB, embedder: &fakeEmbedder{}}

		docA := uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
		docB := uuid.FromStringOrNil("22222222-2222-4222-8222-222222222222")
		mockDB.EXPECT().ChunkSearchByRagID(gomock.Any(), covRagID, gomock.Any(), 2).Return(
			[]*chunk.Chunk{{DocumentID: docA, SectionTitle: "s1", Text: "t1"}, {DocumentID: docB, SectionTitle: "s2", Text: "t2"}},
			[]float64{0.9, 0.4}, nil)
		mockDB.EXPECT().DocumentGet(gomock.Any(), docA).Return(&document.Document{Name: "doc A"}, nil)
		mockDB.EXPECT().DocumentGet(gomock.Any(), docB).Return(nil, dbhandler.ErrNotFound)

		got, err := h.QueryRag(context.Background(), covRagID, "q", 2)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		want := &query.Response{Sources: []query.Source{
			{DocumentID: docA, DocumentName: "doc A", SectionTitle: "s1", RelevanceScore: 0.9, Text: "t1"},
			{DocumentID: docB, DocumentName: "", SectionTitle: "s2", RelevanceScore: 0.4, Text: "t2"},
		}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", want, got)
		}
	})
}

func Test_DocumentGet_and_DocumentList_covers(t *testing.T) {
	t.Run("get maps ErrNotFound to DOCUMENT_NOT_FOUND", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		mockDB.EXPECT().DocumentGet(gomock.Any(), covDocID).Return(nil, dbhandler.ErrNotFound)

		_, err := h.DocumentGet(context.Background(), covDocID)
		var verr *cerrors.VoipbinError
		if !errors.As(err, &verr) || verr.Reason != "DOCUMENT_NOT_FOUND" || !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect DOCUMENT_NOT_FOUND wrapping ErrNotFound, got: %v", err)
		}
	})

	t.Run("get wraps other failures", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		dbErr := errors.New("boom")
		mockDB.EXPECT().DocumentGet(gomock.Any(), covDocID).Return(nil, dbErr)

		_, err := h.DocumentGet(context.Background(), covDocID)
		var verr *cerrors.VoipbinError
		if !errors.Is(err, dbErr) || errors.As(err, &verr) {
			t.Errorf("Wrong match. expect a plain wrapped error, got: %v", err)
		}
	})

	t.Run("get success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		want := &document.Document{ID: covDocID, Name: "n"}
		mockDB.EXPECT().DocumentGet(gomock.Any(), covDocID).Return(want, nil)

		got, err := h.DocumentGet(context.Background(), covDocID)
		if err != nil || got != want {
			t.Errorf("Wrong match. expect the db document, got: %v, %v", got, err)
		}
	})

	t.Run("list passes paging and filters and wraps failures", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		filters := map[document.Field]any{document.FieldStatus: document.StatusReady}
		want := []*document.Document{{ID: covDocID}}
		mockDB.EXPECT().DocumentList(gomock.Any(), uint64(5), "tok", filters).Return(want, nil)
		got, err := h.DocumentList(context.Background(), 5, "tok", filters)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Wrong match. expect the db documents, got: %v, %v", got, err)
		}

		dbErr := errors.New("boom")
		mockDB.EXPECT().DocumentList(gomock.Any(), uint64(5), "tok", filters).Return(nil, dbErr)
		got, err = h.DocumentList(context.Background(), 5, "tok", filters)
		if got != nil || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect nil and wrapped error, got: %v, %v", got, err)
		}
	})
}

func Test_documentIngestFail_covers(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		want       map[document.Field]any
	}{
		{name: "below the retry limit goes back to pending", retryCount: maxRetryCount - 1, want: map[document.Field]any{document.FieldStatus: document.StatusPending}},
		{name: "at the retry limit becomes a terminal error", retryCount: maxRetryCount, want: map[document.Field]any{document.FieldStatus: document.StatusError, document.FieldStatusMessage: "parse failed"}},
		{name: "above the retry limit is terminal too", retryCount: maxRetryCount + 4, want: map[document.Field]any{document.FieldStatus: document.StatusError, document.FieldStatusMessage: "parse failed"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB}

			mockDB.EXPECT().DocumentUpdate(gomock.Any(), covDocID, tt.want).Return(nil)

			h.documentIngestFail(context.Background(), &document.Document{ID: covDocID, RetryCount: tt.retryCount}, errors.New("parse failed"))
		})
	}
}

func Test_embedTextsWithHeartbeat_covers(t *testing.T) {
	texts := func(n int) []string {
		res := make([]string, n)
		for i := range res {
			res[i] = fmt.Sprintf("t%d", i)
		}
		return res
	}

	tests := []struct {
		name        string
		n           int
		wantBatches []int
	}{
		{name: "empty input embeds nothing", n: 0, wantBatches: nil},
		{name: "exactly one batch", n: heartbeatInterval, wantBatches: []int{heartbeatInterval}},
		{name: "one over the batch size makes a second batch of one", n: heartbeatInterval + 1, wantBatches: []int{heartbeatInterval, 1}},
		{name: "two full batches", n: 2 * heartbeatInterval, wantBatches: []int{heartbeatInterval, heartbeatInterval}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			emb := &fakeEmbedder{}
			h := &ragHandler{dbHandler: mockDB, embedder: emb}

			mockDB.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(nil).Times(len(tt.wantBatches))

			got, err := h.embedTextsWithHeartbeat(context.Background(), covDocID, texts(tt.n))
			if err != nil || len(got) != tt.n {
				t.Fatalf("Wrong match. expect %d embeddings, got: %d, %v", tt.n, len(got), err)
			}
			var sizes []int
			for _, b := range emb.textBatches {
				sizes = append(sizes, len(b))
			}
			if !reflect.DeepEqual(sizes, tt.wantBatches) {
				t.Errorf("Wrong match. expect batch sizes %v, got: %v", tt.wantBatches, sizes)
			}
		})
	}

	t.Run("embedding failure names the failing range and stops", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		embErr := errors.New("quota")
		emb := &fakeEmbedder{failOnCall: 2, textErr: embErr}
		h := &ragHandler{dbHandler: mockDB, embedder: emb}

		mockDB.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(nil).Times(1)

		got, err := h.embedTextsWithHeartbeat(context.Background(), covDocID, texts(3*heartbeatInterval))
		if got != nil || !errors.Is(err, embErr) || !strings.Contains(err.Error(), "chunks 10-19") {
			t.Errorf("Wrong match. expect nil and error naming chunks 10-19, got: %v, %v", got, err)
		}
		if len(emb.textBatches) != 2 {
			t.Errorf("Wrong match. embedding must stop after the failing batch, calls: %d", len(emb.textBatches))
		}
	})

	t.Run("heartbeat failures are ignored", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB, embedder: &fakeEmbedder{}}

		mockDB.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(errors.New("boom")).Times(2)

		got, err := h.embedTextsWithHeartbeat(context.Background(), covDocID, texts(heartbeatInterval+1))
		if err != nil || len(got) != heartbeatInterval+1 {
			t.Errorf("Wrong match. expect all embeddings despite heartbeat errors, got: %d, %v", len(got), err)
		}
	})
}

func Test_buildDBChunks_covers(t *testing.T) {
	h := &ragHandler{}
	doc := &document.Document{ID: covDocID, RagID: covRagID, CustomerID: covCustomerID}

	got, err := h.buildDBChunks(doc, []chunker.Chunk{
		{Text: "12345678", SectionTitle: "first"},
		{Text: "abc", SectionTitle: "second"},
	})
	if err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Wrong match. expect 2 chunks, got: %d", len(got))
	}
	if got[0].ID == uuid.Nil || got[1].ID == uuid.Nil || got[0].ID == got[1].ID {
		t.Errorf("Wrong match. expect distinct generated ids, got: %s, %s", got[0].ID, got[1].ID)
	}
	for i, c := range got {
		if c.ChunkIndex != i || c.DocumentID != covDocID || c.RagID != covRagID || c.CustomerID != covCustomerID {
			t.Errorf("Wrong match. chunk %d has wrong linkage: %+v", i, c)
		}
	}
	if got[0].Text != "12345678" || got[0].SectionTitle != "first" || got[0].TokenCount != 2 || got[1].TokenCount != 0 {
		t.Errorf("Wrong match. unexpected text, title or token estimate (len/4): %+v %+v", got[0], got[1])
	}

	empty, err := h.buildDBChunks(doc, nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("Wrong match. expect empty non-nil slice, got: %v, %v", empty, err)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	return p
}

type ingestFixture struct {
	h        *ragHandler
	db       *dbhandler.MockDBHandler
	req      *requesthandler.MockRequestHandler
	bucket   *bucketreader.MockBucketReader
	embedder *fakeEmbedder
}

func newIngestFixture(t *testing.T) *ingestFixture {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	f := &ingestFixture{
		db:       dbhandler.NewMockDBHandler(mc),
		req:      requesthandler.NewMockRequestHandler(mc),
		bucket:   bucketreader.NewMockBucketReader(mc),
		embedder: &fakeEmbedder{},
	}
	f.h = &ragHandler{
		embedder:     f.embedder,
		dbHandler:    f.db,
		reqHandler:   f.req,
		bucketReader: f.bucket,
		bucketName:   "default-bucket",
		ingestSem:    make(chan struct{}, 1),
	}
	return f
}

func Test_documentIngest_covers(t *testing.T) {
	doc := &document.Document{ID: covDocID, RagID: covRagID, CustomerID: covCustomerID, StorageFileID: covFileID}
	claimed := func(retry int) *document.Document {
		c := *doc
		c.RetryCount = retry
		return &c
	}
	pending := map[document.Field]any{document.FieldStatus: document.StatusPending}
	ready := map[document.Field]any{document.FieldStatus: document.StatusReady}

	t.Run("claim failure ends the run without side effects", func(t *testing.T) {
		f := newIngestFixture(t)
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(nil, errors.New("already claimed"))

		f.h.documentIngest(doc)

		if len(f.h.ingestSem) != 0 {
			t.Errorf("Wrong match. the semaphore slot must be released, in use: %d", len(f.h.ingestSem))
		}
	})

	t.Run("document without a source goes back to pending", func(t *testing.T) {
		f := newIngestFixture(t)
		noSource := &document.Document{ID: covDocID}
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(noSource, nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(noSource)
	})

	t.Run("unsupported url scheme is rejected before any network access", func(t *testing.T) {
		f := newIngestFixture(t)
		ftpDoc := &document.Document{ID: covDocID, SourceURL: "ftp://example.invalid/file.txt", RetryCount: maxRetryCount}
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(ftpDoc, nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[document.Field]any) error {
			msg, _ := fields[document.FieldStatusMessage].(string)
			if fields[document.FieldStatus] != document.StatusError || !strings.Contains(msg, "could not acquire file") {
				t.Errorf("Wrong match. expect terminal error mentioning the acquire failure, got: %v", fields)
			}
			return nil
		})

		f.h.documentIngest(ftpDoc)
	})

	t.Run("storage metadata failure", func(t *testing.T) {
		f := newIngestFixture(t)
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(nil, errors.New("storage down"))
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("file larger than the limit is rejected before download", func(t *testing.T) {
		f := newIngestFixture(t)
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{Filesize: maxFileSize + 1}, nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("file exactly at the limit is downloaded", func(t *testing.T) {
		f := newIngestFixture(t)
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{Filesize: maxFileSize, BucketName: "b", Filepath: "p", Filename: "a.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return("", errors.New("gcs down"))
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("download failure", func(t *testing.T) {
		f := newIngestFixture(t)
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(maxRetryCount), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "a.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return("", errors.New("gcs down"))
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[document.Field]any) error {
			if fields[document.FieldStatus] != document.StatusError || !strings.Contains(fields[document.FieldStatusMessage].(string), "could not download from GCS") {
				t.Errorf("Wrong match. expect terminal error mentioning the download, got: %v", fields)
			}
			return nil
		})

		f.h.documentIngest(doc)
	})

	t.Run("unreadable file fails chunking", func(t *testing.T) {
		f := newIngestFixture(t)
		missing := filepath.Join(t.TempDir(), "missing.txt")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "a.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return(missing, nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("blank file produces no chunks and never reaches the embedder", func(t *testing.T) {
		f := newIngestFixture(t)
		p := writeTempFile(t, "blank.txt", " \n\t ")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(maxRetryCount), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "blank.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return(p, nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, map[document.Field]any{
			document.FieldStatus:        document.StatusError,
			document.FieldStatusMessage: "file produced no chunks",
		}).Return(nil)

		f.h.documentIngest(doc)

		if len(f.embedder.textBatches) != 0 {
			t.Errorf("Wrong match. the embedder must not be called for an empty document, calls: %d", len(f.embedder.textBatches))
		}
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("Wrong match. the temp file must be removed after ingestion, stat err: %v", err)
		}
	})

	t.Run("embedding failure", func(t *testing.T) {
		f := newIngestFixture(t)
		f.embedder.failOnCall = 1
		f.embedder.textErr = errors.New("quota")
		p := writeTempFile(t, "a.txt", "some content to embed")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "a.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return(p, nil)
		f.db.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("chunk storage failure", func(t *testing.T) {
		f := newIngestFixture(t)
		p := writeTempFile(t, "a.txt", "some content to embed")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "a.txt"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return(p, nil)
		f.db.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(nil).Times(2)
		f.db.EXPECT().ChunkCreateBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, pending).Return(nil)

		f.h.documentIngest(doc)
	})

	t.Run("success stores linked chunks with their embeddings and marks the document ready", func(t *testing.T) {
		f := newIngestFixture(t)
		p := writeTempFile(t, "a.txt", "some content to embed")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{Filepath: "p", Filename: "a.txt"}, nil)
		// an empty bucket name on the file falls back to the handler default
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "default-bucket", "p").Return(p, nil)
		// a heartbeat failure must not abort the pipeline
		f.db.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(errors.New("boom")).Times(2)
		f.db.EXPECT().ChunkCreateBatch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, chunks []*chunk.Chunk, embs [][]float32) error {
			if len(chunks) != 1 || len(embs) != 1 {
				t.Fatalf("Wrong match. expect one chunk with one embedding, got: %d, %d", len(chunks), len(embs))
			}
			c := chunks[0]
			if c.DocumentID != covDocID || c.RagID != covRagID || c.CustomerID != covCustomerID || c.Text != "some content to embed" || c.ChunkIndex != 0 {
				t.Errorf("Wrong match. unexpected stored chunk: %+v", c)
			}
			return nil
		})
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, ready).Return(nil)

		f.h.documentIngest(doc)

		if len(f.embedder.textBatches) != 1 || len(f.embedder.textBatches[0]) != 1 {
			t.Errorf("Wrong match. expect a single embedding batch with one text, got: %v", f.embedder.textBatches)
		}
	})

	t.Run("a failing ready update is tolerated", func(t *testing.T) {
		f := newIngestFixture(t)
		p := writeTempFile(t, "noext", "content without any file extension")
		f.db.EXPECT().DocumentClaimForProcessing(gomock.Any(), covDocID).Return(claimed(0), nil)
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), covFileID).Return(&smfile.File{BucketName: "b", Filepath: "p", Filename: "noext"}, nil)
		f.bucket.EXPECT().DownloadToTempFile(gomock.Any(), "b", "p").Return(p, nil)
		f.db.EXPECT().DocumentUpdateHeartbeat(gomock.Any(), covDocID).Return(nil).Times(2)
		f.db.EXPECT().ChunkCreateBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		f.db.EXPECT().DocumentUpdate(gomock.Any(), covDocID, ready).Return(errors.New("boom"))

		f.h.documentIngest(doc)
	})
}

func Test_DocumentIngestPendingAll_and_ingestionTickerRun_covers(t *testing.T) {
	type call struct {
		name string
		run  func(h *ragHandler)
	}
	calls := []call{
		{name: "startup sweep", run: func(h *ragHandler) { h.DocumentIngestPendingAll(context.Background()) }},
		{name: "ticker run", run: func(h *ragHandler) { h.ingestionTickerRun(context.Background(), 2*time.Minute) }},
	}

	for _, c := range calls {
		t.Run(c.name+": reset failure does not stop the pending lookup", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB, ingestSem: make(chan struct{}, 1)}

			mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), 2*time.Minute).Return(errors.New("boom"))
			mockDB.EXPECT().DocumentGetPending(gomock.Any()).Return([]*document.Document{}, nil)

			c.run(h)
		})

		t.Run(c.name+": pending lookup failure ends the run", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB, ingestSem: make(chan struct{}, 1)}

			mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), 2*time.Minute).Return(nil)
			mockDB.EXPECT().DocumentGetPending(gomock.Any()).Return(nil, errors.New("boom"))

			c.run(h)
		})
	}
}

func Test_RunIngestionTicker_covers(t *testing.T) {
	t.Run("a cancelled context stops the loop before any tick", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h := &ragHandler{dbHandler: dbhandler.NewMockDBHandler(mc)}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan struct{})
		go func() {
			h.RunIngestionTicker(ctx, time.Hour)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("RunIngestionTicker did not return after the context was cancelled")
		}
	})

	t.Run("a tick runs the sweep with the interval as the stale threshold, then cancel stops the loop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &ragHandler{dbHandler: mockDB}

		const interval = 100 * time.Millisecond
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), interval).DoAndReturn(func(context.Context, time.Duration) error {
			cancel()
			return nil
		}).Times(1)
		mockDB.EXPECT().DocumentGetPending(gomock.Any()).Return([]*document.Document{}, nil).Times(1)

		done := make(chan struct{})
		go func() {
			h.RunIngestionTicker(ctx, interval)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("RunIngestionTicker did not return after the context was cancelled")
		}
	})
}
