package listenhandler

// Tests for handler-call error mapping in v1_rags.go and v1_query.go.
// These tests verify that errors from the ragHandler call site are forwarded
// through errorResponse() so that ErrNotFound yields 404 (not 500) and
// generic internal errors yield 500 (not the wrong 404).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/raghandler"
)

// newTestHandler builds a listenHandler backed by the supplied raghandler mock.
func newTestHandler(t *testing.T, rh raghandler.RagHandler) *listenHandler {
	t.Helper()
	return &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  rh,
	}
}

// uriWithID builds a URI like /v1/rags/<uuid>.
func uriWithID(prefix string, id uuid.UUID) string {
	return prefix + id.String()
}

// ---------------------------------------------------------------------------
// processV1RagsIDGet — latent-bug fix: previously returned 404 for ALL errors;
// must now return 404 only for ErrNotFound, 500 for generic errors.
// ---------------------------------------------------------------------------

func TestProcessV1RagsIDGet_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	id := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().RagGet(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	req := &sock.Request{
		URI:    uriWithID("/v1/rags/", id),
		Method: sock.RequestMethodGet,
	}
	resp, err := h.processV1RagsIDGet(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

func TestProcessV1RagsIDGet_GenericError_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	id := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().RagGet(gomock.Any(), id).Return(nil, fmt.Errorf("boom"))

	h := newTestHandler(t, mock)
	req := &sock.Request{
		URI:    uriWithID("/v1/rags/", id),
		Method: sock.RequestMethodGet,
	}
	resp, err := h.processV1RagsIDGet(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 for generic error, got %d (latent-bug fix: was returning 404)", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// processV1RagsIDPut — was returning 500 for all errors; must return 404 for ErrNotFound.
// ---------------------------------------------------------------------------

func TestProcessV1RagsIDPut_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	id := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().RagUpdate(gomock.Any(), id, gomock.Any()).Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)

	name := "updated-name"
	body, _ := json.Marshal(map[string]string{"name": name})
	req := &sock.Request{
		URI:    uriWithID("/v1/rags/", id),
		Method: sock.RequestMethodPut,
		Data:   body,
	}
	resp, err := h.processV1RagsIDPut(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// processV1RagsIDDelete — was returning 500 for all errors; must return 404 for ErrNotFound.
// ---------------------------------------------------------------------------

func TestProcessV1RagsIDDelete_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	id := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().RagDelete(gomock.Any(), id).Return(dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	req := &sock.Request{
		URI:    uriWithID("/v1/rags/", id),
		Method: sock.RequestMethodDelete,
	}
	resp, err := h.processV1RagsIDDelete(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: processV1RagsPost, processV1RagsGet,
// processV1RagsIDSourcesPost, processV1RagsIDSourcesIDDelete, processV1QueryPost.
// All were returning 500 for all errors; must return 404 for ErrNotFound.
// ---------------------------------------------------------------------------

func TestProcessV1RagsPost_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().
		RagCreate(gomock.Any(), customerID, "test", "", gomock.Any(), gomock.Any()).
		Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)

	body, _ := json.Marshal(map[string]any{
		"customer_id":      customerID.String(),
		"name":             "test",
		"storage_file_ids": []string{uuid.Must(uuid.NewV4()).String()},
	})
	req := &sock.Request{
		URI:    "/v1/rags",
		Method: sock.RequestMethodPost,
		Data:   body,
	}
	resp, err := h.processV1RagsPost(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

func TestProcessV1RagsGet_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().
		RagList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	req := &sock.Request{
		URI:    "/v1/rags",
		Method: sock.RequestMethodGet,
		Data:   []byte("{}"),
	}
	resp, err := h.processV1RagsGet(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

func TestProcessV1RagsIDSourcesPost_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ragID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())

	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().
		RagAddSources(gomock.Any(), ragID, gomock.Any(), gomock.Any()).
		Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	body, _ := json.Marshal(map[string]any{
		"storage_file_ids": []string{fileID.String()},
	})
	req := &sock.Request{
		URI:    "/v1/rags/" + ragID.String() + "/sources",
		Method: sock.RequestMethodPost,
		Data:   body,
	}
	resp, err := h.processV1RagsIDSourcesPost(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

func TestProcessV1RagsIDSourcesIDDelete_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ragID := uuid.Must(uuid.NewV4())
	sourceID := uuid.Must(uuid.NewV4())

	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().
		RagRemoveSource(gomock.Any(), ragID, sourceID).
		Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	req := &sock.Request{
		URI:    "/v1/rags/" + ragID.String() + "/sources/" + sourceID.String(),
		Method: sock.RequestMethodDelete,
	}
	resp, err := h.processV1RagsIDSourcesIDDelete(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

func TestProcessV1QueryPost_NotFound_Returns404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ragID := uuid.Must(uuid.NewV4())
	mock := raghandler.NewMockRagHandler(ctrl)
	mock.EXPECT().
		QueryRag(gomock.Any(), ragID, "test query", 0).
		Return(nil, dbhandler.ErrNotFound)

	h := newTestHandler(t, mock)
	body, _ := json.Marshal(map[string]any{
		"rag_id": ragID.String(),
		"query":  "test query",
	})
	req := &sock.Request{
		URI:    "/v1/query",
		Method: sock.RequestMethodPost,
		Data:   body,
	}
	resp, err := h.processV1QueryPost(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for ErrNotFound, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Compile-time check: MockRagHandler satisfies raghandler.RagHandler.
// ---------------------------------------------------------------------------

var _ raghandler.RagHandler = (*raghandler.MockRagHandler)(nil)
