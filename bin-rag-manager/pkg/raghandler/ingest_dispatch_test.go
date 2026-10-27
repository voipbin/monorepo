package raghandler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/pkg/dbhandler"
)

const dispatchWait = 5 * time.Second

var (
	dispatchRagID      = uuid.FromStringOrNil("11111111-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	dispatchCustomerID = uuid.FromStringOrNil("22222222-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	dispatchDocID      = uuid.FromStringOrNil("33333333-cccc-4ccc-8ccc-cccccccccccc")
)

// claimSignal closes its channel on the first claim, so a test can wait for the ingestion
// goroutine it started without sleeping. The claim fails, which ends that goroutine at once.
type claimSignal struct {
	once    sync.Once
	claimed chan uuid.UUID
}

func newClaimSignal() *claimSignal {
	return &claimSignal{claimed: make(chan uuid.UUID, 1)}
}

func (c *claimSignal) claim(_ context.Context, id uuid.UUID) (*document.Document, error) {
	c.once.Do(func() { c.claimed <- id })
	return nil, errors.New("already claimed")
}

func (c *claimSignal) wait(t *testing.T) uuid.UUID {
	t.Helper()

	select {
	case id := <-c.claimed:
		return id
	case <-time.After(dispatchWait):
		t.Fatal("the ingestion goroutine did not claim the document")
		return uuid.Nil
	}
}

func newDispatchHandler(mockDB *dbhandler.MockDBHandler) *ragHandler {
	return &ragHandler{dbHandler: mockDB, ingestSem: make(chan struct{}, 1)}
}

func Test_DocumentIngestPendingAll_dispatchesPending(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newDispatchHandler(mockDB)
	sig := newClaimSignal()

	mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), 2*time.Minute).Return(nil)
	mockDB.EXPECT().DocumentGetPending(gomock.Any()).Return([]*document.Document{{ID: dispatchDocID}}, nil)
	mockDB.EXPECT().DocumentClaimForProcessing(gomock.Any(), dispatchDocID).DoAndReturn(sig.claim)

	h.DocumentIngestPendingAll(context.Background())

	if got := sig.wait(t); got != dispatchDocID {
		t.Errorf("Wrong match. expect: %v, got: %v", dispatchDocID, got)
	}
}

func Test_ingestionTickerRun_dispatchesPending(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newDispatchHandler(mockDB)
	sig := newClaimSignal()

	threshold := 7 * time.Minute
	mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), threshold).Return(nil)
	mockDB.EXPECT().DocumentGetPending(gomock.Any()).Return([]*document.Document{{ID: dispatchDocID}}, nil)
	mockDB.EXPECT().DocumentClaimForProcessing(gomock.Any(), dispatchDocID).DoAndReturn(sig.claim)

	h.ingestionTickerRun(context.Background(), threshold)

	if got := sig.wait(t); got != dispatchDocID {
		t.Errorf("Wrong match. expect: %v, got: %v", dispatchDocID, got)
	}
}

func Test_RunIngestionTicker_dispatchesPendingUntilCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newDispatchHandler(mockDB)
	sig := newClaimSignal()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// a tick can still win the select once more after the cancel, so the lookups may repeat,
	// but only the first lookup returns a document
	var once sync.Once
	mockDB.EXPECT().DocumentResetStaleToPending(gomock.Any(), 10*time.Millisecond).Return(nil).AnyTimes()
	mockDB.EXPECT().DocumentGetPending(gomock.Any()).DoAndReturn(func(context.Context) ([]*document.Document, error) {
		docs := []*document.Document{}
		once.Do(func() {
			docs = []*document.Document{{ID: dispatchDocID}}
			cancel()
		})
		return docs, nil
	}).AnyTimes()
	mockDB.EXPECT().DocumentClaimForProcessing(gomock.Any(), dispatchDocID).DoAndReturn(sig.claim)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.RunIngestionTicker(ctx, 10*time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(dispatchWait):
		t.Fatal("RunIngestionTicker did not stop after the cancel")
	}

	if got := sig.wait(t); got != dispatchDocID {
		t.Errorf("Wrong match. expect: %v, got: %v", dispatchDocID, got)
	}
}

func Test_createDocumentsForSources_dispatchesURLDocument(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newDispatchHandler(mockDB)
	sig := newClaimSignal()

	sourceURL := "https://example.test/guide.md"

	mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), dispatchRagID, []uuid.UUID(nil), []string{sourceURL}).Return([]*document.Document{}, nil)
	mockDB.EXPECT().DocumentCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, d *document.Document) error {
		if d.RagID != dispatchRagID || d.CustomerID != dispatchCustomerID || d.SourceURL != sourceURL ||
			d.DocType != document.DocTypeURL || d.Status != document.StatusPending {
			t.Errorf("Wrong match. unexpected document: %+v", d)
		}
		return nil
	})
	mockDB.EXPECT().DocumentGet(gomock.Any(), gomock.Any()).Return(&document.Document{ID: dispatchDocID, SourceURL: sourceURL}, nil)
	mockDB.EXPECT().DocumentClaimForProcessing(gomock.Any(), dispatchDocID).DoAndReturn(sig.claim)

	if err := h.createDocumentsForSources(context.Background(), dispatchCustomerID, dispatchRagID, nil, []string{sourceURL}); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	if got := sig.wait(t); got != dispatchDocID {
		t.Errorf("Wrong match. expect: %v, got: %v", dispatchDocID, got)
	}
}
