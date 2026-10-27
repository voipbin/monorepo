package server

import (
	"net/http"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_RagsHandlers_Success(t *testing.T) {
	ragID := uuid.FromStringOrNil("22222222-0000-4000-8000-000000000001")
	fileA := uuid.FromStringOrNil("22222222-0000-4000-8000-000000000002")
	fileB := uuid.FromStringOrNil("22222222-0000-4000-8000-000000000003")
	sourceID := uuid.FromStringOrNil("22222222-0000-4000-8000-000000000004")
	tmCreate := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)

	item := &rmrag.WebhookMessage{ID: ragID, Name: "kb", TMCreate: &tmCreate}
	itemJSON := `{"id":"22222222-0000-4000-8000-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","name":"kb","tm_create":"2024-03-04T05:06:07Z"}`
	base := "/rags/22222222-0000-4000-8000-000000000001"

	tests := []okCase{
		{
			name: "create with files and urls", method: http.MethodPost, path: "/rags", wantStatus: 200, wantJSON: itemJSON,
			body: `{"name":"kb","description":"desc","storage_file_ids":["22222222-0000-4000-8000-000000000002","22222222-0000-4000-8000-000000000003"],"source_urls":["https://example.com/a"]}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagCreate(gomock.Any(), a, "kb", "desc", []uuid.UUID{fileA, fileB}, []string{"https://example.com/a"}).Return(item, nil)
			},
		},
		{
			name: "create name only", method: http.MethodPost, path: "/rags", wantStatus: 200, wantJSON: itemJSON,
			body: `{"name":"kb"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagCreate(gomock.Any(), a, "kb", "", []uuid.UUID{}, []string{}).Return(item, nil)
			},
		},
		{
			name: "list", method: http.MethodGet, path: "/rags?page_size=5&page_token=tok", wantStatus: 200,
			wantJSON: `{"result":[` + itemJSON + `],"next_page_token":"2024-03-04T05:06:07.000000Z"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagGets(gomock.Any(), a, uint64(5), "tok").Return([]*rmrag.WebhookMessage{item}, nil)
			},
		},
		{
			name: "list clamps zero page size", method: http.MethodGet, path: "/rags?page_size=0", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagGets(gomock.Any(), a, uint64(100), "").Return(nil, nil)
			},
		},
		{
			name: "get", method: http.MethodGet, path: base, wantStatus: 200, wantJSON: itemJSON,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagGet(gomock.Any(), a, ragID).Return(item, nil)
			},
		},
		{
			name: "update name and description", method: http.MethodPut, path: base, wantStatus: 200, wantJSON: itemJSON,
			body: `{"name":"new","description":"d2"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagUpdate(gomock.Any(), a, ragID, map[rmrag.Field]any{rmrag.FieldName: "new", rmrag.FieldDescription: "d2"}).Return(item, nil)
			},
		},
		{
			name: "update description only", method: http.MethodPut, path: base, wantStatus: 200, wantJSON: itemJSON,
			body: `{"description":"d2"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagUpdate(gomock.Any(), a, ragID, map[rmrag.Field]any{rmrag.FieldDescription: "d2"}).Return(item, nil)
			},
		},
		{
			name: "add sources", method: http.MethodPost, path: base + "/sources", wantStatus: 200, wantJSON: itemJSON,
			body: `{"storage_file_ids":["22222222-0000-4000-8000-000000000002"],"source_urls":["https://example.com/b"]}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagAddSources(gomock.Any(), a, ragID, []uuid.UUID{fileA}, []string{"https://example.com/b"}).Return(item, nil)
			},
		},
		{
			name: "add sources empty body fields", method: http.MethodPost, path: base + "/sources", wantStatus: 200, wantJSON: itemJSON,
			body: `{}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagAddSources(gomock.Any(), a, ragID, []uuid.UUID{}, []string{}).Return(item, nil)
			},
		},
		{
			name: "remove source", method: http.MethodDelete, path: base + "/sources/22222222-0000-4000-8000-000000000004", wantStatus: 200, wantJSON: itemJSON,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagRemoveSource(gomock.Any(), a, ragID, sourceID).Return(item, nil)
			},
		},
		{
			name: "delete", method: http.MethodDelete, path: base, wantStatus: 200, wantJSON: itemJSON,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagDelete(gomock.Any(), a, ragID).Return(item, nil)
			},
		},
	}

	okRunCases(t, tests)
}

func Test_RagsHandlers_Errors(t *testing.T) {
	base := "/rags/22222222-0000-4000-8000-000000000001"
	nilID := "/rags/00000000-0000-0000-0000-000000000000"

	tests := []s2Case{
		{name: "create service error", method: http.MethodPost, path: "/rags", body: `{"name":"kb"}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagCreate(gomock.Any(), a, "kb", "", []uuid.UUID{}, []string{}).Return(nil, s2Err)
			}},
		{name: "list service error", method: http.MethodGet, path: "/rags", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagGets(gomock.Any(), a, uint64(100), "").Return(nil, s2Err)
			}},
		{name: "get service error", method: http.MethodGet, path: base, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagGet(gomock.Any(), a, gomock.Any()).Return(nil, s2Err)
			}},
		{name: "update unauthenticated", method: http.MethodPut, path: base, body: `{"name":"x"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "update bad json", method: http.MethodPut, path: base, body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "update no fields", method: http.MethodPut, path: base, body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT"},
		{name: "update service error", method: http.MethodPut, path: base, body: `{"name":"x"}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagUpdate(gomock.Any(), a, gomock.Any(), map[rmrag.Field]any{rmrag.FieldName: "x"}).Return(nil, s2Err)
			}},
		{name: "add sources unauthenticated", method: http.MethodPost, path: base + "/sources", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "add sources bad json", method: http.MethodPost, path: base + "/sources", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "add sources service error", method: http.MethodPost, path: base + "/sources", body: `{}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagAddSources(gomock.Any(), a, gomock.Any(), []uuid.UUID{}, []string{}).Return(nil, s2Err)
			}},
		{name: "remove source unauthenticated", method: http.MethodDelete, path: base + "/sources/22222222-0000-4000-8000-000000000004", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "remove source service error", method: http.MethodDelete, path: base + "/sources/22222222-0000-4000-8000-000000000004", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagRemoveSource(gomock.Any(), a, gomock.Any(), gomock.Any()).Return(nil, s2Err)
			}},
		{name: "delete service error", method: http.MethodDelete, path: base, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagDelete(gomock.Any(), a, gomock.Any()).Return(nil, s2Err)
			}},
		{name: "delete unauthenticated", method: http.MethodDelete, path: nilID, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
